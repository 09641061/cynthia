package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go-service/internal/iam/application"
	"go-service/internal/iam/domain"
	sharedredis "go-service/internal/shared/redis"
)

const (
	sessionPrefix      = "iam:session:"
	refreshPrefix      = "iam:refresh:"
	refreshUsedPrefix  = "iam:refresh_used:"
	userSessionsPrefix = "iam:user_sessions:"
)

// SessionStore stores sessions and refresh tokens exclusively in Redis.
type SessionStore struct{ client *sharedredis.Client }

// NewSessionStore creates a Redis-backed SessionStore.
func NewSessionStore(client *sharedredis.Client) *SessionStore { return &SessionStore{client: client} }

// CreateSessionPair stores a new access session, refresh token, and user revocation index in Redis.
func (s *SessionStore) CreateSessionPair(ctx context.Context, userID, email, sessionHash, refreshHash string, sessionTTL, refreshTTL time.Duration) error {
	now := time.Now().UTC()
	sessPayload, err := json.Marshal(application.SessionData{
		UserID: userID, Email: email, RefreshHash: refreshHash, ExpiresAt: now.Add(sessionTTL),
	})
	if err != nil {
		return err
	}
	refreshPayload, err := json.Marshal(application.SessionData{
		UserID: userID, Email: email, RefreshHash: refreshHash, ExpiresAt: now.Add(refreshTTL),
	})
	if err != nil {
		return err
	}
	return s.client.Do(ctx, func() error {
		pipe := s.client.TxPipeline()
		pipe.Set(ctx, sessionPrefix+sessionHash, sessPayload, sessionTTL)
		pipe.Set(ctx, refreshPrefix+refreshHash, refreshPayload, refreshTTL)
		userKey := userSessionsPrefix + userID
		pipe.SAdd(ctx, userKey, "session:"+sessionHash, "refresh:"+refreshHash)
		pipe.Expire(ctx, userKey, refreshTTL)
		_, err := pipe.Exec(ctx)
		return err
	})
}

// ValidateSession loads and validates an access session from Redis.
func (s *SessionStore) ValidateSession(ctx context.Context, sessionHash string) (application.SessionData, error) {
	var raw []byte
	err := s.client.Do(ctx, func() error {
		var err error
		raw, err = s.client.Get(ctx, sessionPrefix+sessionHash).Bytes()
		return err
	})
	if errors.Is(err, goredis.Nil) {
		return application.SessionData{}, domain.ErrInvalidToken
	}
	if err != nil {
		return application.SessionData{}, fmt.Errorf("redis validate session: %w", err)
	}
	var data application.SessionData
	if err := json.Unmarshal(raw, &data); err != nil {
		return application.SessionData{}, domain.ErrInvalidToken
	}
	return data, nil
}

// RotateRefreshToken atomically consumes oldRefreshHash, detects reuse, and issues newSessionHash + newRefreshHash.
func (s *SessionStore) RotateRefreshToken(ctx context.Context, oldRefreshHash, newSessionHash, newRefreshHash string, sessionTTL, refreshTTL time.Duration) (string, error) {
	now := time.Now().UTC()
	keys := []string{
		refreshPrefix + oldRefreshHash, refreshPrefix + newRefreshHash,
		sessionPrefix + newSessionHash, refreshUsedPrefix + oldRefreshHash,
	}
	var res []string
	err := s.client.Do(ctx, func() error {
		var err error
		res, err = rotateRefreshScript.Run(
			ctx, s.client.Client, keys,
			ttlSeconds(sessionTTL), ttlSeconds(refreshTTL),
			"session:"+newSessionHash, "refresh:"+newRefreshHash, newRefreshHash,
			now.Add(sessionTTL).Format(time.RFC3339Nano),
			now.Add(refreshTTL).Format(time.RFC3339Nano),
		).StringSlice()
		return err
	})
	if errors.Is(err, goredis.Nil) {
		return "", domain.ErrInvalidToken
	}
	if err != nil {
		return "", fmt.Errorf("redis rotate refresh token: %w", err)
	}
	if len(res) != 2 {
		return "", domain.ErrInvalidToken
	}
	if res[0] == "REUSED" {
		if revokeErr := s.RevokeAllUserSessions(ctx, res[1]); revokeErr != nil {
			slog.WarnContext(ctx, "failed to revoke user sessions after refresh reuse", "user_id", res[1], "error", revokeErr)
		}
		return "", domain.ErrInvalidToken
	}
	return res[1], nil
}
