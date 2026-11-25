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

// InboundHook receives notifications when a new inbound message is persisted.
type InboundHook = application.InboundHook

// NewModule initializes the CRM module, registers its routes, and returns the assistant CRM gateway.
func NewModule(
	db *sharedpostgres.Client,
	router gin.IRouter,
	configReader application.WhatsAppConfigReader,
	waCfg WhatsAppConfig,
	inboundHook InboundHook,
) *AssistantCRMGateway {
	store := postgres.NewStore(db)
	broker := application.NewEventBroker()
	customers := application.NewCustomerService(store)
	conversations := application.NewConversationService(store, store)
	conversations.SetEventPublisher(broker)
	messages := application.NewMessageService(store, store, store, store, whatsapp.NewClient(waCfg), configReader)
	messages.SetEventPublisher(broker)
	messages.SetInboundHook(inboundHook)
	rest.RegisterRoutes(router.Group("/v1"), customers, conversations, messages, whatsapp.NewVerifier(waCfg), broker)
	return &AssistantCRMGateway{
		store:         store,
		broker:        broker,
		conversations: conversations,
		messages:      messages,
		hasDB:         db != nil,
	}
}
