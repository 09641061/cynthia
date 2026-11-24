package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h Handler) list(c *gin.Context) {
	categories, err := h.catalog.List(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, categories)
}

func (h Handler) getCategory(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	category, err := h.catalog.GetCategory(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("ETag", entityTag(category.Version))
	c.JSON(http.StatusOK, category)
}

func (h Handler) getService(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	service, err := h.catalog.GetService(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("ETag", entityTag(service.Version))
	c.JSON(http.StatusOK, service)
}
