package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"netmap/internal/model"
	"netmap/internal/provider"

	"google.golang.org/api/googleapi"
)

func init() {
	adcPreflight = func(context.Context) error { return nil }
}

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

type errorDiscoveryProvider struct {
	interconnectErr error
	vpnGatewayErr   error
}

func (p errorDiscoveryProvider) ListDedicatedInterconnects(context.Context, string) ([]model.DedicatedInterconnect, error) {
	return nil, p.interconnectErr
}

func (errorDiscoveryProvider) ListVLANAttachments(context.Context, string) ([]model.VLANAttachment, error) {
	return nil, nil
}

func (p errorDiscoveryProvider) ListVPNGateways(context.Context, string) ([]model.VPNGateway, error) {
	return nil, p.vpnGatewayErr
}

func (errorDiscoveryProvider) ListTargetVPNGateways(context.Context, string) ([]model.VPNGateway, error) {
	return nil, nil
}

func (errorDiscoveryProvider) ListVPNTunnels(context.Context, string) ([]model.VPNTunnel, error) {
	return nil, nil
}

func (errorDiscoveryProvider) ListCloudRouters(context.Context, string) ([]model.CloudRouter, error) {
	return nil, nil
}

func (errorDiscoveryProvider) GetCloudRouterStatus(context.Context, string, string, string) (model.RouterStatus, error) {
	return model.RouterStatus{}, nil
}

func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}

func stubADCPreflight(t *testing.T, fn func(context.Context) error) {
	t.Helper()
	restore := adcPreflight
	adcPreflight = fn
	t.Cleanup(func() {
		adcPreflight = restore
	})
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

func TestRunProviderCreationAuthErrorShowsCustomADCGuidance(t *testing.T) {
	stubADCPreflight(t, func(context.Context) error { return nil })
	restoreProvider := newComputeProvider
	newComputeProvider = func(context.Context) (provider.DiscoveryProvider, error) {
		return nil, errors.New("create compute service: google: could not find default credentials")
	}
	t.Cleanup(func() {
		newComputeProvider = restoreProvider
	})

	configPath := writeTempConfig(t, "org:\n  - name: dbc\n    workload:\n      - name: native\n        env:\n          - name: dev\n            project_id: project\n")

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
		t.Fatalf("expected error exit code, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), adcGuidanceLine) {
		t.Fatalf("expected ADC guidance line, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), adcGuidanceCommand) {
		t.Fatalf("expected ADC guidance command, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Underlying error: create compute service: google: could not find default credentials") {
		t.Fatalf("expected raw provider error, got %q", stderr.String())
	}
}

func TestRunProviderCreationNonAuthErrorRemainsUnchanged(t *testing.T) {
	stubADCPreflight(t, func(context.Context) error { return nil })
	restoreProvider := newComputeProvider
	newComputeProvider = func(context.Context) (provider.DiscoveryProvider, error) {
		return nil, errors.New("create compute service: boom")
	}
	t.Cleanup(func() {
		newComputeProvider = restoreProvider
	})

	configPath := writeTempConfig(t, "org:\n  - name: dbc\n    workload:\n      - name: native\n        env:\n          - name: dev\n            project_id: project\n")

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
		t.Fatalf("expected error exit code, got %d", exitCode)
	}
	if strings.Contains(stderr.String(), adcGuidanceLine) {
		t.Fatalf("expected non-auth provider error to remain unchanged, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "create compute service: boom") {
		t.Fatalf("expected raw non-auth provider error, got %q", stderr.String())
	}
}

func TestRunInterconnectRuntimeAuthErrorShowsCustomADCGuidance(t *testing.T) {
	stubADCPreflight(t, func(context.Context) error { return nil })
	restoreProvider := newComputeProvider
	newComputeProvider = func(context.Context) (provider.DiscoveryProvider, error) {
		return errorDiscoveryProvider{
			interconnectErr: fmt.Errorf(
				"list dedicated interconnects for source project %q: %w",
				"src-project",
				&googleapi.Error{Code: 401, Message: "Request had invalid authentication credentials."},
			),
		}, nil
	}
	t.Cleanup(func() {
		newComputeProvider = restoreProvider
	})

	configPath := writeTempConfig(t, "org:\n  - name: dbc\n    workload:\n      - name: native\n        env:\n          - name: dev\n            project_id: project\n")

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
		t.Fatalf("expected error exit code, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), adcGuidanceLine) || !strings.Contains(stderr.String(), adcGuidanceCommand) {
		t.Fatalf("expected ADC guidance for runtime auth error, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Underlying error: list dedicated interconnects for source project") {
		t.Fatalf("expected wrapped runtime error, got %q", stderr.String())
	}
}

func TestRunVPNRuntimeAuthErrorShowsCustomADCGuidance(t *testing.T) {
	stubADCPreflight(t, func(context.Context) error { return nil })
	restoreProvider := newComputeProvider
	newComputeProvider = func(context.Context) (provider.DiscoveryProvider, error) {
		return errorDiscoveryProvider{
			vpnGatewayErr: fmt.Errorf(
				"list vpn gateways for source project %q: %w",
				"project",
				&googleapi.Error{Code: 401, Message: "Request had invalid authentication credentials."},
			),
		}, nil
	}
	t.Cleanup(func() {
		newComputeProvider = restoreProvider
	})

	configPath := writeTempConfig(t, "org:\n  - name: dbc\n    workload:\n      - name: native\n        env:\n          - name: dev\n            project_id: project\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{
		"-t", "vpn",
		"-o", "dbc",
		"-w", "native",
		"-e", "dev",
		"-c", configPath,
	}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected error exit code, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), adcGuidanceLine) || !strings.Contains(stderr.String(), adcGuidanceCommand) {
		t.Fatalf("expected ADC guidance for vpn runtime auth error, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Underlying error: list vpn gateways for source project") {
		t.Fatalf("expected wrapped vpn runtime error, got %q", stderr.String())
	}
}

func TestRunRuntimeNonAuthErrorRemainsUnchanged(t *testing.T) {
	stubADCPreflight(t, func(context.Context) error { return nil })
	restoreProvider := newComputeProvider
	newComputeProvider = func(context.Context) (provider.DiscoveryProvider, error) {
		return errorDiscoveryProvider{
			interconnectErr: errors.New("list dedicated interconnects for source project \"src-project\": boom"),
		}, nil
	}
	t.Cleanup(func() {
		newComputeProvider = restoreProvider
	})

	configPath := writeTempConfig(t, "org:\n  - name: dbc\n    workload:\n      - name: native\n        env:\n          - name: dev\n            project_id: project\n")

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
		t.Fatalf("expected error exit code, got %d", exitCode)
	}
	if strings.Contains(stderr.String(), adcGuidanceLine) {
		t.Fatalf("expected non-auth runtime error to remain unchanged, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "list dedicated interconnects for source project \"src-project\": boom") {
		t.Fatalf("expected raw non-auth runtime error, got %q", stderr.String())
	}
}

func TestRunADCPreflightFailsBeforeConfigReadForInterconnect(t *testing.T) {
	stubADCPreflight(t, func(context.Context) error {
		return errors.New("google: could not find default credentials")
	})
	restoreProvider := newComputeProvider
	newComputeProvider = func(context.Context) (provider.DiscoveryProvider, error) {
		t.Fatal("provider should not be created when ADC preflight fails")
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
		"-w", "native",
		"-e", "dev",
		"-p", "src-project",
		"-c", filepath.Join(t.TempDir(), "missing.yaml"),
	}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected error exit code, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), adcGuidanceLine) || !strings.Contains(stderr.String(), adcGuidanceCommand) {
		t.Fatalf("expected ADC guidance for early preflight failure, got %q", stderr.String())
	}
	if strings.Contains(stderr.String(), "read config") {
		t.Fatalf("expected ADC failure before config read, got %q", stderr.String())
	}
}

func TestRunADCPreflightFailsBeforeConfigParseForVPN(t *testing.T) {
	stubADCPreflight(t, func(context.Context) error {
		return errors.New("google: could not find default credentials")
	})
	restoreProvider := newComputeProvider
	newComputeProvider = func(context.Context) (provider.DiscoveryProvider, error) {
		t.Fatal("provider should not be created when ADC preflight fails")
		return nil, nil
	}
	t.Cleanup(func() {
		newComputeProvider = restoreProvider
	})

	configPath := writeTempConfig(t, "org:\n  - name:\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{
		"-t", "vpn",
		"-o", "dbc",
		"-w", "native",
		"-e", "dev",
		"-c", configPath,
	}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected error exit code, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), adcGuidanceLine) || !strings.Contains(stderr.String(), adcGuidanceCommand) {
		t.Fatalf("expected ADC guidance for early vpn preflight failure, got %q", stderr.String())
	}
	if strings.Contains(stderr.String(), "cannot be empty") {
		t.Fatalf("expected ADC failure before config parse, got %q", stderr.String())
	}
}

func TestRunHelpBypassesADCPreflight(t *testing.T) {
	stubADCPreflight(t, func(context.Context) error {
		t.Fatal("ADC preflight should not run for help")
		return nil
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{"-h"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected success exit code, got %d", exitCode)
	}
}

func TestRunVersionBypassesADCPreflight(t *testing.T) {
	stubADCPreflight(t, func(context.Context) error {
		t.Fatal("ADC preflight should not run for version")
		return nil
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{"version"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected success exit code, got %d", exitCode)
	}
}
