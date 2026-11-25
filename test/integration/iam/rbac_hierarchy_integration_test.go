//go:build integration

package iam_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"go-service/internal/iam"
)

func TestCasbinHierarchyAntiLockoutAndSchedulingAppear(t *testing.T) {
	env := newTestEnv(t)
	ownerTok, _ := env.loginAs(t, "lucia@estudionube.mx", "TakoduDev2026!")

	// Owner creates two roles: AdminRole (pos 1) and StaffRole (pos 2).
	adminRole := createRole(t, env, ownerTok, "AdminRole", []string{"iam.administrator"})
	staffRole := createRole(t, env, ownerTok, "StaffRole", []string{"crm.manage"})

	// Seed two members: adminUser (pos 1) and staffUser (pos 2).
	adminID := seedUser(t, t.Context(), env.db.DB, "admin@estudionube.mx", "AdminPass123!", "user")
	staffID := seedUser(t, t.Context(), env.db.DB, "staff@estudionube.mx", "StaffPass123!", "user")
	assignRoles(t, env, ownerTok, adminID, []string{adminRole.ID}, http.StatusOK)
	assignRoles(t, env, ownerTok, staffID, []string{staffRole.ID}, http.StatusOK)

	staffTok, _ := env.loginAs(t, "staff@estudionube.mx", "StaffPass123!")
	adminTok, _ := env.loginAs(t, "admin@estudionube.mx", "AdminPass123!")

	// Casbin enforcement: staff lacks catalog.manage -> 403; owner & admin allowed -> 200.
	require.Equal(t, http.StatusForbidden, env.doJSON(t, http.MethodGet, "/v1/catalog/probe", staffTok, nil).Code)
	require.Equal(t, http.StatusOK, env.doJSON(t, http.MethodGet, "/v1/catalog/probe", ownerTok, nil).Code)
	require.Equal(t, http.StatusOK, env.doJSON(t, http.MethodGet, "/v1/catalog/probe", adminTok, nil).Code)

	// Hierarchy: give staff iam.manage_users (pos 2), still cannot delete admin (pos 1) or owner (pos 0) -> 403.
	staffRole = updateRole(t, env, ownerTok, staffRole.ID, "StaffRole", []string{"iam.manage_users"})
	require.Equal(t, http.StatusForbidden, env.doJSON(t, http.MethodDelete, "/v1/iam/members/"+adminID, staffTok, nil).Code)
	require.Equal(t, http.StatusForbidden, env.doJSON(t, http.MethodDelete, "/v1/iam/members/"+env.ownerID, staffTok, nil).Code)

	// Anti-lockout: admin cannot strip iam.administrator from their own AdminRole -> 403.
	w := env.doJSON(t, http.MethodPatch, "/v1/iam/roles/"+adminRole.ID, adminTok, map[string]any{
		"name": "AdminRole", "permissions": []string{"catalog.manage"},
	})
	require.Equal(t, http.StatusForbidden, w.Code)

	// scheduling.appear is NOT implied by Owner or Administrator; requires role grant.
	reader := iam.NewPersonReader(env.db)
	people, err := reader.ListActivePeople(t.Context())
	require.NoError(t, err)
	require.Empty(t, people)

	updateRole(t, env, ownerTok, staffRole.ID, "StaffRole", []string{"iam.manage_users", "scheduling.appear"})
	people, err = reader.ListActivePeople(t.Context())
	require.NoError(t, err)
	require.Len(t, people, 1)
	require.Equal(t, staffID, people[0].ID)
}

type roleResp struct {
	ID string `json:"id"`
}

func createRole(t *testing.T, env *testEnv, tok, name string, perms []string) roleResp {
	t.Helper()
	w := env.doJSON(t, http.MethodPost, "/v1/iam/roles", tok, map[string]any{
		"name": name, "permissions": perms,
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var r roleResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &r))
	return r
}

func updateRole(t *testing.T, env *testEnv, tok, id, name string, perms []string) roleResp {
	t.Helper()
	w := env.doJSON(t, http.MethodPatch, "/v1/iam/roles/"+id, tok, map[string]any{
		"name": name, "permissions": perms,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var r roleResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &r))
	return r
}

func assignRoles(t *testing.T, env *testEnv, tok, userID string, roleIDs []string, want int) {
	t.Helper()
	w := env.doJSON(t, http.MethodPut, "/v1/iam/members/"+userID+"/roles", tok, map[string]any{
		"role_ids": roleIDs,
	})
	require.Equal(t, want, w.Code, w.Body.String())
}
