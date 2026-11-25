//go:build integration

package iam_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthLoginRefreshReuseAndLogout(t *testing.T) {
	env := newTestEnv(t)

	// Invalid credentials -> 401.
	bad := env.doJSON(t, http.MethodPost, "/v1/auth/login", "", map[string]string{
		"email": "lucia@estudionube.mx", "password": "wrong-password",
	})
	require.Equal(t, http.StatusUnauthorized, bad.Code)

	// Valid login -> 200 with tokens.
	access1, refresh1 := env.loginAs(t, "Lucia@EstudioNube.MX", "TakoduDev2026!")
	require.Equal(t, http.StatusOK, env.doJSON(t, http.MethodGet, "/v1/auth/me", access1, nil).Code)

	// Refresh rotation issues new pair.
	w := env.doJSON(t, http.MethodPost, "/v1/auth/refresh", "", map[string]string{
		"refresh_token": refresh1,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var rotated struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rotated))
	require.NotEqual(t, refresh1, rotated.RefreshToken)
	require.Equal(t, http.StatusOK, env.doJSON(t, http.MethodGet, "/v1/auth/me", rotated.AccessToken, nil).Code)

	// Reusing old refresh token triggers reuse detection: 401 and revokes rotated session!
	reuse := env.doJSON(t, http.MethodPost, "/v1/auth/refresh", "", map[string]string{
		"refresh_token": refresh1,
	})
	require.Equal(t, http.StatusUnauthorized, reuse.Code)
	require.Equal(t, http.StatusUnauthorized, env.doJSON(t, http.MethodGet, "/v1/auth/me", rotated.AccessToken, nil).Code)

	// Login again, then logout revokes BOTH access session and paired refresh token.
	access3, refresh3 := env.loginAs(t, "lucia@estudionube.mx", "TakoduDev2026!")
	logout := env.doJSON(t, http.MethodPost, "/v1/auth/logout", access3, nil)
	require.Equal(t, http.StatusNoContent, logout.Code)
	require.Equal(t, http.StatusUnauthorized, env.doJSON(t, http.MethodGet, "/v1/auth/me", access3, nil).Code)
	afterLogoutRefresh := env.doJSON(t, http.MethodPost, "/v1/auth/refresh", "", map[string]string{
		"refresh_token": refresh3,
	})
	require.Equal(t, http.StatusUnauthorized, afterLogoutRefresh.Code)
}
