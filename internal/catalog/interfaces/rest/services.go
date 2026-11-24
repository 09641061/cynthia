package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h Handler) createService(c *gin.Context) {
	categoryID, ok := pathID(c, "id")
	if !ok {
		return
	}
	var request createServiceRequest
	if !bindJSON(c, &request) {
		return
	}
	service, err := h.catalog.CreateService(c.Request.Context(), categoryID, request.service(), request.SortOrder)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Location", "/v1/catalog/services/"+service.ID)
	c.Header("ETag", entityTag(service.Version))
	c.JSON(http.StatusCreated, service)
}

func (h Handler) updateService(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	version, ok := ifMatch(c)
	if !ok {
		return
	}
	var request serviceRequest
	if !bindJSON(c, &request) {
		return
	}
	service, err := h.catalog.UpdateService(c.Request.Context(), id, request.service(), version)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("ETag", entityTag(service.Version))
	c.JSON(http.StatusOK, service)
}

func (h Handler) deleteService(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	version, ok := ifMatch(c)
	if !ok {
		return
	}
	if err := h.catalog.DeleteService(c.Request.Context(), id, version); err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}

func (h Handler) moveService(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	version, ok := ifMatch(c)
	if !ok {
		return
	}
	var request placementRequest
	if !bindJSON(c, &request) {
		return
	}
	categoryID, ok := normalizeID(request.CategoryID)
	if !ok {
		badRequest(c)
		return
	}
	service, err := h.catalog.MoveService(c.Request.Context(), id, categoryID, request.SortOrder, version)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("ETag", entityTag(service.Version))
	c.JSON(http.StatusOK, service)
}
