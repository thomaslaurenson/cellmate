package cmd

import "testing"

func TestVersionFrom(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		injected string
		module   string
		want     string
	}{
		{name: "ldflags wins", injected: "1.2.3", module: "v9.9.9", want: "1.2.3"},
		{name: "go install tag", injected: devVersion, module: "v1.2.3", want: "1.2.3"},
		{name: "working tree", injected: devVersion, module: "(devel)", want: devVersion},
		{name: "dirty pseudo", injected: devVersion, module: "v1.2.4-0.20260101120000-abcdef123456+dirty", want: devVersion},
		{name: "pseudo after tag", injected: devVersion, module: "v1.2.4-0.20260101120000-abcdef123456", want: devVersion},
		{name: "pseudo without tag", injected: devVersion, module: "v0.0.0-20260101120000-abcdef123456", want: devVersion},
		{name: "empty", injected: devVersion, module: "", want: devVersion},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := versionFrom(tc.injected, tc.module); got != tc.want {
				t.Errorf("versionFrom(%q, %q) = %q, want %q", tc.injected, tc.module, got, tc.want)
			}
		})
	}
}
