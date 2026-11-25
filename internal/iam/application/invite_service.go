package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"go-service/internal/iam/domain"
)

// InviteService manages pending member invitations in Redis and email delivery via Resend.
type InviteService struct {
	users     UserStore
	invites   InviteStore
	mailer    EmailSender
	inviteTTL time.Duration
}

// NewInviteService constructs an InviteService.
func NewInviteService(users UserStore, invites InviteStore, mailer EmailSender, inviteTTL time.Duration) *InviteService {
	return &InviteService{users: users, invites: invites, mailer: mailer, inviteTTL: inviteTTL}
}

// ListPending returns all currently pending invitations.
func (s *InviteService) ListPending(ctx context.Context) ([]PendingInvite, error) {
	return s.invites.ListPending(ctx)
}

// CreateInvitation creates a pending invitation in Redis and sends the email via Resend.
func (s *InviteService) CreateInvitation(ctx context.Context, actorID, email, firstName, lastName string) (PendingInvite, error) {
	actor, err := s.users.GetActorByID(ctx, actorID)
	if err != nil {
		return PendingInvite{}, err
	}
	if !actor.CanManageUsers() {
		return PendingInvite{}, domain.ErrForbidden
	}
	normEmail, err := domain.NormalizeEmail(email)
	if err != nil {
		return PendingInvite{}, err
	}
	fn := strings.TrimSpace(firstName)
	ln := strings.TrimSpace(lastName)
	if len(fn) > 80 || len(ln) > 80 {
		return PendingInvite{}, domain.ErrValidation
	}
	if _, err := s.users.GetUserByEmail(ctx, normEmail); err == nil {
		return PendingInvite{}, domain.ErrDuplicateEmail
	} else if !errors.Is(err, domain.ErrUserNotFound) {
		return PendingInvite{}, err
	}
	pending, err := s.invites.ListPending(ctx)
	if err != nil {
		return PendingInvite{}, err
	}
	for _, p := range pending {
		if p.Email == normEmail {
			return PendingInvite{}, domain.ErrDuplicateInvitation
		}
	}
	rawToken, tokenHash, err := domain.GenerateSecureToken()
	if err != nil {
		return PendingInvite{}, err
	}
	now := time.Now().UTC()
	invite := PendingInvite{
		ID:        uuid.NewString(),
		Email:     normEmail,
		FirstName: fn,
		LastName:  ln,
		TokenHash: tokenHash,
		InvitedBy: actorID,
		CreatedAt: now,
		ExpiresAt: now.Add(s.inviteTTL),
	}
	if err := s.invites.CreateInvite(ctx, invite, s.inviteTTL); err != nil {
		return PendingInvite{}, err
	}
	if s.mailer != nil {
		if err := s.mailer.SendInvitation(ctx, invite.Email, invite.FirstName, rawToken); err != nil {
			_ = s.invites.DeletePending(ctx, invite.ID)
			return PendingInvite{}, err
		}
	}
	return invite, nil
}

// DeleteInvitation cancels a pending invitation so it can no longer be accepted.
func (s *InviteService) DeleteInvitation(ctx context.Context, actorID, inviteID string) error {
	actor, err := s.users.GetActorByID(ctx, actorID)
	if err != nil {
		return err
	}
	if !actor.CanManageUsers() {
		return domain.ErrForbidden
	}
	return s.invites.DeletePending(ctx, inviteID)
}

// ResendInvitation sends a fresh invitation email and commits the new token only after delivery succeeds.
func (s *InviteService) ResendInvitation(ctx context.Context, actorID, inviteID string) (PendingInvite, error) {
	actor, err := s.users.GetActorByID(ctx, actorID)
	if err != nil {
		return PendingInvite{}, err
	}
	if !actor.CanManageUsers() {
		return PendingInvite{}, domain.ErrForbidden
	}
	existing, err := s.invites.GetPending(ctx, inviteID)
	if err != nil {
		return PendingInvite{}, err
	}
	rawToken, tokenHash, err := domain.GenerateSecureToken()
	if err != nil {
		return PendingInvite{}, err
	}
	if s.mailer != nil {
		if err := s.mailer.SendInvitation(ctx, existing.Email, existing.FirstName, rawToken); err != nil {
			return PendingInvite{}, err
		}
	}
	return s.invites.ResendPending(ctx, inviteID, tokenHash, s.inviteTTL)
}
