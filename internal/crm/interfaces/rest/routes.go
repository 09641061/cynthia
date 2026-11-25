package rest

import (
	"github.com/gin-gonic/gin"
	"go-service/internal/crm/application"
)

// Handler adapts CRM application operations to HTTP.
type Handler struct {
	customers     *application.CustomerService
	conversations *application.ConversationService
	messages      *application.MessageService
	verifier      application.WebhookVerifier
	events        application.EventSubscriber
}

// RegisterRoutes mounts the CRM HTTP resources on the supplied router.
func RegisterRoutes(
	router gin.IRouter,
	customers *application.CustomerService,
	conversations *application.ConversationService,
	messages *application.MessageService,
	verifier application.WebhookVerifier,
	events ...application.EventSubscriber,
) {
	var subscriber application.EventSubscriber
	if len(events) > 0 {
		subscriber = events[0]
	}
	h := Handler{
		customers:     customers,
		conversations: conversations,
		messages:      messages,
		verifier:      verifier,
		events:        subscriber,
	}
	router.GET("/customers", h.listCustomers)
	router.POST("/customers", h.createCustomer)
	router.GET("/customers/:id", h.getCustomer)
	router.PATCH("/customers/:id", h.updateCustomer)
	router.DELETE("/customers/:id", h.deleteCustomer)

	if conversations != nil {
		router.GET("/conversations", h.listConversations)
		router.GET("/conversations/:id/messages", h.listMessages)
		router.PATCH("/conversations/:id/status", h.updateConversationStatus)
	}
	if messages != nil {
		router.POST("/conversations/:id/messages", h.sendMessage)
		router.GET("/crm/webhooks/whatsapp", h.verifyWhatsAppWebhook)
		router.POST("/crm/webhooks/whatsapp", h.receiveWhatsAppWebhook)
	}
	if subscriber != nil {
		router.GET("/crm/events", h.streamEvents)
	}
}
