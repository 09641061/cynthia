package application

import (
	"context"

	"github.com/google/uuid"
	"go-service/internal/iam/domain"
)

// RoleService handles role CRUD and hierarchy reordering.
type RoleService struct {
	users UserStore
	roles RoleStore
}

// NewRoleService constructs a RoleService.
func NewRoleService(users UserStore, roles RoleStore) *RoleService {
	return &RoleService{users: users, roles: roles}
}

// ListRoles returns all active roles ordered by priority position ascending.
func (s *RoleService) ListRoles(ctx context.Context) ([]domain.Role, error) {
	return s.roles.ListRoles(ctx)
}

// CreateRole creates a new role at the bottom of the hierarchy, enforcing the 50-role cap.
func (s *RoleService) CreateRole(ctx context.Context, actorID, name string, perms []string) (domain.Role, error) {
	actor, err := s.users.GetActorByID(ctx, actorID)
	if err != nil {
		return domain.Role{}, err
	}
	if !actor.CanManageRoles() {
		return domain.Role{}, domain.ErrForbidden
	}
	count, err := s.roles.CountRoles(ctx)
	if err != nil {
		return domain.Role{}, err
	}
	if count >= domain.MaxActiveRoles {
		return domain.Role{}, domain.ErrRoleLimitExceeded
	}
	cleanName, cleanPerms, err := domain.ValidateRoleInput(name, perms)
	if err != nil {
		return domain.Role{}, err
	}
	if !actor.User.IsOwner() {
		held := actor.EffectivePermissions()
		for _, p := range cleanPerms {
			if !held[p] {
				return domain.Role{}, domain.ErrPermissionNotHeld
			}
		}
	}
	return s.roles.CreateRole(ctx, domain.Role{
		ID:          uuid.NewString(),
		Name:        cleanName,
		Permissions: cleanPerms,
	})
}

// UpdateRole updates a role's name and permissions subject to hierarchy and anti-lockout rules.
func (s *RoleService) UpdateRole(ctx context.Context, actorID, roleID, name string, perms []string) (domain.Role, error) {
	actor, err := s.users.GetActorByID(ctx, actorID)
	if err != nil {
		return domain.Role{}, err
	}
	role, err := s.roles.GetRole(ctx, roleID)
	if err != nil {
		return domain.Role{}, err
	}
	cleanName, cleanPerms, err := domain.ValidateRoleInput(name, perms)
	if err != nil {
		return domain.Role{}, err
	}
	if err := domain.CanEditRole(actor, role, cleanPerms); err != nil {
		return domain.Role{}, err
	}
	return s.roles.UpdateRole(ctx, roleID, cleanName, cleanPerms)
}

// DeleteRole deletes a role subject to hierarchy rules.
func (s *RoleService) DeleteRole(ctx context.Context, actorID, roleID string) error {
	actor, err := s.users.GetActorByID(ctx, actorID)
	if err != nil {
		return err
	}
	role, err := s.roles.GetRole(ctx, roleID)
	if err != nil {
		return err
	}
	if err := domain.CanDeleteRole(actor, role); err != nil {
		return err
	}
	return s.roles.DeleteRole(ctx, roleID)
}

// ReorderRoles updates role positions after verifying hierarchy boundaries.
func (s *RoleService) ReorderRoles(ctx context.Context, actorID string, orderedIDs []string) ([]domain.Role, error) {
	actor, err := s.users.GetActorByID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	current, err := s.roles.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	if err := domain.CanReorderRoles(actor, orderedIDs, current); err != nil {
		return nil, err
	}
	return s.roles.ReorderRoles(ctx, orderedIDs)
}
