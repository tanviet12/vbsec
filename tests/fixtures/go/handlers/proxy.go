package handlers

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Proxy(c *gin.Context) {
	target := c.Query("url")
	resp, err := http.Get(target)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "fetch failed"})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}
