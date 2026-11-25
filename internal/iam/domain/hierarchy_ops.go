package domain

// CanReorderRoles ensures orderedIDs is a valid permutation and that a Member only reorders roles strictly below actor.Level().
func CanReorderRoles(actor Actor, orderedIDs []string, currentRoles []Role) error {
	if !actor.CanManageRoles() {
		return ErrForbidden
	}
	if len(orderedIDs) != len(currentRoles) {
		return ErrValidation
	}
	currentPos := make(map[string]int, len(currentRoles))
	for _, r := range currentRoles {
		currentPos[r.ID] = r.Position
	}
	seen := make(map[string]bool, len(orderedIDs))
	actorLevel := actor.Level()
	for idx, id := range orderedIDs {
		oldPos, ok := currentPos[id]
		if !ok || seen[id] {
			return ErrValidation
		}
		seen[id] = true
		newPos := idx + 1
		if !actor.User.IsOwner() && oldPos != newPos {
			if oldPos <= actorLevel || newPos <= actorLevel {
				return ErrHierarchyViolation
			}
		}
	}
	return nil
}
