package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go-service/internal/iam/application"
	sharedrest "go-service/internal/shared/rest"
)

type inviteHandler struct{ invites *application.InviteService }

func (h inviteHandler) list(c *gin.Context) {
	items, err := h.invites.ListPending(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	out := make([]InvitationResource, 0, len(items))
	for _, inv := range items {
		out = append(out, toInvitationResource(inv))
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (h inviteHandler) create(c *gin.Context) {
	var body struct {
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	if !sharedrest.BindJSON(c, &body) {
		return
	}
	inv, err := h.invites.CreateInvitation(c.Request.Context(), ActorID(c), body.Email, body.FirstName, body.LastName)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, toInvitationResource(inv))
}

func (h inviteHandler) delete(c *gin.Context) {
	inviteID, ok := sharedrest.PathUUID(c, "id")
	if !ok {
		return
	}
	if err := h.invites.DeleteInvitation(c.Request.Context(), ActorID(c), inviteID); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h inviteHandler) resend(c *gin.Context) {
	inviteID, ok := sharedrest.PathUUID(c, "id")
	if !ok {
		return
	}
	inv, err := h.invites.ResendInvitation(c.Request.Context(), ActorID(c), inviteID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toInvitationResource(inv))
}
