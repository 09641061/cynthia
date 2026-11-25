package rest

import (
	"github.com/gin-gonic/gin"
	"go-service/internal/catalog/application"
)

// Handler adapts catalog application operations to HTTP.
type Handler struct{ catalog *application.Catalog }

// RegisterRoutes mounts the catalog HTTP resources on the supplied router.
func RegisterRoutes(router gin.IRouter, catalog *application.Catalog) {
	h := Handler{catalog: catalog}
	router.GET("/catalog", h.list)
	router.GET("/catalog/categories/:id", h.getCategory)
	router.GET("/catalog/services/:id", h.getService)
	router.POST("/catalog/categories", h.createCategory)
	router.PUT("/catalog/categories/:id", h.updateCategory)
	router.DELETE("/catalog/categories/:id", h.deleteCategory)
	router.PUT("/catalog/categories/:id/placement", h.moveCategory)
	router.POST("/catalog/categories/:id/services", h.createService)
	router.PUT("/catalog/services/:id", h.updateService)
	router.DELETE("/catalog/services/:id", h.deleteService)
	router.PUT("/catalog/services/:id/placement", h.moveService)
}
