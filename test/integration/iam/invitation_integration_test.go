//go:build integration

package iam_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInvitationCreateDuplicateCancelAndSingleUseAccept(t *testing.T) {
	env := newTestEnv(t)
	ownerTok, _ := env.loginAs(t, "lucia@estudionube.mx", "TakoduDev2026!")

	// Create invitation via HTTP with empty first_name and last_name -> 201.
	w := env.doJSON(t, http.MethodPost, "/v1/iam/invitations", ownerTok, map[string]string{
		"email": "cancellable@estudionube.mx", "first_name": "", "last_name": "",
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var created struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	cancelledToken := env.lastInviteToken
	require.NotEmpty(t, cancelledToken)

	// Duplicate pending invitation for same email -> 409 Conflict.
	dup := env.doJSON(t, http.MethodPost, "/v1/iam/invitations", ownerTok, map[string]string{
		"email": "CANCELLABLE@estudionube.mx",
	})
	require.Equal(t, http.StatusConflict, dup.Code)

	// Cancel invitation -> 204 and token can no longer be verified.
	del := env.doJSON(t, http.MethodDelete, "/v1/iam/invitations/"+created.ID, ownerTok, nil)
	require.Equal(t, http.StatusNoContent, del.Code)
	verCancelled := env.doJSON(t, http.MethodGet, "/v1/auth/invitations/verify?token="+cancelledToken, "", nil)
	require.Equal(t, http.StatusBadRequest, verCancelled.Code)

	// Create a new invitation, verify -> 200, accept once -> 201, accept again -> 400.
	w2 := env.doJSON(t, http.MethodPost, "/v1/iam/invitations", ownerTok, map[string]string{
		"email": "diego@estudionube.mx",
	})
	require.Equal(t, http.StatusCreated, w2.Code, w2.Body.String())
	rawToken := env.lastInviteToken
	require.NotEmpty(t, rawToken)

	ver := env.doJSON(t, http.MethodGet, "/v1/auth/invitations/verify?token="+rawToken, "", nil)
	require.Equal(t, http.StatusOK, ver.Code, ver.Body.String())

	acc1 := env.doJSON(t, http.MethodPost, "/v1/auth/invitations/accept", "", map[string]string{
		"token": rawToken, "first_name": "Diego", "last_name": "Lopez", "password": "SecurePass123!",
	})
	require.Equal(t, http.StatusCreated, acc1.Code, acc1.Body.String())

	acc2 := env.doJSON(t, http.MethodPost, "/v1/auth/invitations/accept", "", map[string]string{
		"token": rawToken, "first_name": "Diego", "last_name": "Lopez", "password": "SecurePass123!",
	})
	require.Equal(t, http.StatusBadRequest, acc2.Code)
}
