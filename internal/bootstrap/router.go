package bootstrap

import (
	"github.com/gin-gonic/gin"
	"go-service/internal/calendar"
	calendaracl "go-service/internal/calendar/infrastructure/acl"
	"go-service/internal/catalog"
	"go-service/internal/crm"
	"go-service/internal/iam"
	"go-service/internal/organizations"
	"go-service/internal/shared/postgres"
	sharedredis "go-service/internal/shared/redis"
)

func newRouter(db *postgres.Client, redisClient *sharedredis.Client, iamCfg iam.ModuleConfig, waCfg crm.WhatsAppConfig) (*gin.Engine, error) {
	router := gin.Default()
	iamModule, err := iam.NewModule(db, redisClient, router, iamCfg)
	if err != nil {
		return nil, err
	}
	authMW := iamModule.AuthMiddleware()

	catalogGroup := router.Group("")
	catalogGroup.Use(authMW, iamModule.RequirePermission("catalog", "manage"))
	catalog.NewModule(db, catalogGroup)

	crmGroup := router.Group("")
	crmGroup.Use(skipWhatsAppWebhook(authMW, iamModule.RequirePermission("crm", "manage")))
	crm.NewModule(db, crmGroup, organizations.NewWhatsAppConfigReader(db), waCfg)

	orgGroup := router.Group("")
	orgGroup.Use(authMW, iamModule.RequirePermission("organization", "manage"))
	organizations.NewModule(db, orgGroup)

	calGroup := router.Group("")
	calGroup.Use(authMW, iamModule.RequirePermission("calendar", "manage"))
	calendar.NewModule(
		db,
		calGroup,
		calendaracl.NewPersonAdapter(iam.NewPersonReader(db)),
		crm.NewCustomerReader(db),
		catalog.NewServiceReader(db),
		organizations.NewScheduleReader(db),
	)

	router.StaticFile("/openapi.yaml", "api/openapi.yaml")
	return router, nil
}

func skipWhatsAppWebhook(handlers ...gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.FullPath() == "/v1/crm/webhooks/whatsapp" {
			c.Next()
			return
		}
		for _, h := range handlers {
			h(c)
			if c.IsAborted() {
				return
			}
		}
	}
}
