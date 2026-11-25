package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	sharedrest "go-service/internal/shared/rest"
)

func (h Handler) createService(c *gin.Context) {
	categoryID, ok := sharedrest.PathUUID(c, "id")
	if !ok {
		return
	}
	var request createServiceRequest
	if !sharedrest.BindJSON(c, &request) {
		return
	}
	service, err := h.catalog.CreateService(c.Request.Context(), categoryID, request.service(), request.SortOrder)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Location", "/v1/catalog/services/"+service.ID)
	c.JSON(http.StatusCreated, service)
}

func (h Handler) updateService(c *gin.Context) {
	id, ok := sharedrest.PathUUID(c, "id")
	if !ok {
		return
	}
	var request serviceRequest
	if !sharedrest.BindJSON(c, &request) {
		return
	}
	service, err := h.catalog.UpdateService(c.Request.Context(), id, request.service())
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, service)
}

func (h Handler) deleteService(c *gin.Context) {
	id, ok := sharedrest.PathUUID(c, "id")
	if !ok {
		return
	}
	if err := h.catalog.DeleteService(c.Request.Context(), id); err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}

func (h Handler) moveService(c *gin.Context) {
	id, ok := sharedrest.PathUUID(c, "id")
	if !ok {
		return
	}
	var request placementRequest
	if !sharedrest.BindJSON(c, &request) {
		return
	}
	categoryID, ok := sharedrest.ParseUUID(request.CategoryID)
	if !ok {
		badRequest(c)
		return
	}
	service, err := h.catalog.MoveService(c.Request.Context(), id, categoryID, request.SortOrder)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, service)
}
