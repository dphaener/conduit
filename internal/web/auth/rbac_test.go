package auth

import (
	"testing"
)

func TestRoleHasPermission(t *testing.T) {
	tests := []struct {
		name       string
		role       *Role
		permission RBACPermission
		want       bool
	}{
		{
			name:       "admin has posts.create",
			role:       AdminRole,
			permission: PostsCreate,
			want:       true,
		},
		{
			name:       "admin has posts.delete",
			role:       AdminRole,
			permission: PostsDelete,
			want:       true,
		},
		{
			name:       "admin has system.admin",
			role:       AdminRole,
			permission: SystemAdmin,
			want:       true,
		},
		{
			name:       "editor has posts.create",
			role:       EditorRole,
			permission: PostsCreate,
			want:       true,
		},
		{
			name:       "editor does not have posts.delete",
			role:       EditorRole,
			permission: PostsDelete,
			want:       false,
		},
		{
			name:       "editor does not have system.admin",
			role:       EditorRole,
			permission: SystemAdmin,
			want:       false,
		},
		{
			name:       "viewer has posts.read",
			role:       ViewerRole,
			permission: PostsRead,
			want:       true,
		},
		{
			name:       "viewer does not have posts.create",
			role:       ViewerRole,
			permission: PostsCreate,
			want:       false,
		},
		{
			name:       "viewer does not have posts.delete",
			role:       ViewerRole,
			permission: PostsDelete,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.role.HasPermission(tt.permission)
			if got != tt.want {
				t.Errorf("Role.HasPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRoleByName(t *testing.T) {
	tests := []struct {
		name     string
		roleName string
		want     *Role
	}{
		{
			name:     "gets admin role",
			roleName: "admin",
			want:     AdminRole,
		},
		{
			name:     "gets editor role",
			roleName: "editor",
			want:     EditorRole,
		},
		{
			name:     "gets viewer role",
			roleName: "viewer",
			want:     ViewerRole,
		},
		{
			name:     "returns nil for unknown role",
			roleName: "unknown",
			want:     nil,
		},
		{
			name:     "returns nil for empty string",
			roleName: "",
			want:     nil,
		},
		{
			name:     "case sensitive - Admin vs admin",
			roleName: "Admin",
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRoleByName(tt.roleName)
			if got != tt.want {
				t.Errorf("GetRoleByName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserHasPermission(t *testing.T) {
	tests := []struct {
		name       string
		roles      []string
		permission RBACPermission
		want       bool
	}{
		{
			name:       "admin user has posts.create",
			roles:      []string{"admin"},
			permission: PostsCreate,
			want:       true,
		},
		{
			name:       "editor user has posts.create",
			roles:      []string{"editor"},
			permission: PostsCreate,
			want:       true,
		},
		{
			name:       "viewer user does not have posts.create",
			roles:      []string{"viewer"},
			permission: PostsCreate,
			want:       false,
		},
		{
			name:       "user with multiple roles has permission from any role",
			roles:      []string{"viewer", "editor"},
			permission: PostsCreate,
			want:       true,
		},
		{
			name:       "user with no roles has no permissions",
			roles:      []string{},
			permission: PostsRead,
			want:       false,
		},
		{
			name:       "user with unknown role has no permissions",
			roles:      []string{"unknown"},
			permission: PostsRead,
			want:       false,
		},
		{
			name:       "admin has all permissions",
			roles:      []string{"admin"},
			permission: UsersDelete,
			want:       true,
		},
		{
			name:       "editor can read users",
			roles:      []string{"editor"},
			permission: UsersRead,
			want:       true,
		},
		{
			name:       "editor cannot delete users",
			roles:      []string{"editor"},
			permission: UsersDelete,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UserHasPermission(tt.roles, tt.permission)
			if got != tt.want {
				t.Errorf("UserHasPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPredefinedRoles(t *testing.T) {
	// Test AdminRole
	t.Run("AdminRole properties", func(t *testing.T) {
		if AdminRole.Name != "admin" {
			t.Errorf("AdminRole.Name = %v, want admin", AdminRole.Name)
		}

		expectedAdminPerms := []RBACPermission{
			PostsRead, PostsCreate, PostsUpdate, PostsDelete,
			UsersRead, UsersCreate, UsersUpdate, UsersDelete,
			SystemAdmin,
		}

		for _, perm := range expectedAdminPerms {
			if !AdminRole.HasPermission(perm) {
				t.Errorf("AdminRole should have permission %v", perm)
			}
		}
	})

	// Test EditorRole
	t.Run("EditorRole properties", func(t *testing.T) {
		if EditorRole.Name != "editor" {
			t.Errorf("EditorRole.Name = %v, want editor", EditorRole.Name)
		}

		expectedEditorPerms := []RBACPermission{
			PostsRead, PostsCreate, PostsUpdate,
			UsersRead,
		}

		for _, perm := range expectedEditorPerms {
			if !EditorRole.HasPermission(perm) {
				t.Errorf("EditorRole should have permission %v", perm)
			}
		}

		// Editor should NOT have these permissions
		forbiddenPerms := []RBACPermission{PostsDelete, UsersCreate, UsersUpdate, UsersDelete, SystemAdmin}
		for _, perm := range forbiddenPerms {
			if EditorRole.HasPermission(perm) {
				t.Errorf("EditorRole should NOT have permission %v", perm)
			}
		}
	})

	// Test ViewerRole
	t.Run("ViewerRole properties", func(t *testing.T) {
		if ViewerRole.Name != "viewer" {
			t.Errorf("ViewerRole.Name = %v, want viewer", ViewerRole.Name)
		}

		// Viewer should only have posts.read
		if !ViewerRole.HasPermission(PostsRead) {
			t.Error("ViewerRole should have posts.read permission")
		}

		// Viewer should NOT have any other permissions
		forbiddenPerms := []RBACPermission{
			PostsCreate, PostsUpdate, PostsDelete,
			UsersRead, UsersCreate, UsersUpdate, UsersDelete,
			SystemAdmin,
		}
		for _, perm := range forbiddenPerms {
			if ViewerRole.HasPermission(perm) {
				t.Errorf("ViewerRole should NOT have permission %v", perm)
			}
		}
	})
}

func TestPermissionConstants(t *testing.T) {
	tests := []struct {
		permission RBACPermission
		expected   string
	}{
		{PostsRead, "posts.read"},
		{PostsCreate, "posts.create"},
		{PostsUpdate, "posts.update"},
		{PostsDelete, "posts.delete"},
		{UsersRead, "users.read"},
		{UsersCreate, "users.create"},
		{UsersUpdate, "users.update"},
		{UsersDelete, "users.delete"},
		{SystemAdmin, "system.admin"},
	}

	for _, tt := range tests {
		t.Run(string(tt.permission), func(t *testing.T) {
			if string(tt.permission) != tt.expected {
				t.Errorf("Permission constant = %v, want %v", tt.permission, tt.expected)
			}
		})
	}
}

func TestRoleImmutability(t *testing.T) {
	// Ensure predefined roles are not accidentally modified
	originalAdminPermsCount := len(AdminRole.Permissions)
	originalEditorPermsCount := len(EditorRole.Permissions)
	originalViewerPermsCount := len(ViewerRole.Permissions)

	// Get roles multiple times
	role1 := GetRoleByName("admin")
	role2 := GetRoleByName("admin")

	// They should be the same instance
	if role1 != role2 {
		t.Error("GetRoleByName should return the same instance for the same role")
	}

	// Verify permission counts haven't changed
	if len(AdminRole.Permissions) != originalAdminPermsCount {
		t.Error("AdminRole permissions were modified")
	}
	if len(EditorRole.Permissions) != originalEditorPermsCount {
		t.Error("EditorRole permissions were modified")
	}
	if len(ViewerRole.Permissions) != originalViewerPermsCount {
		t.Error("ViewerRole permissions were modified")
	}
}

func TestUserHasPermissionMultipleRoles(t *testing.T) {
	// Test that a user with multiple roles gets permissions from all roles
	roles := []string{"viewer", "editor", "admin"}

	// Should have all admin permissions since admin is included
	adminPerms := []RBACPermission{
		PostsRead, PostsCreate, PostsUpdate, PostsDelete,
		UsersRead, UsersCreate, UsersUpdate, UsersDelete,
		SystemAdmin,
	}

	for _, perm := range adminPerms {
		if !UserHasPermission(roles, perm) {
			t.Errorf("User with admin role should have permission %v", perm)
		}
	}
}

func BenchmarkUserHasPermission(b *testing.B) {
	roles := []string{"editor"}
	permission := PostsCreate

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = UserHasPermission(roles, permission)
	}
}

func BenchmarkGetRoleByName(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetRoleByName("admin")
	}
}
