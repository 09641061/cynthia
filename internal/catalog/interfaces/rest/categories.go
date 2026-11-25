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
	c.Header("ETag", sharedrest.EntityTag(category.Version))
	c.JSON(http.StatusCreated, category)
}

func (h Handler) updateCategory(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	version, ok := sharedrest.IfMatch(c)
	if !ok {
		return
	}
	var request categoryRequest
	if !sharedrest.BindJSON(c, &request) {
		return
	}
	category, err := h.catalog.UpdateCategory(c.Request.Context(), id, request.Name, version)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("ETag", sharedrest.EntityTag(category.Version))
	c.JSON(http.StatusOK, category)
}

func (h Handler) deleteCategory(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	version, ok := sharedrest.IfMatch(c)
	if !ok {
		return
	}
	if err := h.catalog.DeleteCategory(c.Request.Context(), id, version); err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}

func (h Handler) moveCategory(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	version, ok := sharedrest.IfMatch(c)
	if !ok {
		return
	}
	var request categoryPlacementRequest
	if !sharedrest.BindJSON(c, &request) {
		return
	}
	category, err := h.catalog.MoveCategory(c.Request.Context(), id, request.SortOrder, version)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("ETag", sharedrest.EntityTag(category.Version))
	c.JSON(http.StatusOK, category)
}
