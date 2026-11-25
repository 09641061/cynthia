package casbin

import (
	"context"
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"go-service/internal/iam/domain"
)

const policyModel = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, "owner") || g(r.sub, "iam.administrator") || (g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act)
`

// ActorLoader loads the authoritative user and roles state from PostgreSQL.
type ActorLoader interface {
	GetActorByID(ctx context.Context, userID string) (domain.Actor, error)
}

// Authorizer enforces Casbin authorization against fresh PostgreSQL IAM state.
type Authorizer struct {
	loader ActorLoader
	model  model.Model
}

// NewAuthorizer parses the Casbin model once and returns a fail-closed Authorizer.
func NewAuthorizer(loader ActorLoader) (*Authorizer, error) {
	m, err := model.NewModelFromString(policyModel)
	if err != nil {
		return nil, fmt.Errorf("parse casbin model: %w", err)
	}
	return &Authorizer{loader: loader, model: m}, nil
}

// Enforce loads current user policies from PostgreSQL and evaluates (userID, obj, act).
// Any loader or Casbin error fails closed (returns false, err).
func (a *Authorizer) Enforce(ctx context.Context, userID, obj, act string) (bool, error) {
	if a == nil || a.loader == nil || userID == "" {
		return false, domain.ErrForbidden
	}
	actor, err := a.loader.GetActorByID(ctx, userID)
	if err != nil {
		return false, err
	}
	enforcer, err := casbin.NewEnforcer(a.model.Copy())
	if err != nil {
		return false, fmt.Errorf("create casbin enforcer: %w", err)
	}
	if err := loadActorPolicies(enforcer, actor); err != nil {
		return false, err
	}
	allowed, err := enforcer.Enforce(actor.User.ID, obj, act)
	if err != nil {
		return false, fmt.Errorf("casbin enforce: %w", err)
	}
	return allowed, nil
}

func loadActorPolicies(e *casbin.Enforcer, actor domain.Actor) error {
	if actor.User.IsOwner() {
		if _, err := e.AddGroupingPolicy(actor.User.ID, "owner"); err != nil {
			return err
		}
		return nil
	}
	for _, role := range actor.Roles {
		roleSub := "role:" + role.ID
		if _, err := e.AddGroupingPolicy(actor.User.ID, roleSub); err != nil {
			return err
		}
		for _, perm := range role.Permissions {
			if perm == domain.PermAdministrator {
				if _, err := e.AddGroupingPolicy(roleSub, "iam.administrator"); err != nil {
					return err
				}
				continue
			}
			pObj, pAct, ok := strings.Cut(perm, ".")
			if !ok {
				continue
			}
			if _, err := e.AddPolicy(roleSub, pObj, pAct); err != nil {
				return err
			}
		}
	}
	return nil
}
