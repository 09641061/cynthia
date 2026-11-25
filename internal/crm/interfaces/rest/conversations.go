package rest

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go-service/internal/crm/application"
	sharedrest "go-service/internal/shared/rest"
)

func (h Handler) listConversations(c *gin.Context) {
	page := sharedrest.QueryInt(c, "page", 1, 1, 100000)
	pageSize := sharedrest.QueryInt(c, "page_size", 20, 1, 100)
	customerID := strings.TrimSpace(c.Query("customer_id"))
	if customerID != "" {
		parsed, ok := sharedrest.ParseUUID(customerID)
		if !ok {
			badRequest(c)
			return
		}
		customerID = parsed
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
	convID, ok := sharedrest.PathUUID(c, "id")
	if !ok {
		return
	}
	limit := sharedrest.QueryInt(c, "limit", 50, 1, 200)
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
	convID, ok := sharedrest.PathUUID(c, "id")
	if !ok {
		return
	}
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		writeProblem(c, http.StatusBadRequest, "Bad Request", "Idempotency-Key header is required.")
		return
	}
	var input sendMessageInput
	if !sharedrest.BindJSON(c, &input) {
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
	convID, ok := sharedrest.PathUUID(c, "id")
	if !ok {
		return
	}
	var input updateStatusInput
	if !sharedrest.BindJSON(c, &input) {
		return
	}
	conv, err := h.conversations.UpdateStatus(c.Request.Context(), convID, input.Status)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toConversationResource(conv))
}
