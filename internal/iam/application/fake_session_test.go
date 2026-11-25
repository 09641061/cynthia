package application

import (
	"context"
	"maps"
	"time"

	"go-service/internal/iam/domain"
)

type fakeSessionStore struct {
	sessions map[string]SessionData
	refresh  map[string]string
	revoked  []string
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{sessions: map[string]SessionData{}, refresh: map[string]string{}}
}

func (f *fakeSessionStore) CreateSessionPair(_ context.Context, userID, email, sessHash, refHash string, ttl, _ time.Duration) error {
	f.sessions[sessHash] = SessionData{UserID: userID, Email: email, ExpiresAt: time.Now().Add(ttl)}
	f.refresh[refHash] = userID
	return nil
}

func (f *fakeSessionStore) ValidateSession(_ context.Context, sessHash string) (SessionData, error) {
	s, ok := f.sessions[sessHash]
	if !ok {
		return SessionData{}, domain.ErrInvalidToken
	}
	return s, nil
}

func (f *fakeSessionStore) RotateRefreshToken(_ context.Context, oldRef, newSess, newRef string, ttl, _ time.Duration) (string, error) {
	uid, ok := f.refresh[oldRef]
	if !ok {
		return "", domain.ErrInvalidToken
	}
	delete(f.refresh, oldRef)
	f.refresh[newRef] = uid
	f.sessions[newSess] = SessionData{UserID: uid, ExpiresAt: time.Now().Add(ttl)}
	return uid, nil
}

func (f *fakeSessionStore) RevokeSession(_ context.Context, sessHash string) error {
	delete(f.sessions, sessHash)
	return nil
}

func (f *fakeSessionStore) RevokeAllUserSessions(_ context.Context, userID string) error {
	f.revoked = append(f.revoked, userID)
	maps.DeleteFunc(f.sessions, func(_ string, v SessionData) bool { return v.UserID == userID })
	return nil
}
