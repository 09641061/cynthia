package rest

import (
	"sort"
	"time"

	"go-service/internal/iam/application"
	"go-service/internal/iam/domain"
)

// RoleResource is the JSON representation of an IAM role.
type RoleResource struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Position    int      `json:"position"`
	Permissions []string `json:"permissions"`
}

// ActorResource is the JSON representation of an authenticated user or member.
type ActorResource struct {
	ID                   string         `json:"id"`
	Email                string         `json:"email"`
	FirstName            string         `json:"first_name"`
	LastName             string         `json:"last_name"`
	Type                 string         `json:"type"`
	Level                int            `json:"level"`
	Roles                []RoleResource `json:"roles"`
	EffectivePermissions []string       `json:"effective_permissions"`
}

// AuthResponse is returned on login, refresh, and invitation acceptance.
type AuthResponse struct {
	AccessToken  string         `json:"access_token"`
	RefreshToken string         `json:"refresh_token"`
	TokenType    string         `json:"token_type"`
	ExpiresIn    int64          `json:"expires_in"`
	User         *ActorResource `json:"user,omitempty"`
}

// InvitationResource is the JSON representation of a pending invitation.
type InvitationResource struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func toRoleResource(r domain.Role) RoleResource {
	perms := r.Permissions
	if perms == nil {
		perms = []string{}
	}
	return RoleResource{
		ID: r.ID, Name: r.Name,
		Position: r.Position, Permissions: perms,
	}
}

func toActorResource(a domain.Actor) ActorResource {
	roles := make([]RoleResource, 0, len(a.Roles))
	for _, r := range a.Roles {
		roles = append(roles, toRoleResource(r))
	}
	effMap := a.EffectivePermissions()
	eff := make([]string, 0, len(effMap))
	for k, ok := range effMap {
		if ok {
			eff = append(eff, k)
		}
	}
	sort.Strings(eff)
	return ActorResource{
		ID: a.User.ID, Email: a.User.Email, FirstName: a.User.FirstName, LastName: a.User.LastName,
		Type: string(a.User.Type), Level: a.Level(), Roles: roles, EffectivePermissions: eff,
	}
}

func toInvitationResource(inv application.PendingInvite) InvitationResource {
	return InvitationResource{
		ID: inv.ID, Email: inv.Email, FirstName: inv.FirstName, LastName: inv.LastName,
		CreatedAt: inv.CreatedAt, ExpiresAt: inv.ExpiresAt,
	}
}
