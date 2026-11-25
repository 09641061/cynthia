package domain

const (
	PermManageUsers        = "iam.manage_users"
	PermManageRoles        = "iam.manage_roles"
	PermAdministrator      = "iam.administrator"
	PermCatalogManage      = "catalog.manage"
	PermCRMManage          = "crm.manage"
	PermCalendarManage     = "calendar.manage"
	PermOrganizationManage = "organization.manage"
	PermSchedulingAppear   = "scheduling.appear"
)

var assignablePermissions = map[string]bool{
	PermManageUsers:        true,
	PermManageRoles:        true,
	PermAdministrator:      true,
	PermCatalogManage:      true,
	PermCRMManage:          true,
	PermCalendarManage:     true,
	PermOrganizationManage: true,
	PermSchedulingAppear:   true,
}

// IsAssignablePermission reports whether perm can be enabled on a role.
func IsAssignablePermission(perm string) bool {
	return assignablePermissions[perm]
}

// NormalizePermissions validates and deduplicates a slice of role permissions.
func NormalizePermissions(perms []string) ([]string, error) {
	seen := make(map[string]bool, len(perms))
	out := make([]string, 0, len(perms))
	for _, p := range perms {
		if !IsAssignablePermission(p) {
			return nil, ErrValidation
		}
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out, nil
}
