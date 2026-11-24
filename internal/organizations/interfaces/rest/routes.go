package rest

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"go-service/internal/organizations/domain"
	sharedrest "go-service/internal/shared/rest"
)

// Store persists and loads the singleton organization record.
type Store interface {
	Get(ctx context.Context) (domain.Organization, error)
	Update(ctx context.Context, expectedVersion int64, patch domain.UpdatePatch) (domain.Organization, error)
}

// Handler adapts organization application operations to HTTP.
type Handler struct{ store Store }

// RegisterRoutes mounts the organization HTTP resources on the supplied router.
func RegisterRoutes(router gin.IRouter, store Store) {
	h := Handler{store: store}
	router.GET("/organization", h.get)
	router.PATCH("/organization", h.patch)
}

func (h Handler) get(c *gin.Context) {
	org, err := h.store.Get(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("ETag", sharedrest.EntityTag(org.Version))
	c.JSON(http.StatusOK, org)
}

func (h Handler) patch(c *gin.Context) {
	version, ok := sharedrest.IfMatch(c)
	if !ok {
		return
	}
	var patch domain.UpdatePatch
	if !sharedrest.BindJSON(c, &patch) {
		return
	}
	org, err := h.store.Update(c.Request.Context(), version, patch)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("ETag", sharedrest.EntityTag(org.Version))
	c.JSON(http.StatusOK, org)
}
