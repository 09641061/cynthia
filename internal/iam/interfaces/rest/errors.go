package rest

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go-service/internal/iam/domain"
	sharedrest "go-service/internal/shared/rest"
)

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials), errors.Is(err, domain.ErrInvalidToken):
		sharedrest.WriteProblem(c, http.StatusUnauthorized, "Unauthorized", err.Error())
	case errors.Is(err, domain.ErrForbidden),
		errors.Is(err, domain.ErrHierarchyViolation),
		errors.Is(err, domain.ErrSelfRoleModification),
		errors.Is(err, domain.ErrOwnerProtected),
		errors.Is(err, domain.ErrPermissionNotHeld),
		errors.Is(err, domain.ErrLockoutSelfManageRoles):
		sharedrest.WriteProblem(c, http.StatusForbidden, "Forbidden", err.Error())
	case errors.Is(err, domain.ErrUserNotFound),
		errors.Is(err, domain.ErrRoleNotFound),
		errors.Is(err, domain.ErrInvitationNotFound):
		sharedrest.WriteProblem(c, http.StatusNotFound, "Not Found", err.Error())
	case errors.Is(err, domain.ErrDuplicateEmail),
		errors.Is(err, domain.ErrDuplicateInvitation),
		errors.Is(err, domain.ErrDuplicateRoleName),
		errors.Is(err, domain.ErrRoleLimitExceeded):
		sharedrest.WriteProblem(c, http.StatusConflict, "Conflict", err.Error())
	case errors.Is(err, domain.ErrValidation),
		errors.Is(err, domain.ErrInviteExpiredOrUsed):
		sharedrest.BadRequest(c, err.Error())
	default:
		sharedrest.WriteProblem(c, http.StatusInternalServerError, "Internal Server Error", "An unexpected error occurred.")
	}
}
