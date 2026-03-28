package config

import (
	"strings"
	"testing"
)

func TestParseAndResolve(t *testing.T) {
	cfg, err := Parse([]byte(`
org:
  - name: dbc
    workload:
      - name: native
        env:
          - name: dev
            project_id: project-dev
`))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	project, err := cfg.Resolve("dbc", "native", "dev")
	if err != nil {
		t.Fatalf("resolve config: %v", err)
	}
	if project != "project-dev" {
		t.Fatalf("expected project-dev, got %q", project)
	}
}

func TestResolveErrors(t *testing.T) {
	cfg, err := Parse([]byte(`
org:
  - name: dbc
    workload:
      - name: native
        env:
          - name: dev
            project_id: project-dev
`))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	tests := []struct {
		org, workload, env, want string
	}{
		{org: "missing", workload: "native", env: "dev", want: "not found"},
		{org: "dbc", workload: "missing", env: "dev", want: "not found"},
		{org: "dbc", workload: "native", env: "missing", want: "not found"},
	}
	for _, tc := range tests {
		_, err := cfg.Resolve(tc.org, tc.workload, tc.env)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("expected error containing %q, got %v", tc.want, err)
		}
	}
}

func TestParseInvalidConfig(t *testing.T) {
	_, err := Parse([]byte("org:\n  - nope: broken\n"))
	if err == nil || !strings.Contains(err.Error(), "config parse error") {
		t.Fatalf("expected parse error, got %v", err)
	}
}
