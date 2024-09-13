package team

import (
	"net/http"
	"opencatd-open/pkg/azureopenai"
	"opencatd-open/store"
	"strings"

	"github.com/Sakurasan/to"
	"github.com/gin-gonic/gin"
)

type Key struct {
	ID        int    `json:"id,omitempty"`
	Key       string `json:"key,omitempty"`
	Name      string `json:"name,omitempty"`
	ApiType   string `json:"api_type,omitempty"`
	Endpoint  string `json:"endpoint,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
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

func HandleAddKey(c *gin.Context) {
	var body Key
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"message": err.Error(),
		}})
		return
	}
	body.Name = strings.ToLower(strings.TrimSpace(body.Name))
	body.Key = strings.TrimSpace(body.Key)
	if strings.HasPrefix(body.Name, "azure.") {
		keynames := strings.Split(body.Name, ".")
		if len(keynames) < 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
				"message": "Invalid Key Name",
			}})
			return
		}
		k := &store.Key{
			ApiType:      "azure",
			Name:         body.Name,
			Key:          body.Key,
			ResourceNmae: keynames[1],
			EndPoint:     body.Endpoint,
		}
		if err := store.CreateKey(k); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
				"message": err.Error(),
			}})
			return
		}
	} else if strings.HasPrefix(body.Name, "claude.") {
		keynames := strings.Split(body.Name, ".")
		if len(keynames) < 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
				"message": "Invalid Key Name",
			}})
			return
		}
		if body.Endpoint == "" {
			body.Endpoint = "https://api.anthropic.com"
		}
		k := &store.Key{
			// ApiType:      "anthropic",
			ApiType:      "claude",
			Name:         body.Name,
			Key:          body.Key,
			ResourceNmae: keynames[1],
			EndPoint:     body.Endpoint,
		}
		if err := store.CreateKey(k); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
				"message": err.Error(),
			}})
			return
		}
	} else if strings.HasPrefix(body.Name, "google.") {
		keynames := strings.Split(body.Name, ".")
		if len(keynames) < 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
				"message": "Invalid Key Name",
			}})
			return
		}

		k := &store.Key{
			// ApiType:      "anthropic",
			ApiType:      "google",
			Name:         body.Name,
			Key:          body.Key,
			ResourceNmae: keynames[1],
			EndPoint:     body.Endpoint,
		}
		if err := store.CreateKey(k); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
				"message": err.Error(),
			}})
			return
		}
	} else if strings.HasPrefix(body.Name, "github.") {
		keynames := strings.Split(body.Name, ".")
		if len(keynames) < 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
				"message": "Invalid Key Name",
			}})
			return
		}

		k := &store.Key{
			ApiType:      "github",
			Name:         body.Name,
			Key:          body.Key,
			ResourceNmae: keynames[1],
			EndPoint:     body.Endpoint,
		}
		if err := store.CreateKey(k); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
				"message": err.Error(),
			}})
			return
		}
	} else {
		if body.ApiType == "" {
			if err := store.AddKey("openai", body.Key, body.Name); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
					"message": err.Error(),
				}})
				return
			}
		} else {
			k := &store.Key{
				ApiType:      body.ApiType,
				Name:         body.Name,
				Key:          body.Key,
				ResourceNmae: azureopenai.GetResourceName(body.Endpoint),
				EndPoint:     body.Endpoint,
			}
			if err := store.CreateKey(k); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
					"message": err.Error(),
				}})
				return
			}
		}

	}

	k, err := store.GetKeyrByName(body.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"message": err.Error(),
		}})
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
