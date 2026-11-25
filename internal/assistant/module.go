package assistant

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"go-service/internal/assistant/application"
	"go-service/internal/assistant/infrastructure/acl"
	"go-service/internal/assistant/infrastructure/deepseek"
	"go-service/internal/assistant/infrastructure/postgres"
	"go-service/internal/calendar"
	"go-service/internal/catalog"
	"go-service/internal/crm"
	"go-service/internal/organizations"
	sharedpostgres "go-service/internal/shared/postgres"
)

const inboundDebounce = 2 * time.Second

// Config holds assistant runtime settings supplied by bootstrap.
type Config struct {
	DeepSeekBaseURL string
	DeepSeekAPIKey  string
	DeepSeekModel   string
	Workers         int
}

// InboundEnqueuer implements CRM's InboundHook by enqueuing debounced assistant jobs.
type InboundEnqueuer struct {
	store *postgres.JobStore
}

// NewInboundHook constructs a nil-DB-safe inbound message hook for CRM.
func NewInboundHook(db *sharedpostgres.Client) *InboundEnqueuer {
	if db == nil {
		return &InboundEnqueuer{}
	}
	return &InboundEnqueuer{store: postgres.NewJobStore(db.DB)}
}

// OnInboundMessage enqueues or coalesces a pending assistant job with a 2s debounce.
func (h *InboundEnqueuer) OnInboundMessage(ctx context.Context, convID, msgID string) {
	if h == nil || h.store == nil {
		return
	}
	if err := h.store.Enqueue(ctx, convID, msgID, inboundDebounce); err != nil {
		slog.Error("assistant enqueue inbound job failed", "conversation_id", convID, "error", err)
	}
}

// Module encapsulates the assistant background worker and processor.
type Module struct {
	worker *application.Worker
}

// NewModule wires the assistant worker with PostgreSQL job storage, DeepSeek, and ACL adapters.
func NewModule(
	db *sharedpostgres.Client,
	cfg Config,
	orgReader *organizations.AssistantConfigReader,
	crmGateway *crm.AssistantCRMGateway,
	catalogReader *catalog.ServiceReader,
	calendarGateway *calendar.AssistantCalendarGateway,
) *Module {
	var sqlDB *sql.DB
	if db != nil {
		sqlDB = db.DB
	}
	jobStore := postgres.NewJobStore(sqlDB)
	llmClient := deepseek.NewClient(cfg.DeepSeekBaseURL, cfg.DeepSeekAPIKey, cfg.DeepSeekModel)
	processor := application.NewProcessor(
		jobStore,
		llmClient,
		acl.NewOrgAdapter(orgReader),
		acl.NewCRMAdapter(crmGateway),
		acl.NewCatalogAdapter(catalogReader),
		acl.NewCalendarAdapter(calendarGateway),
	)
	return &Module{
		worker: application.NewWorker(jobStore, processor, cfg.Workers),
	}
}

// StartWorker runs the assistant background worker pool until ctx is cancelled.
func (m *Module) StartWorker(ctx context.Context) {
	if m == nil || m.worker == nil {
		return
	}
	m.worker.Start(ctx)
}
