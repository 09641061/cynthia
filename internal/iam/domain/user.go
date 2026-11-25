package domain

import (
	"math"
	"net/mail"
	"strings"
	"time"
)

// BaseLevel is the hierarchical level of a Member with no assigned roles.
const BaseLevel = math.MaxInt32

// OwnerLevel is above all roles (position >= 1).
const OwnerLevel = 0

// UserType distinguishes the single organization owner from regular members.
type UserType string

const (
	UserTypeOwner UserType = "owner"
	UserTypeUser  UserType = "user"
)

// User represents an internal IAM account (Owner or Member).
type User struct {
	ID           string
	Email        string
	FirstName    string
	LastName     string
	PasswordHash string
	Type         UserType
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsOwner reports whether this user is the organization owner.
func (u User) IsOwner() bool {
	return u.Type == UserTypeOwner
}

// FullName returns the user's trimmed display name.
func (u User) FullName() string {
	return strings.TrimSpace(u.FirstName + " " + u.LastName)
}

// Actor pairs a User with their assigned Roles for authorization and hierarchy checks.
type Actor struct {
	User  User
	Roles []Role
}

// Level returns the actor's hierarchical priority (lower number = higher authority).
// Owner is 0; a Member's level is the minimum Position across their roles, or BaseLevel if none.
func (a Actor) Level() int {
	if a.User.IsOwner() {
		return OwnerLevel
	}
	best := BaseLevel
	for _, r := range a.Roles {
		if r.Position > 0 && r.Position < best {
			best = r.Position
		}
	}
	return best
}

// EffectivePermissions computes the union of all permissions granted by the actor's roles.
// Owner and Administrator grant all functional permissions except scheduling.appear.
func (a Actor) EffectivePermissions() map[string]bool {
	perms := make(map[string]bool)
	if a.User.IsOwner() {
		for p := range assignablePermissions {
			if p != PermSchedulingAppear {
				perms[p] = true
			}
		}
		return perms
	}
	for _, r := range a.Roles {
		for _, p := range r.Permissions {
			perms[p] = true
			if p == PermAdministrator {
				for ap := range assignablePermissions {
					if ap != PermSchedulingAppear {
						perms[ap] = true
					}
				}
			}
		}
	}
	return perms
}

// HasPermission checks whether the actor holds perm.
func (a Actor) HasPermission(perm string) bool {
	return a.EffectivePermissions()[perm]
}

// CanManageRoles reports whether the actor holds iam.manage_roles (or Owner/Administrator).
func (a Actor) CanManageRoles() bool {
	return a.HasPermission(PermManageRoles)
}

// CanManageUsers reports whether the actor holds iam.manage_users (or Owner/Administrator).
func (a Actor) CanManageUsers() bool {
	return a.HasPermission(PermManageUsers)
}

// ValidateNameParts validates and trims firstName and lastName.
func ValidateNameParts(firstName, lastName string) (string, string, error) {
	fn := strings.TrimSpace(firstName)
	ln := strings.TrimSpace(lastName)
	if fn == "" || ln == "" || len(fn) > 80 || len(ln) > 80 {
		return "", "", ErrValidation
	}
	return fn, ln, nil
}

// NormalizeEmail trims, lowercases, and validates an email address.
func NormalizeEmail(raw string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" || len(trimmed) > 254 {
		return "", ErrValidation
	}
	addr, err := mail.ParseAddress(trimmed)
	if err != nil || strings.ToLower(addr.Address) != trimmed {
		return "", ErrValidation
	}
	return trimmed, nil
}
