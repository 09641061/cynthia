package application_test

import (
	"context"
	"testing"

	"go-service/internal/assistant/application"
	"go-service/internal/assistant/domain"
)

type countingLLM struct {
	calls int
	fn    func(call int) (domain.CompletionResult, error)
}

func (c *countingLLM) Complete(context.Context, []domain.ChatMessage, []domain.ToolDefinition) (domain.CompletionResult, error) {
	c.calls++
	return c.fn(c.calls)
}

func TestProcessorCanDeliverPendingSuppressionVsMutation(t *testing.T) {
	job := application.Job{ID: "j1", ConversationID: "c1", TriggerMessageID: "m1", Attempts: 1}
	openConv := application.Conversation{ID: "c1", CustomerID: "u1", Status: "open"}
	enabledOrg := fakeOrg{application.OrgConfig{AiEnabled: true, WhatsAppEnabled: true}}

	// HasPending=true and no mutation/handoff -> skip delivery, Complete job.
	jobs := &fakeJobStore{hasPending: true}
	crm := &fakeCRM{conv: openConv}
	p := application.NewProcessor(jobs, fakeLLM{res: domain.CompletionResult{Content: "Hola"}}, enabledOrg, crm, stubCatalog{}, &stubCalendar{})
	if err := p.Process(context.Background(), job); err != nil || !jobs.completed || crm.sentText != "" {
		t.Fatalf("expected skipped delivery when pending exists: err=%v completed=%v sent=%q", err, jobs.completed, crm.sentText)
	}

	// HasPending=true BUT handedOff=true -> must still deliver reply.
	jobs = &fakeJobStore{hasPending: true}
	crm = &fakeCRM{conv: openConv}
	llm := &countingLLM{fn: func(call int) (domain.CompletionResult, error) {
		if call == 1 {
			return domain.CompletionResult{ToolCalls: []domain.ToolCall{{
				ID: "tc1", Function: domain.ToolCallFunction{Name: "handoff_to_human", Arguments: `{"reason":"help"}`},
			}}}, nil
		}
		return domain.CompletionResult{Content: "Te paso con un humano."}, nil
	}}
	p = application.NewProcessor(jobs, llm, enabledOrg, crm, stubCatalog{}, &stubCalendar{})
	if err := p.Process(context.Background(), job); err != nil || !jobs.completed || crm.sentText != "Te paso con un humano." {
		t.Fatalf("expected delivery after handoff even with pending: err=%v completed=%v sent=%q", err, jobs.completed, crm.sentText)
	}
}

func TestProcessorMaxToolRoundsAndPersistedReplyRetry(t *testing.T) {
	openConv := application.Conversation{ID: "c1", CustomerID: "u1", Status: "open"}
	enabledOrg := fakeOrg{application.OrgConfig{AiEnabled: true, WhatsAppEnabled: true}}

	// MaxToolRounds exceeded -> PauseForHandoff(max_tool_rounds_exceeded) + FallbackReplyText.
	jobs := &fakeJobStore{}
	crm := &fakeCRM{conv: openConv}
	loopLLM := &countingLLM{fn: func(int) (domain.CompletionResult, error) {
		return domain.CompletionResult{ToolCalls: []domain.ToolCall{{
			ID: "tc", Function: domain.ToolCallFunction{Name: "list_services", Arguments: `{}`},
		}}}, nil
	}}
	p := application.NewProcessor(jobs, loopLLM, enabledOrg, crm, stubCatalog{}, &stubCalendar{})
	job := application.Job{ID: "j2", ConversationID: "c1", TriggerMessageID: "m2", Attempts: 1}
	if err := p.Process(context.Background(), job); err != nil || crm.handoffReason != "max_tool_rounds_exceeded" || crm.sentText != application.FallbackReplyText {
		t.Fatalf("expected max_tool_rounds_exceeded handoff: err=%v reason=%q sent=%q", err, crm.handoffReason, crm.sentText)
	}

	// Retry with persisted job.ReplyText -> delivers without calling LLM again.
	jobs = &fakeJobStore{}
	crm = &fakeCRM{conv: openConv}
	noCallLLM := &countingLLM{fn: func(int) (domain.CompletionResult, error) {
		return domain.CompletionResult{Content: "should not be called"}, nil
	}}
	p = application.NewProcessor(jobs, noCallLLM, enabledOrg, crm, stubCatalog{}, &stubCalendar{})
	retryJob := application.Job{ID: "j3", ConversationID: "c1", TriggerMessageID: "m3", Attempts: 2, ReplyText: "Saved reply"}
	if err := p.Process(context.Background(), retryJob); err != nil || noCallLLM.calls != 0 || crm.sentText != "Saved reply" || !jobs.completed {
		t.Fatalf("expected persisted reply without LLM call: err=%v calls=%d sent=%q completed=%v", err, noCallLLM.calls, crm.sentText, jobs.completed)
	}
}
