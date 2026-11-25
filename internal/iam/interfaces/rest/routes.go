package rest

import (
	"github.com/gin-gonic/gin"
	"go-service/internal/iam/application"
)

// RegisterRoutes mounts public auth, self-service, and Casbin-protected IAM routes under router.
func RegisterRoutes(
	router gin.IRouter,
	sessions application.SessionStore,
	authz application.Authorizer,
	authSvc *application.AuthService,
	memberSvc *application.MemberService,
	roleSvc *application.RoleService,
	inviteSvc *application.InviteService,
) {
	ah := authHandler{auth: authSvc}
	mh := memberHandler{members: memberSvc}
	rh := roleHandler{roles: roleSvc}
	ih := inviteHandler{invites: inviteSvc}

	// Public auth routes
	router.POST("/auth/login", ah.login)
	router.POST("/auth/refresh", ah.refresh)
	router.GET("/auth/invitations/verify", ah.verifyInvite)
	router.POST("/auth/invitations/accept", ah.acceptInvite)

	authMW := AuthMiddleware(sessions)

	// Self-service authenticated routes
	authed := router.Group("")
	authed.Use(authMW)
	authed.GET("/auth/me", ah.me)
	authed.PATCH("/auth/me", ah.updateMe)
	authed.POST("/auth/logout", ah.logout)

	// IAM routes (Casbin-protected)
	iamGroup := authed.Group("/iam")
	iamGroup.GET("/members", RequireAnyPermission(authz, [2]string{"iam", "manage_users"}, [2]string{"iam", "manage_roles"}), mh.list)
	iamGroup.DELETE("/members/:id", RequirePermission(authz, "iam", "manage_users"), mh.delete)
	iamGroup.PUT("/members/:id/roles", RequirePermission(authz, "iam", "manage_roles"), mh.assignRoles)

	iamGroup.GET("/roles", RequireAnyPermission(authz, [2]string{"iam", "manage_roles"}, [2]string{"iam", "manage_users"}), rh.list)
	iamGroup.POST("/roles", RequirePermission(authz, "iam", "manage_roles"), rh.create)
	iamGroup.PATCH("/roles/:id", RequirePermission(authz, "iam", "manage_roles"), rh.update)
	iamGroup.DELETE("/roles/:id", RequirePermission(authz, "iam", "manage_roles"), rh.delete)
	iamGroup.PUT("/roles/reorder", RequirePermission(authz, "iam", "manage_roles"), rh.reorder)

	iamGroup.GET("/invitations", RequirePermission(authz, "iam", "manage_users"), ih.list)
	iamGroup.POST("/invitations", RequirePermission(authz, "iam", "manage_users"), ih.create)
	iamGroup.DELETE("/invitations/:id", RequirePermission(authz, "iam", "manage_users"), ih.delete)
	iamGroup.POST("/invitations/:id/resend", RequirePermission(authz, "iam", "manage_users"), ih.resend)
}
