package main

import (
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"example.com/shop/handlers"
)

func main() {
	gin.SetMode(gin.DebugMode)

	db, err := gorm.Open(postgres.Open("host=db user=shop dbname=shop sslmode=disable"), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	h := handlers.New(db)
	r := gin.Default()
	r.GET("/users/search", h.SearchUsers)
	r.GET("/orders/:id", h.RequireUser, h.GetOrder)
	r.POST("/tools/ping", h.Ping)
	r.GET("/proxy", h.Proxy)
	r.GET("/healthz", h.Health)
	r.Run(":8080")
}
