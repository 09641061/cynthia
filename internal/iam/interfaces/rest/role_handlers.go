package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go-service/internal/iam/application"
	sharedrest "go-service/internal/shared/rest"
)

type roleHandler struct{ roles *application.RoleService }

func (h roleHandler) list(c *gin.Context) {
	roles, err := h.roles.ListRoles(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	out := make([]RoleResource, 0, len(roles))
	for _, r := range roles {
		out = append(out, toRoleResource(r))
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (h roleHandler) create(c *gin.Context) {
	var body struct {
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
	}
	if !sharedrest.BindJSON(c, &body) {
		return
	}
	role, err := h.roles.CreateRole(c.Request.Context(), ActorID(c), body.Name, body.Permissions)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, toRoleResource(role))
}

func (h roleHandler) update(c *gin.Context) {
	roleID, ok := sharedrest.PathUUID(c, "id")
	if !ok {
		return
	}
	var body struct {
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
	}
	if !sharedrest.BindJSON(c, &body) {
		return
	}
	role, err := h.roles.UpdateRole(c.Request.Context(), ActorID(c), roleID, body.Name, body.Permissions)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toRoleResource(role))
}

func (h roleHandler) delete(c *gin.Context) {
	roleID, ok := sharedrest.PathUUID(c, "id")
	if !ok {
		return
	}
	if err := h.roles.DeleteRole(c.Request.Context(), ActorID(c), roleID); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h roleHandler) reorder(c *gin.Context) {
	var body struct {
		OrderedIDs []string `json:"ordered_ids"`
	}
	if !sharedrest.BindJSON(c, &body) {
		return
	}
	if !sharedrest.ParseUUIDSlice(c, body.OrderedIDs) {
		return
	}
	roles, err := h.roles.ReorderRoles(c.Request.Context(), ActorID(c), body.OrderedIDs)
	if err != nil {
		writeError(c, err)
		return
	}
	out := make([]RoleResource, 0, len(roles))
	for _, r := range roles {
		out = append(out, toRoleResource(r))
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"items": out})
}
