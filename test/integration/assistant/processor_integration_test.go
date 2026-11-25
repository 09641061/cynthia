//go:build integration

package assistant_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-service/internal/assistant/application"
	"go-service/internal/assistant/infrastructure/acl"
	"go-service/internal/assistant/infrastructure/deepseek"
	"go-service/internal/assistant/infrastructure/postgres"
	"go-service/internal/calendar"
	"go-service/internal/catalog"
	"go-service/internal/crm"
	"go-service/internal/organizations"
)

func TestProcessorEndToEndServiceQAAndBookingLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client, seed := newTestClient(t)
	ctx := t.Context()

	// Seed an appointment owned by otherCust to verify ownership check on cancel.
	otherAptID := uuid.NewString()
	futureSlot := time.Now().UTC().AddDate(0, 0, 2).Truncate(time.Hour)
	_, err := client.DB.ExecContext(ctx,
		`INSERT INTO calendar_appointments (id, person_id, customer_id, service_id, starts_at, ends_at, status, channel) VALUES ($1,$2,$3,$4,$5,$6,'confirmed','whatsapp')`,
		otherAptID, seed.personID, seed.otherCust, seed.serviceID, futureSlot.Add(2*time.Hour), futureSlot.Add(150*time.Minute),
	)
	if err != nil {
		t.Fatalf("seed other appointment: %v", err)
	}

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		n := calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		switch n {
		case 1:
			if !strings.Contains(body.Messages[0].Content, "Mention free parking.") {
				t.Errorf("expected ai_instructions in system prompt")
			}
			_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"tool_calls","message":{"tool_calls":[{"id":"c1","type":"function","function":{"name":"list_services","arguments":"{}"}}]}}]}`))
		case 2:
			_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"content":"Tenemos Haircut por 3000 y parqueadero gratis."}}]}`))
		case 3:
			// Attempt to cancel otherCust's appointment -> ownership check must return appointment_not_found.
			_, _ = fmt.Fprintf(w, `{"choices":[{"finish_reason":"tool_calls","message":{"tool_calls":[{"id":"c2","type":"function","function":{"name":"cancel_appointment","arguments":"{\"appointment_id\":\"%s\",\"confirmed_by_customer\":true}"}}]}}]}`, otherAptID)
		case 4:
			if !strings.Contains(body.Messages[len(body.Messages)-1].Content, "appointment_not_found") {
				t.Errorf("expected appointment_not_found for cross-customer cancel, got %s", body.Messages[len(body.Messages)-1].Content)
			}
			_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"tool_calls","message":{"tool_calls":[{"id":"c3","type":"function","function":{"name":"handoff_to_human","arguments":"{\"reason\":\"customer_requested\"}"}}]}}]}`))
		default:
			_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"content":"Te comunico con un asesor."}}]}`))
		}
	}))
	defer srv.Close()

	crmGW := crm.NewModule(client, gin.New(), organizations.NewWhatsAppConfigReader(client), nil)
	recCRM := &recordingCRM{CRMGateway: acl.NewCRMAdapter(crmGW)}
	svcReader := catalog.NewServiceReader(client)
	calGW := calendar.NewAssistantCalendarGateway(client, svcReader, organizations.NewScheduleReader(client))
	jobStore := postgres.NewJobStore(client.DB)
	proc := application.NewProcessor(
		jobStore,
		deepseek.NewClient(srv.URL, "test-key", "deepseek-chat"),
		acl.NewOrgAdapter(organizations.NewAssistantConfigReader(client)),
		recCRM,
		acl.NewCatalogAdapter(svcReader),
		acl.NewCalendarAdapter(calGW),
	)

	// Job 1: service Q&A
	_ = jobStore.Enqueue(ctx, seed.convID, uuid.NewString(), 0)
	j1, ok, err := jobStore.ClaimNext(ctx, 2*time.Minute)
	if err != nil || !ok {
		t.Fatalf("claim j1: ok=%v err=%v", ok, err)
	}
	if err := proc.Process(ctx, j1); err != nil || len(recCRM.sent) != 1 || !strings.Contains(recCRM.sent[0], "Haircut") {
		t.Fatalf("job 1 failed: err=%v sent=%v", err, recCRM.sent)
	}

	// Job 2: cross-customer cancel rejected + handoff_to_human pauses conversation
	_ = jobStore.Enqueue(ctx, seed.convID, uuid.NewString(), 0)
	j2, ok, err := jobStore.ClaimNext(ctx, 2*time.Minute)
	if err != nil || !ok {
		t.Fatalf("claim j2: ok=%v err=%v", ok, err)
	}
	if err := proc.Process(ctx, j2); err != nil || len(recCRM.sent) != 2 {
		t.Fatalf("job 2 failed: err=%v sent=%v", err, recCRM.sent)
	}
	conv, _ := recCRM.GetConversation(ctx, seed.convID)
	if !conv.AiPaused || conv.HandoffReason != "customer_requested" {
		t.Fatalf("expected conversation paused for handoff, got %+v", conv)
	}

	// Job 3: ai_enabled=false skips LLM completely
	_, _ = client.DB.ExecContext(ctx, `UPDATE organizations SET ai_enabled = FALSE`)
	_, _ = client.DB.ExecContext(ctx, `UPDATE crm_conversations SET ai_paused = FALSE WHERE id = $1`, seed.convID)
	_ = jobStore.Enqueue(ctx, seed.convID, uuid.NewString(), 0)
	j3, _, _ := jobStore.ClaimNext(ctx, 2*time.Minute)
	beforeCalls := calls.Load()
	if err := proc.Process(ctx, j3); err != nil || calls.Load() != beforeCalls || len(recCRM.sent) != 2 {
		t.Fatalf("expected ai_enabled=false to skip LLM and send")
	}
}
