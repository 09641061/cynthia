package crm

import (
	"github.com/gin-gonic/gin"
	"go-service/internal/crm/application"
	"go-service/internal/crm/infrastructure/persistence/postgres"
	"go-service/internal/crm/infrastructure/whatsapp"
	"go-service/internal/crm/interfaces/rest"
	sharedpostgres "go-service/internal/shared/postgres"
)

// WhatsAppConfig holds Meta WhatsApp Cloud API settings injected at startup.
type WhatsAppConfig = whatsapp.Config

// NewModule initializes the CRM module and registers its routes.
func NewModule(db *sharedpostgres.Client, router gin.IRouter, configReader application.WhatsAppConfigReader, waCfg WhatsAppConfig) {
	store := postgres.NewStore(db)
	broker := application.NewEventBroker()
	customers := application.NewCustomerService(store)
	conversations := application.NewConversationService(store, store)
	conversations.SetEventPublisher(broker)
	messages := application.NewMessageService(store, store, store, store, whatsapp.NewClient(waCfg), configReader)
	messages.SetEventPublisher(broker)
	rest.RegisterRoutes(router.Group("/v1"), customers, conversations, messages, whatsapp.NewVerifier(waCfg), broker)
}
