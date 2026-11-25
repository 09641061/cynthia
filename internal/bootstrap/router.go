package bootstrap

import (
	"context"

	"github.com/gin-gonic/gin"
	"go-service/internal/assistant"
	"go-service/internal/bootstrap/config"
	"go-service/internal/calendar"
	"go-service/internal/catalog"
	"go-service/internal/crm"
	"go-service/internal/organizations"
	"go-service/internal/shared/postgres"
)

func newRouter(db *postgres.Client) *gin.Engine {
	router, _ := newApp(db, config.Config{})
	return router
}

func newApp(db *postgres.Client, cfg config.Config) (*gin.Engine, func(context.Context)) {
	router := gin.Default()
	catalog.NewModule(db, router)
	waReader := organizations.NewWhatsAppConfigReader(db)
	var crmGateway *crm.AssistantCRMGateway
	if db != nil {
		crmGateway = crm.NewModule(db, router, waReader, assistant.NewInboundHook(db))
	} else {
		crmGateway = crm.NewModule(db, router, waReader, nil)
	}
	organizations.NewModule(db, router)
	serviceReader := catalog.NewServiceReader(db)
	scheduleReader := organizations.NewScheduleReader(db)
	calendar.NewModule(
		db,
		router,
		nil,
		crm.NewCustomerReader(db),
		serviceReader,
		scheduleReader,
	)
	calendarGateway := calendar.NewAssistantCalendarGateway(db, serviceReader, scheduleReader)
	assistantMod := assistant.NewModule(
		db,
		assistant.Config{
			DeepSeekBaseURL: cfg.DeepSeekBaseURL,
			DeepSeekAPIKey:  cfg.DeepSeekAPIKey,
			DeepSeekModel:   cfg.DeepSeekModel,
			Workers:         cfg.AssistantWorkers,
		},
		organizations.NewAssistantConfigReader(db),
		crmGateway,
		serviceReader,
		calendarGateway,
	)
	router.StaticFile("/openapi.yaml", "api/openapi.yaml")
	return router, assistantMod.StartWorker
}
