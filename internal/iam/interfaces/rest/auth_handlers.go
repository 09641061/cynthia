package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go-service/internal/iam/application"
	sharedrest "go-service/internal/shared/rest"
)

type authHandler struct{ auth *application.AuthService }

func (h authHandler) login(c *gin.Context) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !sharedrest.BindJSON(c, &body) {
		return
	}
	pair, actor, err := h.auth.Login(c.Request.Context(), body.Email, body.Password)
	if err != nil {
		writeError(c, err)
		return
	}
	res := toActorResource(actor)
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, AuthResponse{
		AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken,
		TokenType: "Bearer", ExpiresIn: pair.ExpiresIn, User: &res,
	})
}

func (h authHandler) refresh(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if !sharedrest.BindJSON(c, &body) {
		return
	}
	pair, err := h.auth.Refresh(c.Request.Context(), body.RefreshToken)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, AuthResponse{
		AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken,
		TokenType: "Bearer", ExpiresIn: pair.ExpiresIn,
	})
}

func (h authHandler) logout(c *gin.Context) {
	raw, _ := extractBearer(c.GetHeader("Authorization"))
	if err := h.auth.Logout(c.Request.Context(), raw); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h authHandler) me(c *gin.Context) {
	actor, err := h.auth.Me(c.Request.Context(), ActorID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toActorResource(actor))
}

func (h authHandler) updateMe(c *gin.Context) {
	var body struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Password  string `json:"password"`
	}
	if !sharedrest.BindJSON(c, &body) {
		return
	}
	actor, err := h.auth.UpdateMe(c.Request.Context(), ActorID(c), body.FirstName, body.LastName, body.Password)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toActorResource(actor))
}

func (h authHandler) verifyInvite(c *gin.Context) {
	inv, err := h.auth.VerifyInvitation(c.Request.Context(), c.Query("token"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toInvitationResource(inv))
}

func (h authHandler) acceptInvite(c *gin.Context) {
	var body struct {
		Token     string `json:"token"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Password  string `json:"password"`
	}
	if !sharedrest.BindJSON(c, &body) {
		return
	}
	pair, actor, err := h.auth.AcceptInvitation(c.Request.Context(), body.Token, body.FirstName, body.LastName, body.Password)
	if err != nil {
		writeError(c, err)
		return
	}
	res := toActorResource(actor)
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, AuthResponse{
		AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken,
		TokenType: "Bearer", ExpiresIn: pair.ExpiresIn, User: &res,
	})
}
