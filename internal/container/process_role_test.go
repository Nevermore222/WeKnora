package container

import "testing"

func TestProcessRoleFromEnv(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want processRole
		bad  bool
	}{
		{name: "default active", raw: "", want: processRoleActive},
		{name: "explicit active", raw: "active", want: processRoleActive},
		{name: "case insensitive passive", raw: "PASSIVE", want: processRolePassive},
		{name: "unknown rejected", raw: "writer", bad: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(processRoleEnv, tc.raw)

			got, err := processRoleFromEnv()
			if tc.bad {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("processRoleFromEnv() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("processRoleFromEnv() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBackgroundServicesEnabled(t *testing.T) {
	if !backgroundServicesEnabled(processRoleActive) {
		t.Fatal("active process must enable background services")
	}
	if backgroundServicesEnabled(processRolePassive) {
		t.Fatal("passive process must disable background services")
	}
}
