package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"example.com/wallet/internal/repo"
	"example.com/wallet/internal/service"
)

type API struct {
	catalog *service.Catalog
	wallet  *service.Wallet
	orders  *repo.OrderRepo
}

func New(c *service.Catalog, w *service.Wallet, o *repo.OrderRepo) *API {
	return &API{catalog: c, wallet: w, orders: o}
}

func (a *API) Routes(r *gin.Engine) {
	r.GET("/products", a.SearchProducts)
	r.GET("/products/by-tag", a.ProductsByTag)
	r.GET("/files/*name", a.DownloadFile)
	r.GET("/avatars/:name", a.Avatar)
	r.POST("/webhooks/test", a.TestWebhook)

	auth := r.Group("/", a.RequireUser)
	auth.GET("/orders/:id", a.GetOrder)
	auth.PUT("/orders/:id/address", a.UpdateOrderAddress)
	auth.POST("/wallet/withdraw", a.Withdraw)
	auth.GET("/orders/:id/invoice.pdf", a.Invoice)
}

func (a *API) RequireUser(c *gin.Context) {
	uid, ok := sessionUserID(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	c.Set("userID", uid)
	c.Next()
}
