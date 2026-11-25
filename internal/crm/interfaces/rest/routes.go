package rest

import (
	"github.com/gin-gonic/gin"
	"go-service/internal/crm/application"
)

// Handler adapts CRM application operations to HTTP.
type Handler struct{ customers *application.CustomerService }

// RegisterRoutes mounts the CRM HTTP resources on the supplied router.
func RegisterRoutes(router gin.IRouter, customers *application.CustomerService) {
	h := Handler{customers: customers}
	router.GET("/customers", h.listCustomers)
	router.POST("/customers", h.createCustomer)
	router.GET("/customers/:id", h.getCustomer)
	router.PATCH("/customers/:id", h.updateCustomer)
	router.DELETE("/customers/:id", h.deleteCustomer)
}
