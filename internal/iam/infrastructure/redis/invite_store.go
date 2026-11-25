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
	invitePrefix       = "iam:invite:"
	invitesPendingHash = "iam:invites:pending"
)

// InviteStore stores pending invitations exclusively in Redis.
type InviteStore struct{ client *sharedredis.Client }

// NewInviteStore creates a Redis-backed InviteStore.
func NewInviteStore(client *sharedredis.Client) *InviteStore { return &InviteStore{client: client} }

// CreateInvite stores a pending invitation under both token hash (with TTL) and pending hash.
func (s *InviteStore) CreateInvite(ctx context.Context, invite application.PendingInvite, ttl time.Duration) error {
	payload, err := json.Marshal(invite)
	if err != nil {
		return err
	}
	return s.client.Do(ctx, func() error {
		pipe := s.client.TxPipeline()
		pipe.Set(ctx, invitePrefix+invite.TokenHash, payload, ttl)
		pipe.HSet(ctx, invitesPendingHash, invite.ID, payload)
		_, err := pipe.Exec(ctx)
		return err
	})
}

// GetPending fetches a pending invitation by ID without modifying it.
func (s *InviteStore) GetPending(ctx context.Context, inviteID string) (application.PendingInvite, error) {
	return s.getPendingRaw(ctx, inviteID)
}

// VerifyInvite fetches a pending invitation by tokenHash without consuming it.
func (s *InviteStore) VerifyInvite(ctx context.Context, tokenHash string) (application.PendingInvite, error) {
	var raw []byte
	err := s.client.Do(ctx, func() error {
		var err error
		raw, err = s.client.Get(ctx, invitePrefix+tokenHash).Bytes()
		return err
	})
	if errors.Is(err, goredis.Nil) {
		return application.PendingInvite{}, domain.ErrInviteExpiredOrUsed
	}
	if err != nil {
		return application.PendingInvite{}, fmt.Errorf("redis verify invite: %w", err)
	}
	var inv application.PendingInvite
	if err := json.Unmarshal(raw, &inv); err != nil {
		return application.PendingInvite{}, domain.ErrInviteExpiredOrUsed
	}
	return inv, nil
}

// ConsumeInvite atomically pops the invitation using GETDEL so it can only be accepted once.
func (s *InviteStore) ConsumeInvite(ctx context.Context, tokenHash string) (application.PendingInvite, error) {
	var raw []byte
	err := s.client.Do(ctx, func() error {
		var err error
		raw, err = s.client.GetDel(ctx, invitePrefix+tokenHash).Bytes()
		return err
	})
	if errors.Is(err, goredis.Nil) {
		return application.PendingInvite{}, domain.ErrInviteExpiredOrUsed
	}
	if err != nil {
		return application.PendingInvite{}, fmt.Errorf("redis consume invite: %w", err)
	}
	var inv application.PendingInvite
	if err := json.Unmarshal(raw, &inv); err != nil {
		return application.PendingInvite{}, domain.ErrInviteExpiredOrUsed
	}
	if err := s.client.Do(ctx, func() error { return s.client.HDel(ctx, invitesPendingHash, inv.ID).Err() }); err != nil {
		slog.WarnContext(ctx, "failed to remove consumed invite from pending hash", "invite_id", inv.ID, "error", err)
	}
	return inv, nil
}
