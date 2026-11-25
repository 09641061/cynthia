package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go-service/internal/calendar/application"
	sharedrest "go-service/internal/shared/rest"
)

// Handler adapts calendar application use cases to HTTP.
type Handler struct {
	service *application.Service
}

func (h Handler) listPeople(c *gin.Context) {
	people, err := h.service.ListPeople(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toPersonResources(people))
}

func (h Handler) listAppointments(c *gin.Context) {
	filter, ok := parseFilter(c)
	if !ok {
		return
	}
	views, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toAppointmentResources(views))
}

func (h Handler) createAppointment(c *gin.Context) {
	var input createAppointmentInput
	if !sharedrest.BindJSON(c, &input) {
		return
	}
	if !validUUID(c, input.PersonID) || !validUUID(c, input.CustomerID) || !validUUID(c, input.ServiceID) {
		return
	}
	created, err := h.service.Create(c.Request.Context(), application.CreateAppointmentInput{
		PersonID:   input.PersonID,
		CustomerID: input.CustomerID,
		ServiceID:  input.ServiceID,
		StartsAt:   input.StartsAt,
		EndsAt:     input.EndsAt,
		Status:     input.Status,
		Channel:    input.Channel,
		Notes:      input.Notes,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Location", "/v1/appointments/"+created.Appointment.ID)
	c.JSON(http.StatusCreated, toAppointmentResource(created))
}

func (h Handler) getAppointment(c *gin.Context) {
	id, ok := sharedrest.PathUUID(c, "id")
	if !ok {
		return
	}
	view, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toAppointmentResource(view))
}

func (h Handler) updateAppointment(c *gin.Context) {
	id, ok := sharedrest.PathUUID(c, "id")
	if !ok {
		return
	}
	var input updateAppointmentInput
	if !sharedrest.BindJSON(c, &input) {
		return
	}
	if !validOptionalUUID(c, input.PersonID) || !validOptionalUUID(c, input.CustomerID) || !validOptionalUUID(c, input.ServiceID) {
		return
	}
	updated, err := h.service.Update(c.Request.Context(), id, application.UpdateAppointmentInput{
		PersonID:   input.PersonID,
		CustomerID: input.CustomerID,
		ServiceID:  input.ServiceID,
		StartsAt:   input.StartsAt,
		EndsAt:     input.EndsAt,
		Status:     input.Status,
		Notes:      input.Notes,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toAppointmentResource(updated))
}

func (h Handler) deleteAppointment(c *gin.Context) {
	id, ok := sharedrest.PathUUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}
