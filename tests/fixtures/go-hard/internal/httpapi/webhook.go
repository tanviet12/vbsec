package httpapi

import (
	"bytes"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const allowedHookPrefix = "https://hooks.example.com"

var hookClient = &http.Client{Timeout: 5 * time.Second}

func (a *API) TestWebhook(c *gin.Context) {
	var req struct {
		URL string `json:"url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !strings.HasPrefix(req.URL, allowedHookPrefix) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook url"})
		return
	}
	resp, err := hookClient.Post(req.URL, "application/json", bytes.NewBufferString(`{"event":"ping"}`))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "delivery failed"})
		return
	}
	defer resp.Body.Close()
	c.JSON(http.StatusOK, gin.H{"status": resp.StatusCode})
}
