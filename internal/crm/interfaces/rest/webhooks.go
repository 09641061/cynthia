package rest

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"go-service/internal/crm/domain"
)

type metaWebhookPayload struct {
	Entry []struct {
		Changes []struct {
			Value struct {
				Contacts []struct {
					WaID    string `json:"wa_id"`
					Profile struct {
						Name string `json:"name"`
					} `json:"profile"`
				} `json:"contacts"`
				Messages []struct {
					ID   string `json:"id"`
					From string `json:"from"`
					Type string `json:"type"`
					Text struct {
						Body string `json:"body"`
					} `json:"text"`
				} `json:"messages"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

func (h Handler) verifyWhatsAppWebhook(c *gin.Context) {
	if h.verifier == nil {
		writeProblem(c, http.StatusForbidden, "Forbidden", "Webhook verification is not configured.")
		return
	}
	challenge, ok := h.verifier.VerifyChallenge(
		c.Query("hub.mode"),
		c.Query("hub.verify_token"),
		c.Query("hub.challenge"),
	)
	if !ok {
		writeProblem(c, http.StatusForbidden, "Forbidden", "Webhook verification failed.")
		return
	}
	c.String(http.StatusOK, challenge)
}

func (h Handler) receiveWhatsAppWebhook(c *gin.Context) {
	if h.verifier == nil {
		writeProblem(c, http.StatusForbidden, "Forbidden", "Webhook verification is not configured.")
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		badRequest(c)
		return
	}
	if !h.verifier.VerifySignature(c.GetHeader("X-Hub-Signature-256"), body) {
		writeProblem(c, http.StatusUnauthorized, "Unauthorized", "Invalid webhook signature.")
		return
	}
	var payload metaWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		badRequest(c)
		return
	}
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			names := make(map[string]string, len(change.Value.Contacts))
			for _, contact := range change.Value.Contacts {
				names[contact.WaID] = contact.Profile.Name
			}
			for _, msg := range change.Value.Messages {
				if msg.Type != "text" || msg.ID == "" || msg.From == "" {
					continue
				}
				if _, err := h.messages.ReceiveInbound(c.Request.Context(), msg.ID, msg.From, names[msg.From], msg.Text.Body); err != nil {
					slog.ErrorContext(c.Request.Context(), "failed to process inbound whatsapp message", "external_id", msg.ID, "error", err)
					if !errors.Is(err, domain.ErrInvalid) {
						writeProblem(c, http.StatusInternalServerError, "Internal Server Error", "Failed to process inbound message.")
						return
					}
				}
			}
		}
	}
	c.Status(http.StatusOK)
}
