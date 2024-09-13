package openai

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"opencatd-open/pkg/tokenizer"
	"opencatd-open/store"
	"path/filepath"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/wav"
	"github.com/gin-gonic/gin"
	"gopkg.in/vansante/go-ffprobe.v2"
)

func WhisperProxy(c *gin.Context) {
	var chatlog store.Tokens

	byteBody, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(byteBody))

	model, _ := c.GetPostForm("model")

	key, err := store.SelectKeyCache("openai")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": err.Error(),
			},
		})
		return
	}

	chatlog.Model = model

	token, _ := c.Get("localuser")

	lu, err := store.GetUserByToken(token.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": err.Error(),
			},
		})
		return
	}
	chatlog.UserID = int(lu.ID)

	if err := ParseWhisperRequestTokens(c, &chatlog, byteBody); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": err.Error(),
			},
		})
		return
	}
	if key.EndPoint == "" {
		key.EndPoint = "https://api.openai.com"
	}
	targetUrl, _ := url.ParseRequestURI(key.EndPoint + c.Request.URL.String())
	log.Println(targetUrl)
	proxy := httputil.NewSingleHostReverseProxy(targetUrl)
	proxy.Director = func(req *http.Request) {
		req.Host = targetUrl.Host
		req.URL.Scheme = targetUrl.Scheme
		req.URL.Host = targetUrl.Host

		req.Header.Set("Authorization", "Bearer "+key.Key)
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.StatusCode != http.StatusOK {
			return nil
		}
		chatlog.TotalTokens = chatlog.PromptCount + chatlog.CompletionCount
		chatlog.Cost = fmt.Sprintf("%.6f", tokenizer.Cost(chatlog.Model, chatlog.PromptCount, chatlog.CompletionCount))
		if err := store.Record(&chatlog); err != nil {
			log.Println(err)
		}
		if err := store.SumDaily(chatlog.UserID); err != nil {
			log.Println(err)
		}
		return nil
	}
	proxy.ServeHTTP(c.Writer, c.Request)
}

func probe(fileReader io.Reader) (time.Duration, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()

	data, err := ffprobe.ProbeReader(ctx, fileReader)
	if err != nil {
		return 0, err
	}

	duration := data.Format.DurationSeconds
	pduration, err := time.ParseDuration(fmt.Sprintf("%fs", duration))
	if err != nil {
		return 0, fmt.Errorf("Error parsing duration: %s", err)
	}
	return pduration, nil
}

func getAudioDuration(file *multipart.FileHeader) (time.Duration, error) {
	var (
		streamer beep.StreamSeekCloser
		format   beep.Format
		err      error
	)

	f, err := file.Open()
	defer f.Close()

	// Get the file extension to determine the audio file type
	fileType := filepath.Ext(file.Filename)

	switch fileType {
	case ".mp3":
		streamer, format, err = mp3.Decode(f)
	case ".wav":
		streamer, format, err = wav.Decode(f)
	case ".m4a":
		duration, err := probe(f)
		if err != nil {
			return 0, err
		}
		return duration, nil
	default:
		return 0, errors.New("unsupported audio file format")
	}

	if err != nil {
		return 0, err
	}
	defer streamer.Close()

	// Calculate the audio file's duration.
	numSamples := streamer.Len()
	sampleRate := format.SampleRate
	duration := time.Duration(numSamples) * time.Second / time.Duration(sampleRate)

	return duration, nil
}

func ParseWhisperRequestTokens(c *gin.Context, usage *store.Tokens, byteBody []byte) error {
	file, _ := c.FormFile("file")
	model, _ := c.GetPostForm("model")
	usage.Model = model

	if file != nil {
		duration, err := getAudioDuration(file)
		if err != nil {
			return fmt.Errorf("Error getting audio duration:%s", err)
		}

		if duration > 5*time.Minute {
			return fmt.Errorf("Audio duration exceeds 5 minutes")
		}
		// 计算时长，四舍五入到最接近的秒数
		usage.PromptCount = int(duration.Round(time.Second).Seconds())
	}

	c.Request.Body = io.NopCloser(bytes.NewBuffer(byteBody))

	return nil
}
