package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (a *API) SearchProducts(c *gin.Context) {
	items, err := a.catalog.Search(c.Query("q"), c.Query("sort"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (a *API) ProductsByTag(c *gin.Context) {
	items, err := a.catalog.ByTag(c.Query("tag"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup failed"})
		return
	}
	c.JSON(http.StatusOK, items)
}
