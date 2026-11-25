package rest

import (
	"github.com/gin-gonic/gin"
	"go-service/internal/calendar/application"
)

// RegisterRoutes mounts the calendar HTTP endpoints on the supplied router.
func RegisterRoutes(router gin.IRouter, service *application.Service) {
	h := Handler{service: service}
	router.GET("/calendar/people", h.listPeople)
	router.GET("/appointments", h.listAppointments)
	router.POST("/appointments", h.createAppointment)
	router.GET("/appointments/:id", h.getAppointment)
	router.PATCH("/appointments/:id", h.updateAppointment)
	router.DELETE("/appointments/:id", h.deleteAppointment)
}
