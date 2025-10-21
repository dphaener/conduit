package auth

// Permission represents a specific action that can be performed on a resource
type RBACPermission string

const (
	// Post permissions
	PostsRead   RBACPermission = "posts.read"
	PostsCreate RBACPermission = "posts.create"
	PostsUpdate RBACPermission = "posts.update"
	PostsDelete RBACPermission = "posts.delete"

	// User permissions
	UsersRead   RBACPermission = "users.read"
	UsersCreate RBACPermission = "users.create"
	UsersUpdate RBACPermission = "users.update"
	UsersDelete RBACPermission = "users.delete"

	// System permissions
	SystemAdmin RBACPermission = "system.admin"
)

// Role represents a user role with a set of permissions
type Role struct {
	Name        string
	Permissions []RBACPermission
}

// HasPermission checks if the role has a specific permission
func (r *Role) HasPermission(permission RBACPermission) bool {
	for _, p := range r.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// Predefined roles
var (
	// AdminRole has all permissions
	AdminRole = &Role{
		Name: "admin",
		Permissions: []RBACPermission{
			PostsRead, PostsCreate, PostsUpdate, PostsDelete,
			UsersRead, UsersCreate, UsersUpdate, UsersDelete,
			SystemAdmin,
		},
	}

	// EditorRole can read, create, and update posts, but not delete
	EditorRole = &Role{
		Name: "editor",
		Permissions: []RBACPermission{
			PostsRead, PostsCreate, PostsUpdate,
			UsersRead,
		},
	}

	// ViewerRole can only read posts
	ViewerRole = &Role{
		Name: "viewer",
		Permissions: []RBACPermission{
			PostsRead,
		},
	}
)

// GetRoleByName returns a predefined role by name
// Returns nil if the role is not found
func GetRoleByName(name string) *Role {
	switch name {
	case "admin":
		return AdminRole
	case "editor":
		return EditorRole
	case "viewer":
		return ViewerRole
	default:
		return nil
	}
}

// UserHasPermission checks if any of the user's roles has the required permission
func UserHasPermission(roles []string, permission RBACPermission) bool {
	for _, roleName := range roles {
		role := GetRoleByName(roleName)
		if role != nil && role.HasPermission(permission) {
			return true
		}
	}
	return false
}
