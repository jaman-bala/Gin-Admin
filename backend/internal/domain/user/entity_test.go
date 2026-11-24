package user

import "testing"

func TestRoleLevel(t *testing.T) {
	tests := []struct {
		role Role
		want int
	}{
		{RoleSuperuser, LevelSuperuser},
		{RoleAdmin, LevelAdmin},
		{RoleUser, LevelUser},
		{Role(""), LevelNone},
		{Role("root"), LevelNone},
		{Role("Admin"), LevelNone}, // case-sensitive: no privilege through casing
		{Role("admin "), LevelNone},
	}
	for _, tc := range tests {
		if got := tc.role.Level(); got != tc.want {
			t.Errorf("Role(%q).Level() = %d, want %d", tc.role, got, tc.want)
		}
	}
	if LevelUser >= LevelAdmin || LevelAdmin >= LevelSuperuser || LevelNone >= LevelUser {
		t.Error("levels must be strictly ordered none < user < admin < superuser")
	}
}
