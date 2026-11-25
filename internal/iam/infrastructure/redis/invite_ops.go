package redis

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sort"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go-service/internal/iam/application"
	"go-service/internal/iam/domain"
)

// ListPending returns all non-expired pending invitations ordered newest first.
func (s *InviteStore) ListPending(ctx context.Context) ([]application.PendingInvite, error) {
	var vals []string
	if err := s.client.Do(ctx, func() error {
		var err error
		vals, err = s.client.HVals(ctx, invitesPendingHash).Result()
		return err
	}); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	out := make([]application.PendingInvite, 0, len(vals))
	for _, v := range vals {
		var inv application.PendingInvite
		if err := json.Unmarshal([]byte(v), &inv); err != nil {
			continue
		}
		if now.After(inv.ExpiresAt) {
			if err := s.client.Do(ctx, func() error { return s.client.HDel(ctx, invitesPendingHash, inv.ID).Err() }); err != nil {
				slog.WarnContext(ctx, "failed to prune expired invite", "invite_id", inv.ID, "error", err)
			}
			continue
		}
		out = append(out, inv)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

// DeletePending cancels a pending invitation by removing both its hash entry and token key.
func (s *InviteStore) DeletePending(ctx context.Context, inviteID string) error {
	inv, err := s.getPendingRaw(ctx, inviteID)
	if err != nil {
		return err
	}
	return s.client.Do(ctx, func() error {
		pipe := s.client.TxPipeline()
		pipe.Del(ctx, invitePrefix+inv.TokenHash)
		pipe.HDel(ctx, invitesPendingHash, inviteID)
		_, err := pipe.Exec(ctx)
		return err
	})
}

// ResendPending replaces the invitation token hash and extends its TTL.
func (s *InviteStore) ResendPending(ctx context.Context, inviteID, newTokenHash string, ttl time.Duration) (application.PendingInvite, error) {
	inv, err := s.getPendingRaw(ctx, inviteID)
	if err != nil {
		return application.PendingInvite{}, err
	}
	oldHash := inv.TokenHash
	inv.TokenHash = newTokenHash
	inv.ExpiresAt = time.Now().UTC().Add(ttl)
	updated, err := json.Marshal(inv)
	if err != nil {
		return application.PendingInvite{}, err
	}
	err = s.client.Do(ctx, func() error {
		pipe := s.client.TxPipeline()
		pipe.Del(ctx, invitePrefix+oldHash)
		pipe.Set(ctx, invitePrefix+newTokenHash, updated, ttl)
		pipe.HSet(ctx, invitesPendingHash, inviteID, updated)
		_, err := pipe.Exec(ctx)
		return err
	})
	return inv, err
}

func (s *InviteStore) getPendingRaw(ctx context.Context, inviteID string) (application.PendingInvite, error) {
	var raw []byte
	err := s.client.Do(ctx, func() error {
		var err error
		raw, err = s.client.HGet(ctx, invitesPendingHash, inviteID).Bytes()
		return err
	})
	if errors.Is(err, goredis.Nil) {
		return application.PendingInvite{}, domain.ErrInvitationNotFound
	}
	if err != nil {
		return application.PendingInvite{}, err
	}
	var inv application.PendingInvite
	if err := json.Unmarshal(raw, &inv); err != nil {
		if delErr := s.client.Do(ctx, func() error { return s.client.HDel(ctx, invitesPendingHash, inviteID).Err() }); delErr != nil {
			slog.WarnContext(ctx, "failed to prune corrupt invite", "invite_id", inviteID, "error", delErr)
		}
		return application.PendingInvite{}, domain.ErrInvitationNotFound
	}
	return inv, nil
}
