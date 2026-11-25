package domain

// CanDeleteMember enforces permission and hierarchy rules for deleting a member.
func CanDeleteMember(actor, target Actor) error {
	if target.User.IsOwner() {
		return ErrOwnerProtected
	}
	if !actor.CanManageUsers() {
		return ErrForbidden
	}
	if actor.User.ID == target.User.ID {
		return ErrHierarchyViolation
	}
	if !actor.User.IsOwner() && target.Level() <= actor.Level() {
		return ErrHierarchyViolation
	}
	return nil
}

// CanAssignRoles enforces permission, self-modification, owner protection, and hierarchy rules.
func CanAssignRoles(actor, target Actor, roleIDs []string, rolesByID map[string]Role) error {
	if target.User.IsOwner() {
		return ErrOwnerProtected
	}
	if !actor.CanManageRoles() {
		return ErrForbidden
	}
	if !actor.User.IsOwner() && actor.User.ID == target.User.ID {
		return ErrSelfRoleModification
	}
	if !actor.User.IsOwner() && target.Level() <= actor.Level() {
		return ErrHierarchyViolation
	}
	actorLevel := actor.Level()
	for _, id := range roleIDs {
		role, ok := rolesByID[id]
		if !ok {
			return ErrRoleNotFound
		}
		if !actor.User.IsOwner() && role.Position <= actorLevel {
			return ErrHierarchyViolation
		}
	}
	return nil
}

// CanEditRole enforces hierarchy, held-permission, and anti-lockout rules when editing a role.
func CanEditRole(actor Actor, role Role, newPerms []string) error {
	if !actor.CanManageRoles() {
		return ErrForbidden
	}
	if actor.User.IsOwner() {
		return nil
	}
	if role.Position <= actor.Level() {
		return ErrHierarchyViolation
	}
	actorPerms := actor.EffectivePermissions()
	oldSet := toPermSet(role.Permissions)
	newSet := toPermSet(newPerms)

	for p := range assignablePermissions {
		if oldSet[p] != newSet[p] && !actorPerms[p] {
			return ErrPermissionNotHeld
		}
	}
	// Anti-lockout: if actor belongs to this shared role, editing it must not remove actor's own iam.manage_roles.
	if memberHasRole(actor.Roles, role.ID) {
		simulated := make([]Role, len(actor.Roles))
		for i, r := range actor.Roles {
			if r.ID == role.ID {
				r.Permissions = newPerms
			}
			simulated[i] = r
		}
		after := Actor{User: actor.User, Roles: simulated}
		if !after.CanManageRoles() {
			return ErrLockoutSelfManageRoles
		}
	}
	return nil
}

// CanDeleteRole enforces permission and hierarchy rules for deleting a role.
func CanDeleteRole(actor Actor, role Role) error {
	if !actor.CanManageRoles() {
		return ErrForbidden
	}
	if !actor.User.IsOwner() && role.Position <= actor.Level() {
		return ErrHierarchyViolation
	}
	return nil
}

func toPermSet(perms []string) map[string]bool {
	out := make(map[string]bool, len(perms))
	for _, p := range perms {
		out[p] = true
	}
	return out
}

func memberHasRole(roles []Role, roleID string) bool {
	for _, r := range roles {
		if r.ID == roleID {
			return true
		}
	}
	return false
}
