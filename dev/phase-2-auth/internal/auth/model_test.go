package auth

import "testing"

func TestParseRoleAcceptsValidRoles(t *testing.T) {
	cases := map[string]Role{"ADMIN": RoleAdmin, "OPERATOR": RoleOperator}
	for in, want := range cases {
		got, err := ParseRole(in)
		if err != nil {
			t.Fatalf("ParseRole(%q) returned error: %v", in, err)
		}
		if got != want {
			t.Fatalf("ParseRole(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseRoleRejectsInvalidRole(t *testing.T) {
	for _, in := range []string{"", "admin", "SUPERUSER", "root"} {
		if _, err := ParseRole(in); err == nil {
			t.Fatalf("ParseRole(%q) expected error, got nil", in)
		}
	}
}
