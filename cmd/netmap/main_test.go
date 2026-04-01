package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"netmap/internal/provider"
)

func TestRunVersionCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{"version"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected success exit code, got %d", exitCode)
	}
	if strings.TrimSpace(stdout.String()) != "1.1.0" {
		t.Fatalf("expected version output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunWithoutArgsShowsHelpWithoutProvider(t *testing.T) {
	restoreProvider := newComputeProvider
	newComputeProvider = func(context.Context) (provider.DiscoveryProvider, error) {
		t.Fatal("provider should not be created")
		return nil, nil
	}
	t.Cleanup(func() {
		newComputeProvider = restoreProvider
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected success exit code, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "Usage:") || !strings.Contains(stdout.String(), "netmap version") {
		t.Fatalf("expected help output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunHelpFlagShowsHelpWithoutProvider(t *testing.T) {
	restoreProvider := newComputeProvider
	newComputeProvider = func(context.Context) (provider.DiscoveryProvider, error) {
		t.Fatal("provider should not be created")
		return nil, nil
	}
	t.Cleanup(func() {
		newComputeProvider = restoreProvider
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{"-h"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected success exit code, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "Usage:") || !strings.Contains(stdout.String(), "netmap version") {
		t.Fatalf("expected help output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunVersionCommandRejectsExtraArgs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{"version", "extra"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected error exit code, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "version command does not accept additional arguments") {
		t.Fatalf("expected version command error, got %q", stderr.String())
	}
}
