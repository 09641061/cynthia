package application

import (
	"context"

	"go-service/internal/iam/domain"
)

// MemberService handles member listing, deletion, and role assignments.
type MemberService struct {
	users    UserStore
	roles    RoleStore
	sessions SessionStore
}

// NewMemberService constructs a MemberService.
func NewMemberService(users UserStore, roles RoleStore, sessions SessionStore) *MemberService {
	return &MemberService{users: users, roles: roles, sessions: sessions}
}

// ListMembers returns all members (including Owner) with their roles.
func (s *MemberService) ListMembers(ctx context.Context) ([]domain.Actor, error) {
	return s.users.ListActors(ctx)
}

// DeleteMember removes a member after verifying permission, owner protection, and hierarchy.
func (s *MemberService) DeleteMember(ctx context.Context, actorID, targetID string) error {
	actor, err := s.users.GetActorByID(ctx, actorID)
	if err != nil {
		return err
	}
	target, err := s.users.GetActorByID(ctx, targetID)
	if err != nil {
		return err
	}
	if err := domain.CanDeleteMember(actor, target); err != nil {
		return err
	}
	if err := s.users.DeleteMember(ctx, targetID); err != nil {
		return err
	}
	if s.sessions != nil {
		_ = s.sessions.RevokeAllUserSessions(ctx, targetID)
	}
	return nil
}

// AssignRoles replaces a member's assigned roles after verifying hierarchy and self-modification rules.
func (s *MemberService) AssignRoles(ctx context.Context, actorID, targetID string, roleIDs []string) (domain.Actor, error) {
	actor, err := s.users.GetActorByID(ctx, actorID)
	if err != nil {
		return domain.Actor{}, err
	}
	target, err := s.users.GetActorByID(ctx, targetID)
	if err != nil {
		return domain.Actor{}, err
	}
	allRoles, err := s.roles.ListRoles(ctx)
	if err != nil {
		return domain.Actor{}, err
	}
	rolesByID := make(map[string]domain.Role, len(allRoles))
	for _, r := range allRoles {
		rolesByID[r.ID] = r
	}
	if err := domain.CanAssignRoles(actor, target, roleIDs, rolesByID); err != nil {
		return domain.Actor{}, err
	}
	return s.users.ReplaceMemberRoles(ctx, targetID, roleIDs)
}
