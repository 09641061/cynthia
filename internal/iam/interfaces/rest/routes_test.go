package rest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-service/internal/iam/application"
	"go-service/internal/iam/domain"
)

type stubSessionStore struct {
	userByHash map[string]application.SessionData
	err        error
}

func (s stubSessionStore) CreateSessionPair(context.Context, string, string, string, string, time.Duration, time.Duration) error {
	return nil
}
func (s stubSessionStore) ValidateSession(_ context.Context, h string) (application.SessionData, error) {
	if s.err != nil {
		return application.SessionData{}, s.err
	}
	sess, ok := s.userByHash[h]
	if !ok {
		return application.SessionData{}, domain.ErrInvalidToken
	}
	return sess, nil
}
func (s stubSessionStore) RotateRefreshToken(context.Context, string, string, string, time.Duration, time.Duration) (string, error) {
	return "", nil
}
func (s stubSessionStore) RevokeSession(context.Context, string) error         { return nil }
func (s stubSessionStore) RevokeAllUserSessions(context.Context, string) error { return nil }

type stubAuthorizer struct {
	allow map[string]bool
	err   error
}

func (a stubAuthorizer) Enforce(_ context.Context, userID, obj, act string) (bool, error) {
	if a.err != nil {
		return false, a.err
	}
	return a.allow[userID+":"+obj+":"+act], nil
}

func TestAuthAndPermissionMiddlewareFailClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.NewString()
	rawTok := "test-token"
	sum := sha256.Sum256([]byte(rawTok))
	tokHash := hex.EncodeToString(sum[:])

	sessions := stubSessionStore{userByHash: map[string]application.SessionData{tokHash: {UserID: userID}}}
	authz := stubAuthorizer{allow: map[string]bool{userID + ":iam:manage_roles": true}}

	r := gin.New()
	r.GET("/protected", AuthMiddleware(sessions), RequirePermission(authz, "iam", "manage_roles"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	r.GET("/forbidden", AuthMiddleware(sessions), RequirePermission(authz, "iam", "manage_users"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Missing Bearer -> 401
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing token = %d, want 401", rec.Code)
	}

	// Valid token + permission -> 200
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+rawTok)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("authorized = %d, want 200", rec.Code)
	}

	// Valid token without permission -> 403
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/forbidden", nil)
	req.Header.Set("Authorization", "Bearer "+rawTok)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("unauthorized = %d, want 403", rec.Code)
	}

	// Redis error -> 503 fail-closed
	rErr := gin.New()
	rErr.GET("/protected", AuthMiddleware(stubSessionStore{err: errors.New("redis down")}), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+rawTok)
	rErr.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("redis error = %d, want 503", rec.Code)
	}
}

func TestMemberUUIDValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.NewString()
	rawTok := "tok"
	sum := sha256.Sum256([]byte(rawTok))
	sessions := stubSessionStore{userByHash: map[string]application.SessionData{hex.EncodeToString(sum[:]): {UserID: userID}}}
	authz := stubAuthorizer{allow: map[string]bool{userID + ":iam:manage_users": true}}

	r := gin.New()
	RegisterRoutes(r.Group("/v1"), sessions, authz, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/v1/iam/members/not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+rawTok)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid uuid = %d, want 400", rec.Code)
	}
}
