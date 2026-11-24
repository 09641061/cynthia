package rest

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"go-service/internal/catalog/application"
	"go-service/internal/catalog/domain"
)

type problem struct {
	Type     string         `json:"type"`
	Title    string         `json:"title"`
	Status   int            `json:"status"`
	Detail   string         `json:"detail"`
	Instance string         `json:"instance"`
	Errors   []problemField `json:"errors"`
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
		writeProblemFields(c, http.StatusUnprocessableEntity, "Unprocessable Content", "Catalog fields are invalid.", []problemField{{
			Field: fieldErr.Field, Pointer: "#/" + fieldErr.Field, Code: "INVALID_FIELD", Reason: fieldErr.Reason,
		}})
		return
	}
	switch {
	case errors.Is(err, application.ErrNotFound):
		writeProblem(c, http.StatusNotFound, "Not Found", "The requested catalog resource does not exist.")
	case errors.Is(err, application.ErrConflict):
		writeProblem(c, http.StatusConflict, "Conflict", "The catalog state conflicts with this operation.")
	case errors.Is(err, application.ErrPreconditionFailed):
		writeProblem(c, http.StatusPreconditionFailed, "Precondition Failed", "The catalog resource has changed.")
	case errors.Is(err, domain.ErrInvalid):
		writeProblem(c, http.StatusUnprocessableEntity, "Unprocessable Content", "The catalog data or ordering is invalid.")
	default:
		slog.ErrorContext(c.Request.Context(), "catalog operation failed", "error", err)
		writeProblem(c, http.StatusInternalServerError, "Internal Server Error", "The catalog operation could not be completed.")
	}
}

func writeProblem(c *gin.Context, status int, title, detail string) {
	writeProblemFields(c, status, title, detail, []problemField{})
}

func writeProblemFields(c *gin.Context, status int, title, detail string, fields []problemField) {
	c.Header("Cache-Control", "no-store")
	c.Header("Content-Type", "application/problem+json")
	c.AbortWithStatusJSON(status, problem{
		Type: "about:blank", Title: title, Status: status,
		Detail: detail, Instance: c.Request.URL.Path, Errors: fields,
	})
}
