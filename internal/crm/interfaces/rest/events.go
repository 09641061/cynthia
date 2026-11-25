package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const sseKeepaliveInterval = 25 * time.Second

type sseEventPayload struct {
	ConversationID string `json:"conversation_id"`
}

func (h Handler) streamEvents(c *gin.Context) {
	w := c.Writer
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	ctx := c.Request.Context()
	events, unsubscribe := h.events.Subscribe()
	defer unsubscribe()

	if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
		return
	}
	w.Flush()

	ticker := time.NewTicker(sseKeepaliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			payload, err := json.Marshal(sseEventPayload{ConversationID: ev.ConversationID})
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, payload); err != nil {
				return
			}
			w.Flush()
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			w.Flush()
		}
	}
}
