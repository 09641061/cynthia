package rest

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go-service/internal/iam/application"
	"go-service/internal/iam/domain"
	sharedrest "go-service/internal/shared/rest"
)

const contextUserIDKey = "iam_user_id"

// AuthMiddleware validates the Bearer access token against Redis (fail-closed on any error).
func AuthMiddleware(sessions application.SessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		if sessions == nil {
			sharedrest.WriteProblem(c, http.StatusServiceUnavailable, "Service Unavailable", "Authentication service unavailable.")
			return
		}
		raw, ok := extractBearer(c.GetHeader("Authorization"))
		if !ok {
			sharedrest.WriteProblem(c, http.StatusUnauthorized, "Unauthorized", "Missing or invalid Bearer token.")
			return
		}
		sess, err := sessions.ValidateSession(c.Request.Context(), domain.HashTokenSHA256(raw))
		if err != nil {
			if errors.Is(err, domain.ErrInvalidToken) {
				sharedrest.WriteProblem(c, http.StatusUnauthorized, "Unauthorized", "Session is invalid or expired.")
				return
			}
			sharedrest.WriteProblem(c, http.StatusServiceUnavailable, "Service Unavailable", "Authentication service unavailable.")
			return
		}
		c.Set(contextUserIDKey, sess.UserID)
		c.Next()
	}
}

// RequirePermission enforces (userID, obj, act) via Casbin against live PostgreSQL state (fail-closed).
func RequirePermission(authz application.Authorizer, obj, act string) gin.HandlerFunc {
	return RequireAnyPermission(authz, [2]string{obj, act})
}

// RequireAnyPermission allows the request if the actor holds at least one of the listed (obj, act) pairs.
func RequireAnyPermission(authz application.Authorizer, pairs ...[2]string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if authz == nil {
			sharedrest.WriteProblem(c, http.StatusServiceUnavailable, "Service Unavailable", "Authorization service unavailable.")
			return
		}
		userID := ActorID(c)
		if userID == "" {
			sharedrest.WriteProblem(c, http.StatusUnauthorized, "Unauthorized", "Authentication required.")
			return
		}
		for _, pair := range pairs {
			allowed, err := authz.Enforce(c.Request.Context(), userID, pair[0], pair[1])
			if err != nil {
				sharedrest.WriteProblem(c, http.StatusServiceUnavailable, "Service Unavailable", "Authorization check failed.")
				return
			}
			if allowed {
				c.Next()
				return
			}
		}
		sharedrest.WriteProblem(c, http.StatusForbidden, "Forbidden", "You do not have permission to perform this action.")
	}
}

// ActorID extracts the authenticated user ID from the Gin context.
func ActorID(c *gin.Context) string {
	val, _ := c.Get(contextUserIDKey)
	id, _ := val.(string)
	return id
}

func extractBearer(header string) (string, bool) {
	parts := strings.SplitN(strings.TrimSpace(header), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", false
	}
	return strings.TrimSpace(parts[1]), true
}
