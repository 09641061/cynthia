package application

import (
	"context"
	"strings"
	"time"
)

const (
	// FallbackReplyText is sent when the assistant hands off due to limits or exhaustion.
	FallbackReplyText = "Te paso con una persona de nuestro equipo para ayudarte mejor en un momento."

	jobTimeout         = 90 * time.Second
	maxRecentCompleted = 20
	rateLimitWindow    = time.Hour
	baseRetryBackoff   = 5 * time.Second
	maxRetryBackoff    = 5 * time.Minute
	maxErrorRunes      = 300
)

// Processor executes a single claimed assistant job end to end.
type Processor struct {
	jobs     JobStore
	llm      LLMClient
	orgs     OrgReader
	crm      CRMGateway
	catalog  CatalogReader
	calendar CalendarGateway
	now      func() time.Time
}

// NewProcessor constructs a Processor with all required ports.
func NewProcessor(
	jobs JobStore,
	llm LLMClient,
	orgs OrgReader,
	crm CRMGateway,
	catalog CatalogReader,
	calendar CalendarGateway,
) *Processor {
	return &Processor{
		jobs:     jobs,
		llm:      llm,
		orgs:     orgs,
		crm:      crm,
		catalog:  catalog,
		calendar: calendar,
		now:      time.Now,
	}
}

// Process runs one claimed assistant job with a 90s timeout.
func (p *Processor) Process(ctx context.Context, job Job) error {
	ctx, cancel := context.WithTimeout(ctx, jobTimeout)
	defer cancel()

	orgCfg, conv, ok, err := p.loadAndCheckGates(ctx, job)
	if err != nil {
		return p.failOrRetry(ctx, job, err, true, retryBackoff(job.Attempts))
	}
	if !ok {
		return p.complete(ctx, job.ID)
	}

	recentCount, err := p.jobs.CountRecentCompleted(ctx, job.ConversationID, p.now().Add(-rateLimitWindow))
	if err != nil {
		return p.failOrRetry(ctx, job, err, true, retryBackoff(job.Attempts))
	}
	if recentCount >= maxRecentCompleted {
		_ = p.crm.PauseForHandoff(ctx, job.ConversationID, "rate_limit_exceeded")
		_, _ = p.crm.SendAssistantReply(ctx, job.ConversationID, "ai:"+job.TriggerMessageID, FallbackReplyText)
		return p.complete(ctx, job.ID)
	}

	replyText := strings.TrimSpace(job.ReplyText)
	var handedOff, mutatedOrHandedOff bool
	if replyText == "" {
		replyText, handedOff, mutatedOrHandedOff, err = p.generateReply(ctx, orgCfg, conv)
		if err != nil {
			retryable := IsRetryableError(err)
			backoff := time.Duration(0)
			if retryable {
				backoff = retryBackoff(job.Attempts)
			}
			return p.failOrRetry(ctx, job, err, retryable, backoff)
		}
		if err := p.jobs.SaveReply(ctx, job.ID, replyText); err != nil {
			return p.failOrRetry(ctx, job, err, true, retryBackoff(job.Attempts))
		}
	}

	canSend, err := p.canDeliver(ctx, job.ConversationID, handedOff, mutatedOrHandedOff)
	if err != nil {
		return p.failOrRetry(ctx, job, err, true, retryBackoff(job.Attempts))
	}
	if !canSend {
		return p.complete(ctx, job.ID)
	}
	return p.deliverReply(ctx, job, replyText)
}

func (p *Processor) loadAndCheckGates(ctx context.Context, job Job) (OrgConfig, Conversation, bool, error) {
	orgCfg, err := p.orgs.GetAssistantConfig(ctx)
	if err != nil {
		return OrgConfig{}, Conversation{}, false, err
	}
	conv, err := p.crm.GetConversation(ctx, job.ConversationID)
	if err != nil {
		return OrgConfig{}, Conversation{}, false, err
	}
	eligible := orgCfg.AiEnabled && orgCfg.WhatsAppEnabled && !conv.AiPaused && conv.Status == "open"
	return orgCfg, conv, eligible, nil
}

func (p *Processor) canDeliver(ctx context.Context, convID string, handedOff, mutatedOrHandedOff bool) (bool, error) {
	orgCfg, err := p.orgs.GetAssistantConfig(ctx)
	if err != nil {
		return false, err
	}
	conv, err := p.crm.GetConversation(ctx, convID)
	if err != nil {
		return false, err
	}
	if !(orgCfg.AiEnabled && orgCfg.WhatsAppEnabled && (!conv.AiPaused || handedOff) && conv.Status == "open") {
		return false, nil
	}
	if !mutatedOrHandedOff {
		hasPending, err := p.jobs.HasPendingForConversation(ctx, convID)
		if err != nil {
			return false, err
		}
		if hasPending {
			return false, nil
		}
	}
	return true, nil
}
