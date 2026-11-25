package rest

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"go-service/internal/crm/application"
	"go-service/internal/crm/domain"
)

type problem struct {
	Type             string            `json:"type"`
	Title            string            `json:"title"`
	Status           int               `json:"status"`
	Detail           string            `json:"detail"`
	Instance         string            `json:"instance"`
	Errors           []problemField    `json:"errors"`
	ExistingCustomer *customerResource `json:"existing_customer,omitempty"`
	Message          *messageResource  `json:"message,omitempty"`
}

type problemField struct {
	Field   string `json:"field"`
	Pointer string `json:"pointer"`
	Code    string `json:"code"`
	Reason  string `json:"reason"`
}

func badRequest(c *gin.Context) {
	writeProblem(c, http.StatusBadRequest, "Bad Request", "The request body or identifier is invalid.")
}

func writeError(c *gin.Context, err error) {
	var fieldErr domain.InvalidFieldError
	if errors.As(err, &fieldErr) {
		writeProblemFields(c, http.StatusUnprocessableEntity, "Unprocessable Content", "CRM fields are invalid.", []problemField{{
			Field: fieldErr.Field, Pointer: "#/" + fieldErr.Field, Code: "INVALID_FIELD", Reason: fieldErr.Reason,
		}}, nil, nil)
		return
	}
	var dupErr application.DuplicatePhoneError
	if errors.As(err, &dupErr) {
		existing := toCustomerResource(dupErr.Existing)
		writeProblemFields(c, http.StatusConflict, "Conflict", "A customer with this phone already exists.", []problemField{}, &existing, nil)
		return
	}
	var delErr application.DeliveryError
	if errors.As(err, &delErr) {
		saved := toMessageResource(delErr.Message)
		detail := delErr.Reason
		if detail == "" {
			detail = "WhatsApp delivery failed."
		}
		writeProblemFields(c, http.StatusBadGateway, "Bad Gateway", detail, []problemField{}, nil, &saved)
		return
	}
	switch {
	case errors.Is(err, application.ErrNotFound):
		writeProblem(c, http.StatusNotFound, "Not Found", "The requested CRM resource does not exist.")
	case errors.Is(err, application.ErrConflict):
		writeProblem(c, http.StatusConflict, "Conflict", "The CRM state conflicts with this operation.")
	case errors.Is(err, domain.ErrInvalid):
		writeProblem(c, http.StatusUnprocessableEntity, "Unprocessable Content", "The CRM data is invalid.")
	default:
		slog.ErrorContext(c.Request.Context(), "crm operation failed", "error", err)
		writeProblem(c, http.StatusInternalServerError, "Internal Server Error", "The CRM operation could not be completed.")
	}
}

func writeProblem(c *gin.Context, status int, title, detail string) {
	writeProblemFields(c, status, title, detail, []problemField{}, nil, nil)
}

func writeProblemFields(c *gin.Context, status int, title, detail string, fields []problemField, existing *customerResource, msg *messageResource) {
	c.Header("Cache-Control", "no-store")
	c.Header("Content-Type", "application/problem+json")
	c.AbortWithStatusJSON(status, problem{
		Type: "about:blank", Title: title, Status: status,
		Detail: detail, Instance: c.Request.URL.Path, Errors: fields,
		ExistingCustomer: existing, Message: msg,
	})
}
