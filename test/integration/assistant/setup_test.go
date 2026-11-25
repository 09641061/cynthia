//go:build integration

package assistant_test

import (
	"context"
	"database/sql"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go-service/internal/assistant/application"
	"go-service/internal/shared/circuitbreaker"
	sharedpostgres "go-service/internal/shared/postgres"
)

type seedData struct {
	convID     string
	customerID string
	otherCust  string
	serviceID  string
	personID   string
}

func newTestClient(t *testing.T) (*sharedpostgres.Client, seedData) {
	t.Helper()
	parsed, err := url.Parse(testDatabaseURL)
	if err != nil || parsed.Host == "" {
		t.Fatal("missing testDatabaseURL")
	}
	admin, err := sql.Open("pgx", parsed.String())
	if err != nil {
		t.Fatalf("open admin db: %v", err)
	}
	ctx := t.Context()
	schema := "asst_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		_ = admin.Close()
		t.Fatalf("create schema: %v", err)
	}
	var db *sql.DB
	t.Cleanup(func() {
		if db != nil {
			_ = db.Close()
		}
		_, _ = admin.ExecContext(context.Background(), `DROP SCHEMA `+schema+` CASCADE`)
		_ = admin.Close()
	})
	q := parsed.Query()
	q.Set("search_path", schema+",public")
	parsed.RawQuery = q.Encode()
	db, err = sql.Open("pgx", parsed.String())
	if err != nil {
		t.Fatalf("open schema db: %v", err)
	}
	db.SetMaxOpenConns(8)
	if err := sharedpostgres.Migrate(ctx, db, "../../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	breaker, _ := circuitbreaker.New(5, 30*time.Second)
	client := sharedpostgres.NewClient(db, breaker)
	return client, seedAssistantFixtures(t, db)
}

func seedAssistantFixtures(t *testing.T, db *sql.DB) seedData {
	t.Helper()
	ctx := t.Context()
	s := seedData{
		convID: uuid.NewString(), customerID: uuid.NewString(),
		otherCust: uuid.NewString(), serviceID: uuid.NewString(), personID: uuid.NewString(),
	}
	catID := uuid.NewString()
	roleID := uuid.NewString()
	stmts := []struct {
		q    string
		args []any
	}{
		{`UPDATE organizations SET ai_enabled = TRUE, whatsapp_enabled = TRUE, whatsapp_phone_number = '+573000000000', whatsapp_phone_number_id = '1234567890', timezone = 'UTC', ai_instructions = 'Mention free parking.'`, nil},
		{`INSERT INTO crm_customers (id, name, phone) VALUES ($1, 'Alice', '+573001112233'), ($2, 'Bob', '+573009998877')`, []any{s.customerID, s.otherCust}},
		{`INSERT INTO crm_conversations (id, customer_id, status) VALUES ($1, $2, 'open')`, []any{s.convID, s.customerID}},
		{`INSERT INTO crm_messages (id, conversation_id, direction, content, status, created_at) VALUES ($1, $2, 'outbound', 'Hola, en que te ayudo?', 'sent', NOW() - INTERVAL '2 minute'), ($3, $2, 'inbound', 'Quiero agendar', 'received', NOW() - INTERVAL '1 minute')`, []any{uuid.NewString(), s.convID, uuid.NewString()}},
		{`INSERT INTO catalog_categories (id, name, sort_order) VALUES ($1, 'Hair', 1)`, []any{catID}},
		{`INSERT INTO catalog_services (id, category_id, name, duration_minutes, price_minor_units, sort_order) VALUES ($1, $2, 'Haircut', 30, 3000, 1)`, []any{s.serviceID, catID}},
		{`INSERT INTO iam_users (id, email, first_name, last_name, password_hash, type) VALUES ($1, 'ana@test.local', 'Stylist', 'Ana', 'hash', 'owner')`, []any{s.personID}},
		{`INSERT INTO iam_roles (id, name, position, permissions) VALUES ($1, 'Staff', 1, ARRAY['scheduling.appear'])`, []any{roleID}},
		{`INSERT INTO iam_member_roles (user_id, role_id) VALUES ($1, $2)`, []any{s.personID, roleID}},
	}
	for _, st := range stmts {
		if _, err := db.ExecContext(ctx, st.q, st.args...); err != nil {
			t.Fatalf("seed query failed: %v", err)
		}
	}
	return s
}

type recordingCRM struct {
	application.CRMGateway
	sent []string
}

func (r *recordingCRM) SendAssistantReply(_ context.Context, _, _, content string) (application.Message, error) {
	r.sent = append(r.sent, content)
	return application.Message{ID: uuid.NewString(), Direction: "outbound", Content: content, Status: "sent"}, nil
}
