package rest

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-service/internal/crm/application"
)

func (h Handler) listConversations(c *gin.Context) {
	page, ok := queryInt(c, "page", 1)
	if !ok {
		return
	}
	pageSize, ok := queryInt(c, "page_size", 20)
	if !ok {
		return
	}
	customerID := strings.TrimSpace(c.Query("customer_id"))
	if customerID != "" {
		if _, err := uuid.Parse(customerID); err != nil {
			badRequest(c)
			return
		}
	}
	result, err := h.conversations.List(c.Request.Context(), application.ConversationFilter{
		Query:      c.Query("q"),
		Status:     c.Query("status"),
		CustomerID: customerID,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toConversationPageResource(result))
}

func (h Handler) listMessages(c *gin.Context) {
	convID, ok := pathID(c, "id")
	if !ok {
		return
	}
	limit, ok := queryInt(c, "limit", 50)
	if !ok {
		return
	}
	var before *time.Time
	if rawBefore := strings.TrimSpace(c.Query("before")); rawBefore != "" {
		parsed, err := time.Parse(time.RFC3339Nano, rawBefore)
		if err != nil {
			badRequest(c)
			return
		}
		before = &parsed
	}
	items, err := h.conversations.ListMessages(c.Request.Context(), convID, limit, before)
	if err != nil {
		writeError(c, err)
		return
	}
	res := make([]messageResource, 0, len(items))
	for _, m := range items {
		res = append(res, toMessageResource(m))
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"items": res})
}

func (h Handler) sendMessage(c *gin.Context) {
	convID, ok := pathID(c, "id")
	if !ok {
		return
	}
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		writeProblem(c, http.StatusBadRequest, "Bad Request", "Idempotency-Key header is required.")
		return
	}
	var input sendMessageInput
	if !bindJSON(c, &input) {
		return
	}
	msg, err := h.messages.SendOutbound(c.Request.Context(), convID, idempotencyKey, input.Content)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, toMessageResource(msg))
}

func (h Handler) updateConversationStatus(c *gin.Context) {
	convID, ok := pathID(c, "id")
	if !ok {
		return
	}
	version, ok := ifMatch(c)
	if !ok {
		return
	}
	var input updateStatusInput
	if !bindJSON(c, &input) {
		return
	}
	conv, err := h.conversations.UpdateStatus(c.Request.Context(), convID, version, input.Status)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("ETag", entityTag(conv.Version))
	c.JSON(http.StatusOK, toConversationResource(conv))
}
