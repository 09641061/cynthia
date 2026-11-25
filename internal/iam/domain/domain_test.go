package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestActorLevelAndPermissions(t *testing.T) {
	owner := Actor{User: User{ID: uuid.NewString(), Type: UserTypeOwner}}
	if owner.Level() != OwnerLevel {
		t.Fatalf("owner.Level() = %d, want %d", owner.Level(), OwnerLevel)
	}
	if !owner.HasPermission(PermManageUsers) || !owner.HasPermission(PermManageRoles) {
		t.Fatal("owner must hold functional permissions")
	}
	if owner.HasPermission(PermSchedulingAppear) {
		t.Fatal("owner must not have scheduling.appear without explicit grant")
	}

	noRoles := Actor{User: User{ID: uuid.NewString(), Type: UserTypeUser}}
	if noRoles.Level() != BaseLevel {
		t.Fatalf("noRoles.Level() = %d, want BaseLevel", noRoles.Level())
	}
	if noRoles.HasPermission(PermManageUsers) || noRoles.HasPermission(PermCatalogManage) {
		t.Fatal("member without roles must have no functional permissions")
	}

	r1 := Role{ID: uuid.NewString(), Position: 1, Permissions: []string{PermManageUsers}}
	r3 := Role{ID: uuid.NewString(), Position: 3, Permissions: []string{PermManageRoles, PermSchedulingAppear}}
	multi := Actor{User: User{ID: uuid.NewString(), Type: UserTypeUser}, Roles: []Role{r3, r1}}
	if multi.Level() != 1 {
		t.Fatalf("multi.Level() = %d, want 1", multi.Level())
	}
	if !multi.HasPermission(PermManageUsers) || !multi.HasPermission(PermManageRoles) || !multi.HasPermission(PermSchedulingAppear) {
		t.Fatal("member permissions must be union of all roles")
	}

	admin := Actor{
		User:  User{ID: uuid.NewString(), Type: UserTypeUser},
		Roles: []Role{{ID: uuid.NewString(), Position: 2, Permissions: []string{PermAdministrator}}},
	}
	if !admin.HasPermission(PermManageUsers) || !admin.HasPermission(PermCatalogManage) {
		t.Fatal("iam.administrator must grant all functional permissions")
	}
	if admin.HasPermission(PermSchedulingAppear) {
		t.Fatal("iam.administrator must NOT grant scheduling.appear")
	}
}

func TestValidationHelpers(t *testing.T) {
	if !IsAssignablePermission(PermSchedulingAppear) {
		t.Fatal("scheduling.appear must be assignable to roles")
	}
	perms, err := NormalizePermissions([]string{PermSchedulingAppear, PermCalendarManage, PermSchedulingAppear})
	if err != nil || len(perms) != 2 {
		t.Fatalf("NormalizePermissions = (%v, %v), want 2 unique perms", perms, err)
	}
	if _, err := NormalizePermissions([]string{"unknown.perm"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("NormalizePermissions(unknown.perm) = %v, want ErrValidation", err)
	}
	if _, err := NormalizeEmail("BadEmail"); !errors.Is(err, ErrValidation) {
		t.Fatalf("NormalizeEmail(BadEmail) = %v, want ErrValidation", err)
	}
	got, err := NormalizeEmail("  Lucia@EstudioNube.MX ")
	if err != nil || got != "lucia@estudionube.mx" {
		t.Fatalf("NormalizeEmail = (%q, %v), want (lucia@estudionube.mx, nil)", got, err)
	}
}
