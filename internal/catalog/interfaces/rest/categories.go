package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	sharedrest "go-service/internal/shared/rest"
)

func (h Handler) createCategory(c *gin.Context) {
	var request categoryRequest
	if !sharedrest.BindJSON(c, &request) {
		return
	}
	category, err := h.catalog.CreateCategory(c.Request.Context(), request.Name)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Location", "/v1/catalog/categories/"+category.ID)
	c.JSON(http.StatusCreated, category)
}

func (h Handler) updateCategory(c *gin.Context) {
	id, ok := sharedrest.PathUUID(c, "id")
	if !ok {
		return
	}
	var request categoryRequest
	if !sharedrest.BindJSON(c, &request) {
		return
	}
	category, err := h.catalog.UpdateCategory(c.Request.Context(), id, request.Name)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, category)
}

func (h Handler) deleteCategory(c *gin.Context) {
	id, ok := sharedrest.PathUUID(c, "id")
	if !ok {
		return
	}
	if err := h.catalog.DeleteCategory(c.Request.Context(), id); err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}

func (h Handler) moveCategory(c *gin.Context) {
	id, ok := sharedrest.PathUUID(c, "id")
	if !ok {
		return
	}
	var request categoryPlacementRequest
	if !sharedrest.BindJSON(c, &request) {
		return
	}
	category, err := h.catalog.MoveCategory(c.Request.Context(), id, request.SortOrder)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, category)
}
