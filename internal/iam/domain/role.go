package domain

import (
	"strings"
	"time"
)

// MaxActiveRoles is the maximum number of active roles allowed in the organization.
const MaxActiveRoles = 50

// Role represents a permission group in the IAM hierarchy.
type Role struct {
	ID          string
	Name        string
	Position    int
	Permissions []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ValidateRoleInput validates and normalizes role name and permissions.
func ValidateRoleInput(name string, perms []string) (string, []string, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" || len(trimmedName) > 80 {
		return "", nil, ErrValidation
	}
	normalized, err := NormalizePermissions(perms)
	if err != nil {
		return "", nil, err
	}
	return trimmedName, normalized, nil
}

// HasPermission reports whether this role directly grants perm (or iam.administrator).
func (r Role) HasPermission(perm string) bool {
	for _, p := range r.Permissions {
		if p == perm || (p == PermAdministrator && perm != PermSchedulingAppear) {
			return true
		}
	}
	return false
}
