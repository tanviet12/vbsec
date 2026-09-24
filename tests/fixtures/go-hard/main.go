package main

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/go-sql-driver/mysql"

	"example.com/wallet/internal/httpapi"
	"example.com/wallet/internal/repo"
	"example.com/wallet/internal/service"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	db := sqlx.MustConnect("mysql", os.Getenv("DATABASE_DSN"))

	products := repo.NewProductRepo(db)
	api := httpapi.New(
		service.NewCatalog(products),
		service.NewWallet(db),
		repo.NewOrderRepo(db),
	)

	r := gin.New()
	r.Use(gin.Recovery(), httpapi.CORS())
	api.Routes(r)
	r.Run(":8080")
}
