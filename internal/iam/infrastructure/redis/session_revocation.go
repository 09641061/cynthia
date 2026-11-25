package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go-service/internal/iam/application"
)

// RevokeSession deletes an access session and its paired refresh token from Redis.
func (s *SessionStore) RevokeSession(ctx context.Context, sessionHash string) error {
	var raw []byte
	err := s.client.Do(ctx, func() error {
		var err error
		raw, err = s.client.GetDel(ctx, sessionPrefix+sessionHash).Bytes()
		return err
	})
	if errors.Is(err, goredis.Nil) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("redis revoke session: %w", err)
	}
	var data application.SessionData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil
	}
	return s.client.Do(ctx, func() error {
		pipe := s.client.TxPipeline()
		remMembers := []any{"session:" + sessionHash}
		if data.RefreshHash != "" {
			pipe.Del(ctx, refreshPrefix+data.RefreshHash)
			remMembers = append(remMembers, "refresh:"+data.RefreshHash)
		}
		if data.UserID != "" {
			pipe.SRem(ctx, userSessionsPrefix+data.UserID, remMembers...)
		}
		_, err := pipe.Exec(ctx)
		return err
	})
}

// RevokeAllUserSessions deletes all tracked sessions and refresh tokens for userID.
func (s *SessionStore) RevokeAllUserSessions(ctx context.Context, userID string) error {
	userKey := userSessionsPrefix + userID
	return s.client.Do(ctx, func() error {
		members, err := s.client.SMembers(ctx, userKey).Result()
		if err != nil && !errors.Is(err, goredis.Nil) {
			return err
		}
		keys := make([]string, 0, len(members)+1)
		for _, m := range members {
			keys = append(keys, "iam:"+m)
		}
		keys = append(keys, userKey)
		return s.client.Del(ctx, keys...).Err()
	})
}

func ttlSeconds(d time.Duration) string {
	if secs := int64(d.Seconds()); secs >= 1 {
		return strconv.FormatInt(secs, 10)
	}
	return "1"
}
