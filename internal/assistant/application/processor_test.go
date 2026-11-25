package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-service/internal/assistant/application"
	"go-service/internal/assistant/domain"
)

type fakeRetryableErr struct{ retryable bool }

func (e fakeRetryableErr) Error() string     { return "llm error" }
func (e fakeRetryableErr) IsRetryable() bool { return e.retryable }

type fakeJobStore struct {
	completed   bool
	failed      bool
	retryable   bool
	hasPending  bool
	recentCount int
	savedReply  string
}

func (f *fakeJobStore) Enqueue(context.Context, string, string, time.Duration) error { return nil }
func (f *fakeJobStore) ClaimNext(context.Context, time.Duration) (application.Job, bool, error) {
	return application.Job{}, false, nil
}
func (f *fakeJobStore) HasPendingForConversation(context.Context, string) (bool, error) {
	return f.hasPending, nil
}
func (f *fakeJobStore) CountRecentCompleted(context.Context, string, time.Time) (int, error) {
	return f.recentCount, nil
}
func (f *fakeJobStore) SaveReply(_ context.Context, _, text string) error {
	f.savedReply = text
	return nil
}
func (f *fakeJobStore) Complete(context.Context, string) error { f.completed = true; return nil }
func (f *fakeJobStore) FailOrRetry(_ context.Context, _, _, _ string, retryable bool, _ time.Duration) error {
	f.failed, f.retryable = true, retryable
	return nil
}

type fakeOrg struct{ cfg application.OrgConfig }

func (f fakeOrg) GetAssistantConfig(context.Context) (application.OrgConfig, error) {
	return f.cfg, nil
}

type fakeLLM struct {
	res domain.CompletionResult
	err error
}

func (f fakeLLM) Complete(context.Context, []domain.ChatMessage, []domain.ToolDefinition) (domain.CompletionResult, error) {
	return f.res, f.err
}

type fakeCRM struct {
	conv          application.Conversation
	sendErr       error
	sentText      string
	handoffReason string
}

func (f *fakeCRM) GetConversation(context.Context, string) (application.Conversation, error) {
	return f.conv, nil
}
func (f *fakeCRM) ListRecentMessages(context.Context, string, int) ([]application.Message, error) {
	return []application.Message{{ID: "m1", Direction: "inbound", Content: "Hola"}}, nil
}
func (f *fakeCRM) SendAssistantReply(_ context.Context, _, _, content string) (application.Message, error) {
	f.sentText = content
	return application.Message{}, f.sendErr
}
func (f *fakeCRM) PauseForHandoff(_ context.Context, _, reason string) error {
	f.handoffReason = reason
	return nil
}

func TestProcessorGatesAndRateLimit(t *testing.T) {
	job := application.Job{ID: "j1", ConversationID: "c1", TriggerMessageID: "m1", Attempts: 1}
	openConv := application.Conversation{ID: "c1", CustomerID: "u1", Status: "open"}

	// AI disabled -> Complete without sending
	jobs := &fakeJobStore{}
	crm := &fakeCRM{conv: openConv}
	p := application.NewProcessor(jobs, fakeLLM{}, fakeOrg{application.OrgConfig{AiEnabled: false, WhatsAppEnabled: true}}, crm, stubCatalog{}, &stubCalendar{})
	if err := p.Process(context.Background(), job); err != nil || !jobs.completed || crm.sentText != "" {
		t.Fatalf("expected complete without send when AI disabled: err=%v completed=%v sent=%q", err, jobs.completed, crm.sentText)
	}

	// Rate limit exceeded -> PauseForHandoff + fallback + Complete
	jobs = &fakeJobStore{recentCount: 20}
	crm = &fakeCRM{conv: openConv}
	p = application.NewProcessor(jobs, fakeLLM{}, fakeOrg{application.OrgConfig{AiEnabled: true, WhatsAppEnabled: true}}, crm, stubCatalog{}, &stubCalendar{})
	if err := p.Process(context.Background(), job); err != nil || crm.handoffReason != "rate_limit_exceeded" || crm.sentText != application.FallbackReplyText || !jobs.completed {
		t.Fatalf("unexpected rate limit outcome: reason=%q sent=%q completed=%v", crm.handoffReason, crm.sentText, jobs.completed)
	}
}

func TestProcessorLLMErrorsAndDeliveryErrors(t *testing.T) {
	job := application.Job{ID: "j1", ConversationID: "c1", TriggerMessageID: "m1", Attempts: 1}
	openConv := application.Conversation{ID: "c1", CustomerID: "u1", Status: "open"}
	enabledOrg := fakeOrg{application.OrgConfig{AiEnabled: true, WhatsAppEnabled: true}}

	// Retryable LLM error -> FailOrRetry(retryable=true)
	jobs := &fakeJobStore{}
	p := application.NewProcessor(jobs, fakeLLM{err: fakeRetryableErr{retryable: true}}, enabledOrg, &fakeCRM{conv: openConv}, stubCatalog{}, &stubCalendar{})
	_ = p.Process(context.Background(), job)
	if !jobs.failed || !jobs.retryable {
		t.Fatalf("expected retryable failure, got failed=%v retryable=%v", jobs.failed, jobs.retryable)
	}

	// DeliveryFailed -> PauseForHandoff + FailOrRetry(retryable=false)
	jobs = &fakeJobStore{}
	crm := &fakeCRM{conv: openConv, sendErr: application.ErrDeliveryFailed}
	p = application.NewProcessor(jobs, fakeLLM{res: domain.CompletionResult{Content: "Hola!"}}, enabledOrg, crm, stubCatalog{}, &stubCalendar{})
	_ = p.Process(context.Background(), job)
	if crm.handoffReason != "assistant_delivery_failed" || !jobs.failed || jobs.retryable {
		t.Fatalf("unexpected delivery failed outcome: reason=%q failed=%v retryable=%v", crm.handoffReason, jobs.failed, jobs.retryable)
	}

	// DeliveryUncertain -> FailOrRetry(retryable=false), no handoff
	jobs = &fakeJobStore{}
	crm = &fakeCRM{conv: openConv, sendErr: application.ErrDeliveryUncertain}
	p = application.NewProcessor(jobs, fakeLLM{res: domain.CompletionResult{Content: "Hola!"}}, enabledOrg, crm, stubCatalog{}, &stubCalendar{})
	_ = p.Process(context.Background(), job)
	if crm.handoffReason != "" || !jobs.failed || jobs.retryable {
		t.Fatalf("unexpected delivery uncertain outcome: reason=%q failed=%v retryable=%v", crm.handoffReason, jobs.failed, jobs.retryable)
	}

	// ConversationClosed -> Complete, no handoff
	jobs = &fakeJobStore{}
	crm = &fakeCRM{conv: openConv, sendErr: errors.Join(application.ErrConversationClosed)}
	p = application.NewProcessor(jobs, fakeLLM{res: domain.CompletionResult{Content: "Hola!"}}, enabledOrg, crm, stubCatalog{}, &stubCalendar{})
	_ = p.Process(context.Background(), job)
	if !jobs.completed || crm.handoffReason != "" {
		t.Fatalf("expected complete without handoff on closed conversation")
	}
}
