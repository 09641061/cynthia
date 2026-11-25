package iam

import (
	"time"

	"github.com/gin-gonic/gin"
	"go-service/internal/iam/application"
	"go-service/internal/iam/infrastructure/casbin"
	"go-service/internal/iam/infrastructure/crypto"
	"go-service/internal/iam/infrastructure/email"
	"go-service/internal/iam/infrastructure/postgres"
	iamredis "go-service/internal/iam/infrastructure/redis"
	"go-service/internal/iam/interfaces/rest"
	sharedpostgres "go-service/internal/shared/postgres"
	sharedredis "go-service/internal/shared/redis"
)

// ModuleConfig holds runtime settings for the IAM module.
type ModuleConfig struct {
	ResendAPIKey  string
	ResendFrom    string
	ResendBaseURL string
	AppURL        string
	SessionTTL    time.Duration
	RefreshTTL    time.Duration
	InviteTTL     time.Duration
}

// Module exposes IAM middleware and permission guards to the composition root.
type Module struct {
	sessions application.SessionStore
	authz    application.Authorizer
}

// NewModule initializes the IAM module, registers its routes, and returns middleware handles.
func NewModule(db *sharedpostgres.Client, redisClient *sharedredis.Client, router gin.IRouter, cfg ModuleConfig) (*Module, error) {
	users := postgres.NewUserStore(db)
	roles := postgres.NewRoleStore(db)

	var sessions application.SessionStore
	var invites application.InviteStore
	if redisClient != nil {
		sessions = iamredis.NewSessionStore(redisClient)
		invites = iamredis.NewInviteStore(redisClient)
	}
	authz, err := casbin.NewAuthorizer(users)
	if err != nil {
		return nil, err
	}
	hasher := crypto.NewArgon2idHasher()
	mailer := email.NewResendSender(cfg.ResendAPIKey, cfg.ResendFrom, cfg.AppURL)
	mailer.SetBaseURL(cfg.ResendBaseURL)

	authSvc := application.NewAuthService(users, sessions, invites, hasher, cfg.SessionTTL, cfg.RefreshTTL)
	memberSvc := application.NewMemberService(users, roles, sessions)
	roleSvc := application.NewRoleService(users, roles)
	inviteSvc := application.NewInviteService(users, invites, mailer, cfg.InviteTTL)

	rest.RegisterRoutes(router.Group("/v1"), sessions, authz, authSvc, memberSvc, roleSvc, inviteSvc)
	return &Module{sessions: sessions, authz: authz}, nil
}

// AuthMiddleware returns the Gin middleware that validates Redis bearer sessions.
func (m *Module) AuthMiddleware() gin.HandlerFunc {
	return rest.AuthMiddleware(m.sessions)
}

// RequirePermission returns the Gin middleware that enforces a Casbin permission.
func (m *Module) RequirePermission(obj, act string) gin.HandlerFunc {
	return rest.RequirePermission(m.authz, obj, act)
}
