package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestHierarchyMemberDeletionAndRoles(t *testing.T) {
	owner := Actor{User: User{ID: uuid.NewString(), Type: UserTypeOwner}}
	r1 := Role{ID: uuid.NewString(), Position: 1, Permissions: []string{PermManageUsers, PermManageRoles}}
	r2 := Role{ID: uuid.NewString(), Position: 2, Permissions: []string{PermManageUsers, PermAdministrator}}
	r3 := Role{ID: uuid.NewString(), Position: 3, Permissions: []string{PermCatalogManage}}

	m1 := Actor{User: User{ID: uuid.NewString(), Type: UserTypeUser}, Roles: []Role{r1}}
	m2 := Actor{User: User{ID: uuid.NewString(), Type: UserTypeUser}, Roles: []Role{r2}}
	m3 := Actor{User: User{ID: uuid.NewString(), Type: UserTypeUser}, Roles: []Role{r3}}
	rolesByID := map[string]Role{r1.ID: r1, r2.ID: r2, r3.ID: r3}

	if err := CanDeleteMember(m1, owner); !errors.Is(err, ErrOwnerProtected) {
		t.Fatalf("delete owner = %v, want ErrOwnerProtected", err)
	}
	if err := CanDeleteMember(m2, m1); !errors.Is(err, ErrHierarchyViolation) {
		t.Fatalf("lower admin delete higher = %v, want ErrHierarchyViolation", err)
	}
	if err := CanDeleteMember(m1, m2); err != nil {
		t.Fatalf("higher delete lower = %v, want nil", err)
	}
	if err := CanAssignRoles(m1, m1, []string{r3.ID}, rolesByID); !errors.Is(err, ErrSelfRoleModification) {
		t.Fatalf("self role assign = %v, want ErrSelfRoleModification", err)
	}
	if err := CanAssignRoles(m1, m3, []string{r2.ID}, rolesByID); err != nil {
		t.Fatalf("m1 assign lower role r2 (even with admin) to m3 = %v, want nil", err)
	}
	if err := CanAssignRoles(m2, m3, []string{r1.ID}, rolesByID); !errors.Is(err, ErrHierarchyViolation) {
		t.Fatalf("m2 assign higher role r1 = %v, want ErrHierarchyViolation", err)
	}
}

func TestEditRolePermissionHeldAndAntiLockout(t *testing.T) {
	r1 := Role{ID: uuid.NewString(), Position: 1, Permissions: []string{PermManageRoles}}
	r2 := Role{ID: uuid.NewString(), Position: 2, Permissions: []string{PermManageRoles, PermCatalogManage}}
	r3 := Role{ID: uuid.NewString(), Position: 3, Permissions: []string{PermAdministrator}}

	actor := Actor{User: User{ID: uuid.NewString(), Type: UserTypeUser}, Roles: []Role{r1, r2}}
	if err := CanEditRole(actor, r3, []string{PermCatalogManage}); !errors.Is(err, ErrPermissionNotHeld) {
		t.Fatalf("toggle unheld admin perm = %v, want ErrPermissionNotHeld", err)
	}
	if err := CanEditRole(actor, r2, []string{PermCatalogManage}); err != nil {
		t.Fatalf("remove manage_roles from r2 while holding r1 = %v, want nil", err)
	}

	r1NoManage := Role{ID: r1.ID, Position: 1, Permissions: []string{PermManageUsers}}
	actorLockout := Actor{User: actor.User, Roles: []Role{r1NoManage, r2}}
	if err := CanEditRole(actorLockout, r2, []string{PermCatalogManage}); !errors.Is(err, ErrLockoutSelfManageRoles) {
		t.Fatalf("self lockout = %v, want ErrLockoutSelfManageRoles", err)
	}
}

func TestReorderRolesHierarchy(t *testing.T) {
	owner := Actor{User: User{ID: uuid.NewString(), Type: UserTypeOwner}}
	r1 := Role{ID: uuid.NewString(), Position: 1, Permissions: []string{PermManageRoles, PermManageUsers}}
	r2 := Role{ID: uuid.NewString(), Position: 2, Permissions: []string{PermManageRoles}}
	r3 := Role{ID: uuid.NewString(), Position: 3, Permissions: []string{PermCatalogManage}}

	m1 := Actor{User: User{ID: uuid.NewString(), Type: UserTypeUser}, Roles: []Role{r1}}
	roles := []Role{r1, r2, r3}

	if err := CanReorderRoles(m1, []string{r1.ID, r3.ID, r2.ID}, roles); err != nil {
		t.Fatalf("m1 swap r2 and r3 = %v, want nil", err)
	}
	if err := CanReorderRoles(m1, []string{r2.ID, r1.ID, r3.ID}, roles); !errors.Is(err, ErrHierarchyViolation) {
		t.Fatalf("m1 move r1 = %v, want ErrHierarchyViolation", err)
	}
	if err := CanReorderRoles(owner, []string{r3.ID, r1.ID, r2.ID}, roles); err != nil {
		t.Fatalf("owner reorder all = %v, want nil", err)
	}
}
