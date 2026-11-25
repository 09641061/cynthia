package application

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"go-service/internal/iam/domain"
)

// VerifyInvitation validates a pending invitation token without consuming it.
func (s *AuthService) VerifyInvitation(ctx context.Context, rawToken string) (PendingInvite, error) {
	if strings.TrimSpace(rawToken) == "" {
		return PendingInvite{}, domain.ErrInviteExpiredOrUsed
	}
	return s.invites.VerifyInvite(ctx, domain.HashTokenSHA256(rawToken))
}

// AcceptInvitation atomically consumes the invitation, creates the Member account, and logs them in.
func (s *AuthService) AcceptInvitation(ctx context.Context, rawToken, firstName, lastName, password string) (TokenPair, domain.Actor, error) {
	if strings.TrimSpace(rawToken) == "" || len(password) < 8 {
		return TokenPair{}, domain.Actor{}, domain.ErrValidation
	}
	fn, ln, err := domain.ValidateNameParts(firstName, lastName)
	if err != nil {
		return TokenPair{}, domain.Actor{}, err
	}
	invite, err := s.invites.ConsumeInvite(ctx, domain.HashTokenSHA256(rawToken))
	if err != nil {
		return TokenPair{}, domain.Actor{}, err
	}
	pwHash, err := s.hasher.Hash(password)
	if err != nil {
		return TokenPair{}, domain.Actor{}, err
	}
	user, err := s.users.CreateUser(ctx, domain.User{
		ID:           uuid.NewString(),
		Email:        invite.Email,
		FirstName:    fn,
		LastName:     ln,
		PasswordHash: pwHash,
		Type:         domain.UserTypeUser,
	})
	if err != nil {
		return TokenPair{}, domain.Actor{}, err
	}
	pair, err := s.issuePair(ctx, user.ID, user.Email)
	if err != nil {
		return TokenPair{}, domain.Actor{}, err
	}
	return pair, domain.Actor{User: user}, nil
}
