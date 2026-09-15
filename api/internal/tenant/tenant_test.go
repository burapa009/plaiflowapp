package tenant

import "testing"

func TestFixedRolePermissions(t *testing.T) {
	tests := []struct {
		role    Role
		action  Action
		allowed bool
	}{
		{Owner, InviteMember, true}, {Admin, InviteMember, true}, {Member, InviteMember, false},
		{Owner, ManageGroup, true}, {Admin, ManageGroup, true}, {Member, ManageGroup, false},
		{Owner, ManageRoles, true}, {Admin, ManageRoles, false}, {Member, ManageRoles, false},
		{Owner, TransferOwnership, true}, {Admin, TransferOwnership, false},
		{Owner, RemoveMember, true}, {Admin, RemoveMember, true}, {Member, RemoveMember, false},
	}
	for _, tt := range tests {
		if got := tt.role.Allows(tt.action); got != tt.allowed {
			t.Fatalf("%s allows %s = %v", tt.role, tt.action, got)
		}
	}
}
