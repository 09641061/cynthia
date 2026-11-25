package rest

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"go-service/internal/organizations/application"
	"go-service/internal/organizations/domain"
	sharedrest "go-service/internal/shared/rest"
)

func writeError(c *gin.Context, err error) {
	var fieldErr domain.InvalidFieldError
	if errors.As(err, &fieldErr) {
		sharedrest.WriteProblemFields(c, http.StatusUnprocessableEntity, "Unprocessable Content", "Organization fields are invalid.", []sharedrest.ProblemField{{
			Field: fieldErr.Field, Pointer: "#/" + fieldErr.Field, Code: "INVALID_FIELD", Reason: fieldErr.Reason,
		}})
		return
	}
	switch {
	case errors.Is(err, application.ErrNotFound):
		sharedrest.WriteProblem(c, http.StatusNotFound, "Not Found", "The organization profile does not exist.")
	case errors.Is(err, domain.ErrInvalid):
		sharedrest.WriteProblem(c, http.StatusUnprocessableEntity, "Unprocessable Content", "The organization data is invalid.")
	default:
		slog.ErrorContext(c.Request.Context(), "organization operation failed", "error", err)
		sharedrest.WriteProblem(c, http.StatusInternalServerError, "Internal Server Error", "The organization operation could not be completed.")
	}
}
