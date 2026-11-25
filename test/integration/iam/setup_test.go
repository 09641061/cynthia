//go:build integration

package iam_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"go-service/internal/iam"
	"go-service/internal/iam/infrastructure/crypto"
	"go-service/internal/shared/circuitbreaker"
	sharedpostgres "go-service/internal/shared/postgres"
	sharedredis "go-service/internal/shared/redis"
)

type testEnv struct {
	db              *sharedpostgres.Client
	redis           *goredis.Client
	router          *gin.Engine
	ownerID         string
	lastInviteToken string
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	parsed, err := url.Parse(testDatabaseURL)
	require.NoError(t, err)
	admin, err := sql.Open("pgx", parsed.String())
	require.NoError(t, err)
	ctx := t.Context()
	require.NoError(t, admin.PingContext(ctx))

	schema := "iam_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = admin.ExecContext(ctx, `CREATE SCHEMA `+schema)
	require.NoError(t, err)

	query := parsed.Query()
	query.Set("search_path", schema+",public")
	parsed.RawQuery = query.Encode()
	sqlDB, err := sql.Open("pgx", parsed.String())
	require.NoError(t, err)
	require.NoError(t, sharedpostgres.Migrate(ctx, sqlDB, "../../../migrations"))

	rdb := goredis.NewClient(&goredis.Options{Addr: testRedisAddr})
	require.NoError(t, rdb.FlushDB(ctx).Err())

	t.Cleanup(func() {
		_ = rdb.Close()
		_ = sqlDB.Close()
		_, _ = admin.ExecContext(context.Background(), `DROP SCHEMA `+schema+` CASCADE`)
		_ = admin.Close()
	})

	ownerID := seedUser(t, ctx, sqlDB, "lucia@estudionube.mx", "TakoduDev2026!", "owner")
	breaker, err := circuitbreaker.New(5, 30*time.Second)
	require.NoError(t, err)
	pgClient := sharedpostgres.NewClient(sqlDB, breaker)
	env := &testEnv{db: pgClient, redis: rdb, ownerID: ownerID}
	tokenRe := regexp.MustCompile(`token=([^"&]+)`)
	resendSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p struct {
			HTML string `json:"html"`
		}
		_ = json.NewDecoder(r.Body).Decode(&p)
		if m := tokenRe.FindStringSubmatch(p.HTML); len(m) == 2 {
			env.lastInviteToken, _ = url.QueryUnescape(m[1])
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(resendSrv.Close)

	router := gin.New()
	mod, err := iam.NewModule(pgClient, sharedredis.NewClient(rdb, breaker), router, iam.ModuleConfig{
		ResendAPIKey:  "re_test_key",
		ResendFrom:    "noreply@cynthia.test",
		ResendBaseURL: resendSrv.URL,
		AppURL:        "https://app.cynthia.test",
		SessionTTL:    30 * time.Minute,
		RefreshTTL:    24 * time.Hour,
		InviteTTL:     72 * time.Hour,
	})
	require.NoError(t, err)
	router.GET("/v1/catalog/probe", mod.AuthMiddleware(), mod.RequirePermission("catalog", "manage"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	env.router = router
	return env
}

func seedUser(t *testing.T, ctx context.Context, db *sql.DB, email, password, userType string) string {
	t.Helper()
	id := uuid.NewString()
	hash, err := crypto.NewArgon2idHasher().Hash(password)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO iam_users (id, email, first_name, last_name, password_hash, type)
		VALUES ($1, $2, 'Test', 'User', $3, $4)`, id, email, hash, userType)
	require.NoError(t, err)
	return id
}

func (e *testEnv) doJSON(t *testing.T, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

func (e *testEnv) loginAs(t *testing.T, email, password string) (string, string) {
	t.Helper()
	w := e.doJSON(t, http.MethodPost, "/v1/auth/login", "", map[string]string{
		"email": email, "password": password,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	return out.AccessToken, out.RefreshToken
}
