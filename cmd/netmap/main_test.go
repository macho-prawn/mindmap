package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"netmap/internal/model"
	"netmap/internal/provider"
)

type stubDiscoveryProvider struct{}

func (stubDiscoveryProvider) ListDedicatedInterconnects(context.Context, string) ([]model.DedicatedInterconnect, error) {
	return nil, nil
}

func (stubDiscoveryProvider) ListVLANAttachments(context.Context, string) ([]model.VLANAttachment, error) {
	return nil, nil
}

func (stubDiscoveryProvider) ListVPNGateways(context.Context, string) ([]model.VPNGateway, error) {
	return nil, nil
}

func (stubDiscoveryProvider) ListTargetVPNGateways(context.Context, string) ([]model.VPNGateway, error) {
	return nil, nil
}

func (stubDiscoveryProvider) ListVPNTunnels(context.Context, string) ([]model.VPNTunnel, error) {
	return nil, nil
}

func (stubDiscoveryProvider) ListCloudRouters(context.Context, string) ([]model.CloudRouter, error) {
	return nil, nil
}

func (stubDiscoveryProvider) GetCloudRouterStatus(context.Context, string, string, string) (model.RouterStatus, error) {
	return model.RouterStatus{}, nil
}

func TestRunVersionCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{"version"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected success exit code, got %d", exitCode)
	}
	if strings.TrimSpace(stdout.String()) != "v2.0.0" {
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

func TestRunInvalidArgsFailsWithoutProvider(t *testing.T) {
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

	exitCode := run(context.Background(), []string{"-o", "dbc"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected error exit code, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "missing mandatory parameter -t") {
		t.Fatalf("expected validation error, got %q", stderr.String())
	}
}

func TestRunUnreadableConfigFailsWithoutProvider(t *testing.T) {
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

	exitCode := run(context.Background(), []string{
		"-t", "interconnect",
		"-o", "dbc",
		"-p", "src-project",
		"-c", filepath.Join(t.TempDir(), "missing.yaml"),
	}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected error exit code, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "read config") {
		t.Fatalf("expected config read error, got %q", stderr.String())
	}
}

func TestRunInvalidConfigFailsWithoutProvider(t *testing.T) {
	restoreProvider := newComputeProvider
	newComputeProvider = func(context.Context) (provider.DiscoveryProvider, error) {
		t.Fatal("provider should not be created")
		return nil, nil
	}
	t.Cleanup(func() {
		newComputeProvider = restoreProvider
	})

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("org:\n  - name:\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{
		"-t", "interconnect",
		"-o", "dbc",
		"-p", "src-project",
		"-c", configPath,
	}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected error exit code, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "cannot be empty") {
		t.Fatalf("expected config parse error, got %q", stderr.String())
	}
}

func TestRunUnresolvedSelectorFailsWithoutProvider(t *testing.T) {
	restoreProvider := newComputeProvider
	newComputeProvider = func(context.Context) (provider.DiscoveryProvider, error) {
		t.Fatal("provider should not be created")
		return nil, nil
	}
	t.Cleanup(func() {
		newComputeProvider = restoreProvider
	})

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("org:\n  - name: dbc\n    workload:\n      - name: native\n        env:\n          - name: dev\n            project_id: project\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{
		"-t", "interconnect",
		"-o", "missing",
		"-p", "src-project",
		"-c", configPath,
	}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected error exit code, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "not found in config") {
		t.Fatalf("expected selector resolution error, got %q", stderr.String())
	}
}

func TestRunValidConfigCreatesProviderAfterPreflight(t *testing.T) {
	restoreProvider := newComputeProvider
	providerCreated := false
	newComputeProvider = func(context.Context) (provider.DiscoveryProvider, error) {
		providerCreated = true
		return stubDiscoveryProvider{}, nil
	}
	t.Cleanup(func() {
		newComputeProvider = restoreProvider
	})

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("org:\n  - name: dbc\n    workload:\n      - name: native\n        env:\n          - name: dev\n            project_id: project\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{
		"-t", "interconnect",
		"-o", "dbc",
		"-w", "native",
		"-e", "dev",
		"-p", "src-project",
		"-c", configPath,
	}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected error exit code because stub provider returns no interconnects, got %d", exitCode)
	}
	if !providerCreated {
		t.Fatalf("expected provider to be created after successful preflight")
	}
	if !strings.Contains(stderr.String(), "no dedicated interconnects found in source project") {
		t.Fatalf("expected post-provider execution error, got %q", stderr.String())
	}
}
