package rest

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"go-service/internal/calendar/domain"
	sharedrest "go-service/internal/shared/rest"
)

func writeError(c *gin.Context, err error) {
	var fieldErr domain.InvalidFieldError
	if errors.As(err, &fieldErr) {
		sharedrest.WriteProblemFields(c, http.StatusUnprocessableEntity, "Unprocessable Content", "Appointment fields are invalid.", []sharedrest.ProblemField{{
			Field:   fieldErr.Field,
			Pointer: "#/" + fieldErr.Field,
			Code:    "INVALID_FIELD",
			Reason:  fieldErr.Reason,
		}})
		return
	}
	switch {
	case errors.Is(err, domain.ErrNotFound):
		sharedrest.WriteProblem(c, http.StatusNotFound, "Not Found", "The requested appointment does not exist.")
	case errors.Is(err, domain.ErrPersonNotFound):
		sharedrest.WriteProblem(c, http.StatusNotFound, "Not Found", "The referenced person does not exist.")
	case errors.Is(err, domain.ErrCustomerNotFound):
		sharedrest.WriteProblem(c, http.StatusNotFound, "Not Found", "The referenced customer does not exist.")
	case errors.Is(err, domain.ErrServiceNotFound):
		sharedrest.WriteProblem(c, http.StatusNotFound, "Not Found", "The referenced service does not exist.")
	case errors.Is(err, domain.ErrTimeConflict):
		sharedrest.WriteProblem(c, http.StatusConflict, "Conflict", "The requested time slot conflicts with an existing appointment.")
	case errors.Is(err, domain.ErrVersionConflict):
		sharedrest.WriteProblem(c, http.StatusPreconditionFailed, "Precondition Failed", "The appointment has changed since it was last read.")
	case errors.Is(err, domain.ErrInvalid):
		sharedrest.WriteProblem(c, http.StatusUnprocessableEntity, "Unprocessable Content", "The appointment data is invalid.")
	default:
		slog.ErrorContext(c.Request.Context(), "calendar operation failed", "error", err)
		sharedrest.WriteProblem(c, http.StatusInternalServerError, "Internal Server Error", "The calendar operation could not be completed.")
	}
}
