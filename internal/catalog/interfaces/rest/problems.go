package rest

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"go-service/internal/catalog/application"
	"go-service/internal/catalog/domain"
	sharedrest "go-service/internal/shared/rest"
)

func badRequest(c *gin.Context) {
	sharedrest.BadRequest(c, "The request body or identifier is invalid.")
}

func writeError(c *gin.Context, err error) {
	var fieldErr domain.InvalidFieldError
	if errors.As(err, &fieldErr) {
		sharedrest.WriteProblemFields(c, http.StatusUnprocessableEntity, "Unprocessable Content", "Catalog fields are invalid.", []sharedrest.ProblemField{{
			Field: fieldErr.Field, Pointer: "#/" + fieldErr.Field, Code: "INVALID_FIELD", Reason: fieldErr.Reason,
		}})
		return
	}
	switch {
	case errors.Is(err, application.ErrNotFound):
		sharedrest.WriteProblem(c, http.StatusNotFound, "Not Found", "The requested catalog resource does not exist.")
	case errors.Is(err, application.ErrConflict):
		sharedrest.WriteProblem(c, http.StatusConflict, "Conflict", "The catalog state conflicts with this operation.")
	case errors.Is(err, application.ErrPreconditionFailed):
		sharedrest.WriteProblem(c, http.StatusPreconditionFailed, "Precondition Failed", "The catalog resource has changed.")
	case errors.Is(err, domain.ErrInvalid):
		sharedrest.WriteProblem(c, http.StatusUnprocessableEntity, "Unprocessable Content", "The catalog data or ordering is invalid.")
	default:
		slog.ErrorContext(c.Request.Context(), "catalog operation failed", "error", err)
		sharedrest.WriteProblem(c, http.StatusInternalServerError, "Internal Server Error", "The catalog operation could not be completed.")
	}
}
