package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"go-service/internal/assistant/domain"
)

func (p *Processor) generateReply(ctx context.Context, orgCfg OrgConfig, conv Conversation) (string, bool, bool, error) {
	msgs, err := p.crm.ListRecentMessages(ctx, conv.ID, domain.MaxHistoryMessages)
	if err != nil {
		return "", false, false, err
	}
	history := make([]domain.HistoryMessage, 0, len(msgs))
	hasPriorAssistantTurn := false
	for _, m := range msgs {
		if m.Direction == "outbound" {
			hasPriorAssistantTurn = true
		}
		history = append(history, domain.HistoryMessage{Direction: m.Direction, Body: m.Content})
	}
	promptCfg := domain.PromptConfig{
		Timezone:       orgCfg.Timezone,
		CurrencyCode:   orgCfg.CurrencyCode,
		BusinessHours:  orgCfg.BusinessHours,
		AiInstructions: orgCfg.AiInstructions,
		Now:            p.now(),
	}
	chatMsgs := domain.BuildMessages(promptCfg, history)
	tools := AllToolDefinitions()
	exec := NewToolExecutor(p.crm, p.catalog, p.calendar, conv, hasPriorAssistantTurn, orgCfg.CurrencyCode)

	for round := 0; round < domain.MaxToolRounds; round++ {
		res, err := p.llm.Complete(ctx, chatMsgs, tools)
		if err != nil {
			return "", false, false, err
		}
		if len(res.ToolCalls) == 0 {
			text := domain.SanitizeWhatsAppReply(res.Content)
			if text != "" {
				return text, exec.HandedOff(), exec.MutatedOrHandedOff(), nil
			}
			break
		}
		chatMsgs = append(chatMsgs, domain.ChatMessage{
			Role:      "assistant",
			Content:   res.Content,
			ToolCalls: res.ToolCalls,
		})
		for _, call := range res.ToolCalls {
			result := exec.Execute(ctx, call)
			chatMsgs = append(chatMsgs, domain.ChatMessage{
				Role:       "tool",
				ToolCallID: call.ID,
				Content:    result,
			})
		}
	}

	_ = p.crm.PauseForHandoff(ctx, conv.ID, "max_tool_rounds_exceeded")
	return FallbackReplyText, true, true, nil
}

func retryBackoff(attempts int) time.Duration {
	if attempts <= 1 {
		return baseRetryBackoff
	}
	d := baseRetryBackoff << min(attempts-1, 6)
	if d > maxRetryBackoff {
		return maxRetryBackoff
	}
	return d
}

func sanitizeErr(err error) string {
	if err == nil {
		return ""
	}
	s := strings.TrimSpace(err.Error())
	runes := []rune(s)
	if len(runes) > maxErrorRunes {
		return string(runes[:maxErrorRunes])
	}
	return s
}

func persistCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
}

func (p *Processor) complete(ctx context.Context, jobID string) error {
	pCtx, cancel := persistCtx(ctx)
	defer cancel()
	return p.jobs.Complete(pCtx, jobID)
}

func (p *Processor) failOrRetry(ctx context.Context, job Job, err error, retryable bool, backoff time.Duration) error {
	pCtx, cancel := persistCtx(ctx)
	defer cancel()
	return p.jobs.FailOrRetry(pCtx, job.ID, job.ConversationID, sanitizeErr(err), retryable, backoff)
}

func (p *Processor) deliverReply(ctx context.Context, job Job, replyText string) error {
	_, err := p.crm.SendAssistantReply(ctx, job.ConversationID, "ai:"+job.TriggerMessageID, replyText)
	if err == nil {
		return p.complete(ctx, job.ID)
	}
	switch {
	case errors.Is(err, ErrConversationClosed):
		return p.complete(ctx, job.ID)
	case errors.Is(err, ErrDeliveryFailed):
		pCtx, cancel := persistCtx(ctx)
		defer cancel()
		_ = p.crm.PauseForHandoff(pCtx, job.ConversationID, "assistant_delivery_failed")
		return p.jobs.FailOrRetry(pCtx, job.ID, job.ConversationID, sanitizeErr(err), false, 0)
	case errors.Is(err, ErrDeliveryUncertain):
		return p.failOrRetry(ctx, job, err, false, 0)
	default:
		return p.failOrRetry(ctx, job, err, true, retryBackoff(job.Attempts))
	}
}
