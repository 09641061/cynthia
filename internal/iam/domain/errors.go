package domain

import "errors"

var (
	ErrUserNotFound           = errors.New("user not found")
	ErrRoleNotFound           = errors.New("role not found")
	ErrInvitationNotFound     = errors.New("invitation not found")
	ErrInvalidCredentials     = errors.New("invalid email or password")
	ErrInvalidToken           = errors.New("invalid or expired token")
	ErrForbidden              = errors.New("forbidden")
	ErrHierarchyViolation     = errors.New("hierarchy level prevents this operation")
	ErrSelfRoleModification   = errors.New("cannot directly modify own role memberships")
	ErrOwnerProtected         = errors.New("owner account cannot be deleted or assigned roles")
	ErrRoleLimitExceeded      = errors.New("maximum number of active roles reached")
	ErrDuplicateEmail         = errors.New("email is already registered")
	ErrDuplicateInvitation    = errors.New("pending invitation already exists")
	ErrDuplicateRoleName      = errors.New("role name is already in use")
	ErrValidation             = errors.New("validation failed")
	ErrPermissionNotHeld      = errors.New("cannot toggle a permission you do not hold")
	ErrLockoutSelfManageRoles = errors.New("cannot remove your own effective role management capability")
	ErrInviteExpiredOrUsed    = errors.New("invitation is invalid, expired, or already used")
)
