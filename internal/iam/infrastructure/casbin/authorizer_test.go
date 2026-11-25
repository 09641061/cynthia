package casbin

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go-service/internal/iam/domain"
)

type fakeActorLoader struct {
	actors map[string]domain.Actor
	err    error
}

func (f fakeActorLoader) GetActorByID(_ context.Context, userID string) (domain.Actor, error) {
	if f.err != nil {
		return domain.Actor{}, f.err
	}
	a, ok := f.actors[userID]
	if !ok {
		return domain.Actor{}, domain.ErrUserNotFound
	}
	return a, nil
}

func TestCasbinEnforcesOwnerAdminAndRolePermissions(t *testing.T) {
	ownerID := uuid.NewString()
	adminID := uuid.NewString()
	memberID := uuid.NewString()

	loader := &fakeActorLoader{
		actors: map[string]domain.Actor{
			ownerID: {User: domain.User{ID: ownerID, Type: domain.UserTypeOwner}},
			adminID: {
				User:  domain.User{ID: adminID, Type: domain.UserTypeUser},
				Roles: []domain.Role{{ID: uuid.NewString(), Position: 1, Permissions: []string{domain.PermAdministrator}}},
			},
			memberID: {
				User:  domain.User{ID: memberID, Type: domain.UserTypeUser},
				Roles: []domain.Role{{ID: uuid.NewString(), Position: 2, Permissions: []string{domain.PermCalendarManage}}},
			},
		},
	}
	authz, err := NewAuthorizer(loader)
	if err != nil {
		t.Fatalf("NewAuthorizer: %v", err)
	}
	ctx := t.Context()

	assertEnforce(t, authz, ctx, ownerID, "iam", "manage_roles", true)
	assertEnforce(t, authz, ctx, adminID, "catalog", "manage", true)
	assertEnforce(t, authz, ctx, memberID, "calendar", "manage", true)
	assertEnforce(t, authz, ctx, memberID, "iam", "manage_users", false)

	// Immediate revocation: removing roles immediately denies access.
	m := loader.actors[memberID]
	m.Roles = nil
	loader.actors[memberID] = m
	assertEnforce(t, authz, ctx, memberID, "calendar", "manage", false)
}

func TestCasbinFailClosedOnLoaderError(t *testing.T) {
	loader := fakeActorLoader{err: errors.New("db unavailable")}
	authz, err := NewAuthorizer(loader)
	if err != nil {
		t.Fatal(err)
	}
	allowed, err := authz.Enforce(t.Context(), uuid.NewString(), "catalog", "manage")
	if err == nil || allowed {
		t.Fatalf("expected fail-closed (false, err), got (%v, %v)", allowed, err)
	}
}

func assertEnforce(t *testing.T, authz *Authorizer, ctx context.Context, sub, obj, act string, want bool) {
	t.Helper()
	got, err := authz.Enforce(ctx, sub, obj, act)
	if err != nil {
		t.Fatalf("Enforce(%s, %s, %s) unexpected err: %v", sub, obj, act, err)
	}
	if got != want {
		t.Fatalf("Enforce(%s, %s, %s) = %v, want %v", sub, obj, act, got, want)
	}
}
