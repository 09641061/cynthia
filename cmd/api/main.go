package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatal(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	fmt.Printf("OpenAPI disponible en http://localhost:%s/openapi.yaml\n", port)
	if err := newRouter().Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

func newRouter() *gin.Engine {
	router := gin.Default()
	router.GET("/hello", func(c *gin.Context) {
		c.String(http.StatusOK, "hello world")
	})
	router.StaticFile("/openapi.yaml", "api/openapi.yaml")
	return router
}
