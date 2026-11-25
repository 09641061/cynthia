package crm

import (
	"context"
	"errors"
	"testing"

	"github.com/gin-gonic/gin"
	"go-service/internal/crm/application"
	"go-service/internal/crm/domain"
)

func TestAssistantCRMGatewayNilAndNilDBGuard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	var nilGW *AssistantCRMGateway
	nilDBGW := NewModule(nil, gin.New(), nil, WhatsAppConfig{}, nil)

	for name, gw := range map[string]*AssistantCRMGateway{"nil_receiver": nilGW, "nil_db": nilDBGW} {
		if _, err := gw.GetConversation(ctx, "c1"); !errors.Is(err, ErrGatewayUnavailable) {
			t.Fatalf("%s GetConversation: expected ErrGatewayUnavailable, got %v", name, err)
		}
		if _, err := gw.ListRecentMessages(ctx, "c1", 10); !errors.Is(err, ErrGatewayUnavailable) {
			t.Fatalf("%s ListRecentMessages: expected ErrGatewayUnavailable, got %v", name, err)
		}
		if _, err := gw.SendAssistantReply(ctx, "c1", "k1", "hi"); !errors.Is(err, ErrGatewayUnavailable) {
			t.Fatalf("%s SendAssistantReply: expected ErrGatewayUnavailable, got %v", name, err)
		}
		if err := gw.PauseForHandoff(ctx, "c1", "handoff"); !errors.Is(err, ErrGatewayUnavailable) {
			t.Fatalf("%s PauseForHandoff: expected ErrGatewayUnavailable, got %v", name, err)
		}
	}
}

func TestTranslateGatewayErr(t *testing.T) {
	if err := translateGatewayErr(nil); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if err := translateGatewayErr(application.ErrNotFound); !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("expected ErrConversationNotFound, got %v", err)
	}
	if err := translateGatewayErr(application.ErrConflict); !errors.Is(err, ErrConversationClosed) {
		t.Fatalf("expected ErrConversationClosed, got %v", err)
	}
	failed := application.DeliveryError{Message: domain.Message{ID: "m1", Status: domain.MessageStatusFailed}, Reason: "boom"}
	if err := translateGatewayErr(failed); !errors.Is(err, ErrDeliveryFailed) {
		t.Fatalf("expected ErrDeliveryFailed, got %v", err)
	}
	uncertain := application.DeliveryError{Message: domain.Message{ID: "m2", Status: domain.MessageStatusUncertain}, Reason: "timeout"}
	if err := translateGatewayErr(uncertain); !errors.Is(err, ErrDeliveryUncertain) {
		t.Fatalf("expected ErrDeliveryUncertain, got %v", err)
	}
}
