package router

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"opencatd-open/store"
	"time"

	"github.com/Sakurasan/to"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	rootToken string
	baseUrl   = "https://api.openai.com"
)

type User struct {
	IsDelete  bool   `json:"IsDelete,omitempty"`
	ID        int    `json:"id,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	Name      string `json:"name,omitempty"`
	Token     string `json:"token,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

type Key struct {
	ID        int    `json:"id,omitempty"`
	Key       string `json:"key,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	Name      string `json:"name,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if rootToken == "" {
			u, err := store.GetUserByID(uint(1))
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
				c.Abort()
				return
			}
			rootToken = u.Token
		}
		token := c.GetHeader("Authorization")
		if token == "" || token[:7] != "Bearer " || token[7:] != rootToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		// 可以在这里对 token 进行验证并检查权限

		c.Next()
	}
}

func Handleinit(c *gin.Context) {
	user, err := store.GetUserByID(1)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			u := store.User{Name: "root", Token: uuid.NewString()}
			u.ID = 1
			if err := store.CreateUser(&u); err != nil {
				c.JSON(http.StatusOK, gin.H{
					"error": err.Error(),
				})
				return
			} else {
				rootToken = u.Token
				resJSON := User{
					false,
					int(u.ID),
					u.UpdatedAt.Format(time.RFC3339),
					u.Name,
					u.Token,
					u.CreatedAt.Format(time.RFC3339),
				}
				c.JSON(http.StatusOK, resJSON)
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}
	if user.ID == uint(1) {
		c.JSON(http.StatusOK, gin.H{
			"error": "super user already exists, use cli to reset password",
		})
	}
}

func HandleMe(c *gin.Context) {
	u, err := store.GetUserByID(1)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
	}

	resJSON := User{
		false,
		int(u.ID),
		u.UpdatedAt.Format(time.RFC3339),
		u.Name,
		u.Token,
		u.CreatedAt.Format(time.RFC3339),
	}
	c.JSON(http.StatusOK, resJSON)
}

func HandleKeys(c *gin.Context) {
	keys, err := store.GetAllKeys()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
	}

	c.JSON(http.StatusOK, keys)
}

func HandleUsers(c *gin.Context) {
	users, err := store.GetAllUsers()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
	}

	c.JSON(http.StatusOK, users)
}

func HandleAddKey(c *gin.Context) {
	var body Key
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	if err := store.AddKey(body.Key, body.Name); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	k, err := store.GetKeyrByName(body.Name)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, k)
}

func HandleDelKey(c *gin.Context) {
	id := to.Int(c.Param("id"))
	if id < 1 {
		c.JSON(http.StatusOK, gin.H{"error": "invalid key id"})
		return
	}
	if err := store.DeleteKey(uint(id)); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": "invalid key id"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func HandleAddUser(c *gin.Context) {
	var body User
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	// if len(body.Name) == 0 {
	// 	c.JSON(http.StatusOK, gin.H{"error": "invalid user name"})
	// 	return
	// }

	if err := store.AddUser(body.Name, uuid.NewString()); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	u, err := store.GetUserByName(body.Name)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, u)
}

func HandleDelUser(c *gin.Context) {
	id := to.Int(c.Param("id"))
	if id <= 1 {
		c.JSON(http.StatusOK, gin.H{"error": "invalid user id"})
		return
	}
	if err := store.DeleteUser(uint(id)); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func HandleResetUserToken(c *gin.Context) {
	id := to.Int(c.Param("id"))

	if err := store.UpdateUser(uint(id), uuid.NewString()); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	u, err := store.GetUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, u)
}

func GenerateToken() string {
	token := uuid.New()
	return token.String()
}

func HandleProy(c *gin.Context) {
	var localuser bool
	auth := c.Request.Header.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		localuser = store.IsExistAuthCache(auth[7:])
	}
	client := http.DefaultClient
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	client.Transport = tr

	// 创建 API 请求
	req, err := http.NewRequest(c.Request.Method, baseUrl+c.Request.URL.Path, c.Request.Body)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	req.Header = c.Request.Header
	if localuser {
		if store.KeysCache.ItemCount() == 0 {
			c.JSON(http.StatusOK, gin.H{"error": "No Api-Key Available"})
			return
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", store.FromKeyCacheRandomItem()))
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	// 复制 API 响应头部
	for name, values := range resp.Header {
		for _, value := range values {
			c.Writer.Header().Add(name, value)
		}
	}
	head := map[string]string{
		"Cache-Control":                    "no-store",
		"access-control-allow-origin":      "*",
		"access-control-allow-credentials": "true",
	}
	for k, v := range head {
		if _, ok := resp.Header[k]; !ok {
			c.Writer.Header().Set(k, v)
		}
	}
	resp.Header.Del("content-security-policy")
	resp.Header.Del("content-security-policy-report-only")
	resp.Header.Del("clear-site-data")

	bodyRes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if resp.StatusCode == 200 {
		// todo
	}
	resbody := io.NopCloser(bytes.NewReader(bodyRes))
	// 返回 API 响应主体
	c.Writer.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(c.Writer, resbody); err != nil {
		log.Println(err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
}

func HandleReverseProxy(c *gin.Context) {
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "https"
			req.URL.Host = "api.openai.com"
			// req.Header.Set("Authorization", "Bearer YOUR_API_KEY_HERE")
		},
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		},
	}

	var localuser bool
	auth := c.Request.Header.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		log.Println(store.IsExistAuthCache(auth[7:]))
		localuser = store.IsExistAuthCache(auth[7:])
	}

	req, err := http.NewRequest(c.Request.Method, c.Request.URL.Path, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	req.Header = c.Request.Header
	if localuser {
		if store.KeysCache.ItemCount() == 0 {
			c.JSON(http.StatusOK, gin.H{"error": "No Api-Key Available"})
			return
		}
		// c.Request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", store.FromKeyCacheRandomItem()))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", store.FromKeyCacheRandomItem()))
	}

	proxy.ServeHTTP(c.Writer, req)

}
