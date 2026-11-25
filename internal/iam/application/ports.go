package application

import (
	"context"
	"time"

	"go-service/internal/iam/domain"
)

// SessionData represents an active user session stored in Redis.
type SessionData struct {
	UserID      string    `json:"user_id"`
	Email       string    `json:"email"`
	RefreshHash string    `json:"refresh_hash,omitempty"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// TokenPair is returned by login, refresh, and invitation acceptance.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

// PendingInvite represents an invitation stored in Redis.
type PendingInvite struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	TokenHash string    `json:"token_hash"`
	InvitedBy string    `json:"invited_by"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// UserStore persists IAM users and their role memberships in PostgreSQL.
type UserStore interface {
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	GetActorByID(ctx context.Context, userID string) (domain.Actor, error)
	ListActors(ctx context.Context) ([]domain.Actor, error)
	CreateUser(ctx context.Context, user domain.User) (domain.User, error)
	UpdateProfile(ctx context.Context, userID, firstName, lastName, passwordHash string) (domain.User, error)
	DeleteMember(ctx context.Context, userID string) error
	ReplaceMemberRoles(ctx context.Context, userID string, roleIDs []string) (domain.Actor, error)
}

// RoleStore persists IAM roles and hierarchy positions in PostgreSQL.
type RoleStore interface {
	ListRoles(ctx context.Context) ([]domain.Role, error)
	GetRole(ctx context.Context, roleID string) (domain.Role, error)
	CountRoles(ctx context.Context) (int, error)
	CreateRole(ctx context.Context, role domain.Role) (domain.Role, error)
	UpdateRole(ctx context.Context, roleID, name string, perms []string) (domain.Role, error)
	DeleteRole(ctx context.Context, roleID string) error
	ReorderRoles(ctx context.Context, orderedIDs []string) ([]domain.Role, error)
}

// SessionStore manages sessions and refresh token rotation exclusively in Redis.
type SessionStore interface {
	CreateSessionPair(ctx context.Context, userID, email, sessionHash, refreshHash string, sessionTTL, refreshTTL time.Duration) error
	ValidateSession(ctx context.Context, sessionHash string) (SessionData, error)
	RotateRefreshToken(ctx context.Context, oldRefreshHash, newSessionHash, newRefreshHash string, sessionTTL, refreshTTL time.Duration) (string, error)
	RevokeSession(ctx context.Context, sessionHash string) error
	RevokeAllUserSessions(ctx context.Context, userID string) error
}

// InviteStore manages pending invitations exclusively in Redis.
type InviteStore interface {
	CreateInvite(ctx context.Context, invite PendingInvite, ttl time.Duration) error
	GetPending(ctx context.Context, inviteID string) (PendingInvite, error)
	VerifyInvite(ctx context.Context, tokenHash string) (PendingInvite, error)
	ConsumeInvite(ctx context.Context, tokenHash string) (PendingInvite, error)
	ListPending(ctx context.Context) ([]PendingInvite, error)
	DeletePending(ctx context.Context, inviteID string) error
	ResendPending(ctx context.Context, inviteID, newTokenHash string, ttl time.Duration) (PendingInvite, error)
}

// PasswordHasher hashes and verifies passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, encoded string) (bool, error)
}

// Authorizer enforces Casbin permissions against current PostgreSQL state.
type Authorizer interface {
	Enforce(ctx context.Context, userID, obj, act string) (bool, error)
}

// EmailSender sends transactional invitation emails.
type EmailSender interface {
	SendInvitation(ctx context.Context, toEmail, firstName, rawToken string) error
}
