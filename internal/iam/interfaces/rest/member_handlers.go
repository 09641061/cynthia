package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go-service/internal/iam/application"
	sharedrest "go-service/internal/shared/rest"
)

type memberHandler struct{ members *application.MemberService }

func (h memberHandler) list(c *gin.Context) {
	actors, err := h.members.ListMembers(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	out := make([]ActorResource, 0, len(actors))
	for _, a := range actors {
		out = append(out, toActorResource(a))
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (h memberHandler) delete(c *gin.Context) {
	targetID, ok := sharedrest.PathUUID(c, "id")
	if !ok {
		return
	}
	if err := h.members.DeleteMember(c.Request.Context(), ActorID(c), targetID); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h memberHandler) assignRoles(c *gin.Context) {
	targetID, ok := sharedrest.PathUUID(c, "id")
	if !ok {
		return
	}
	var body struct {
		RoleIDs []string `json:"role_ids"`
	}
	if !sharedrest.BindJSON(c, &body) {
		return
	}
	if !sharedrest.ParseUUIDSlice(c, body.RoleIDs) {
		return
	}
	actor, err := h.members.AssignRoles(c.Request.Context(), ActorID(c), targetID, body.RoleIDs)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toActorResource(actor))
}
