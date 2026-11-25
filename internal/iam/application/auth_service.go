package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"go-service/internal/iam/domain"
)

// AuthService handles login, token refresh, logout, self-service profile, and invitation onboarding.
type AuthService struct {
	users      UserStore
	sessions   SessionStore
	invites    InviteStore
	hasher     PasswordHasher
	sessionTTL time.Duration
	refreshTTL time.Duration
}

// NewAuthService constructs an AuthService.
func NewAuthService(users UserStore, sessions SessionStore, invites InviteStore, hasher PasswordHasher, sessionTTL, refreshTTL time.Duration) *AuthService {
	return &AuthService{
		users: users, sessions: sessions, invites: invites, hasher: hasher,
		sessionTTL: sessionTTL, refreshTTL: refreshTTL,
	}
}

// Login authenticates email/password and issues a Redis-backed session + refresh token pair.
func (s *AuthService) Login(ctx context.Context, email, password string) (TokenPair, domain.Actor, error) {
	normEmail, err := domain.NormalizeEmail(email)
	if err != nil || password == "" {
		return TokenPair{}, domain.Actor{}, domain.ErrInvalidCredentials
	}
	user, err := s.users.GetUserByEmail(ctx, normEmail)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return TokenPair{}, domain.Actor{}, domain.ErrInvalidCredentials
		}
		return TokenPair{}, domain.Actor{}, err
	}
	ok, err := s.hasher.Verify(password, user.PasswordHash)
	if err != nil || !ok {
		return TokenPair{}, domain.Actor{}, domain.ErrInvalidCredentials
	}
	actor, err := s.users.GetActorByID(ctx, user.ID)
	if err != nil {
		return TokenPair{}, domain.Actor{}, err
	}
	pair, err := s.issuePair(ctx, actor.User.ID, actor.User.Email)
	return pair, actor, err
}

// Refresh atomically rotates a refresh token and issues a new session + refresh token pair.
func (s *AuthService) Refresh(ctx context.Context, rawRefresh string) (TokenPair, error) {
	if strings.TrimSpace(rawRefresh) == "" {
		return TokenPair{}, domain.ErrInvalidToken
	}
	newSessionRaw, newSessionHash, err := domain.GenerateSecureToken()
	if err != nil {
		return TokenPair{}, err
	}
	newRefreshRaw, newRefreshHash, err := domain.GenerateSecureToken()
	if err != nil {
		return TokenPair{}, err
	}
	_, err = s.sessions.RotateRefreshToken(
		ctx, domain.HashTokenSHA256(rawRefresh), newSessionHash, newRefreshHash, s.sessionTTL, s.refreshTTL,
	)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken:  newSessionRaw,
		RefreshToken: newRefreshRaw,
		ExpiresIn:    int64(s.sessionTTL.Seconds()),
	}, nil
}

// Logout revokes the current access session in Redis.
func (s *AuthService) Logout(ctx context.Context, rawAccess string) error {
	if rawAccess == "" {
		return nil
	}
	return s.sessions.RevokeSession(ctx, domain.HashTokenSHA256(rawAccess))
}

// Me returns the authenticated actor's current profile, roles, and effective permissions.
func (s *AuthService) Me(ctx context.Context, userID string) (domain.Actor, error) {
	return s.users.GetActorByID(ctx, userID)
}

// UpdateMe updates the authenticated user's own name and/or password.
func (s *AuthService) UpdateMe(ctx context.Context, userID, firstName, lastName, password string) (domain.Actor, error) {
	fn, ln, err := domain.ValidateNameParts(firstName, lastName)
	if err != nil {
		return domain.Actor{}, err
	}
	var passwordHash string
	if password != "" {
		if len(password) < 8 {
			return domain.Actor{}, domain.ErrValidation
		}
		passwordHash, err = s.hasher.Hash(password)
		if err != nil {
			return domain.Actor{}, err
		}
	}
	if _, err := s.users.UpdateProfile(ctx, userID, fn, ln, passwordHash); err != nil {
		return domain.Actor{}, err
	}
	return s.users.GetActorByID(ctx, userID)
}

func (s *AuthService) issuePair(ctx context.Context, userID, email string) (TokenPair, error) {
	accessRaw, accessHash, err := domain.GenerateSecureToken()
	if err != nil {
		return TokenPair{}, err
	}
	refreshRaw, refreshHash, err := domain.GenerateSecureToken()
	if err != nil {
		return TokenPair{}, err
	}
	if err := s.sessions.CreateSessionPair(ctx, userID, email, accessHash, refreshHash, s.sessionTTL, s.refreshTTL); err != nil {
		return TokenPair{}, err
	}
	return TokenPair{AccessToken: accessRaw, RefreshToken: refreshRaw, ExpiresIn: int64(s.sessionTTL.Seconds())}, nil
}
