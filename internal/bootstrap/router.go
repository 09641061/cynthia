package bootstrap

import (
	"context"

	"github.com/gin-gonic/gin"
	"go-service/internal/assistant"
	"go-service/internal/bootstrap/config"
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
	router, _, err := buildApp(db, redisClient, iamCfg, waCfg, assistant.Config{})
	return router, err
}

func newApp(db *postgres.Client, redisClient *sharedredis.Client, cfg config.Config) (*gin.Engine, func(context.Context), error) {
	iamCfg := iam.ModuleConfig{
		ResendAPIKey: cfg.ResendAPIKey,
		ResendFrom:   cfg.ResendFrom,
		AppURL:       cfg.AppURL,
		SessionTTL:   cfg.SessionTTL,
		RefreshTTL:   cfg.RefreshTTL,
		InviteTTL:    cfg.InviteTTL,
	}
	waCfg := crm.WhatsAppConfig{
		AccessToken:     cfg.WhatsAppAccessToken,
		PhoneNumberID:   cfg.WhatsAppPhoneNumberID,
		GraphAPIVersion: cfg.WhatsAppGraphAPIVersion,
		VerifyToken:     cfg.WhatsAppVerifyToken,
		AppSecret:       cfg.WhatsAppAppSecret,
	}
	assistantCfg := assistant.Config{
		DeepSeekBaseURL: cfg.DeepSeekBaseURL,
		DeepSeekAPIKey:  cfg.DeepSeekAPIKey,
		DeepSeekModel:   cfg.DeepSeekModel,
		Workers:         cfg.AssistantWorkers,
	}
	return buildApp(db, redisClient, iamCfg, waCfg, assistantCfg)
}

func buildApp(db *postgres.Client, redisClient *sharedredis.Client, iamCfg iam.ModuleConfig, waCfg crm.WhatsAppConfig, assistantCfg assistant.Config) (*gin.Engine, func(context.Context), error) {
	router := gin.Default()
	iamModule, err := iam.NewModule(db, redisClient, router, iamCfg)
	if err != nil {
		return nil, nil, err
	}
	authMW := iamModule.AuthMiddleware()

	catalogGroup := router.Group("")
	catalogGroup.Use(authMW, iamModule.RequirePermission("catalog", "manage"))
	catalog.NewModule(db, catalogGroup)

	crmGroup := router.Group("")
	crmGroup.Use(skipWhatsAppWebhook(authMW, iamModule.RequirePermission("crm", "manage")))
	var hook crm.InboundHook
	if db != nil {
		hook = assistant.NewInboundHook(db)
	}
	crmGateway := crm.NewModule(db, crmGroup, organizations.NewWhatsAppConfigReader(db), waCfg, hook)

	orgGroup := router.Group("")
	orgGroup.Use(authMW, iamModule.RequirePermission("organization", "manage"))
	organizations.NewModule(db, orgGroup)

	serviceReader := catalog.NewServiceReader(db)
	scheduleReader := organizations.NewScheduleReader(db)
	personResolver := calendaracl.NewPersonAdapter(iam.NewPersonReader(db))
	calGroup := router.Group("")
	calGroup.Use(authMW, iamModule.RequirePermission("calendar", "manage"))
	calendar.NewModule(
		db,
		calGroup,
		personResolver,
		crm.NewCustomerReader(db),
		serviceReader,
		scheduleReader,
	)
	calendarGateway := calendar.NewAssistantCalendarGateway(db, personResolver, serviceReader, scheduleReader)
	assistantMod := assistant.NewModule(
		db,
		assistantCfg,
		organizations.NewAssistantConfigReader(db),
		crmGateway,
		serviceReader,
		calendarGateway,
	)

	router.StaticFile("/openapi.yaml", "api/openapi.yaml")
	return router, assistantMod.StartWorker, nil
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
