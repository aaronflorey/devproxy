package devproxy

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mochaka/devproxy/internal/admin"
	"github.com/mochaka/devproxy/internal/adminapi"
	"github.com/mochaka/devproxy/internal/config"
	"github.com/mochaka/devproxy/internal/doctor"
	"github.com/mochaka/devproxy/internal/routing"
)

func TestDoctorCommandUsesSharedDoctorProjection(t *testing.T) {
	originalChecker := buildDoctorChecker
	originalCfg := loadedCfg
	defer func() {
		buildDoctorChecker = originalChecker
		loadedCfg = originalCfg
	}()

	buildDoctorChecker = func(string) *doctor.Checker {
		return doctor.NewChecker(doctor.Dependencies{
			CheckDocker:      func(context.Context) error { return nil },
			CheckLaunchd:     func(context.Context) error { return nil },
			CheckAdminSocket: func(context.Context) error { return nil },
			CheckProxyHTTP:   func(context.Context, string) error { return nil },
			CheckProxyHTTPS:  func(context.Context, string) error { return nil },
			CheckMKCert:      func(context.Context) error { return nil },
			CheckLocalCA:     func(context.Context) error { return nil },
			ReadResolverState: func(context.Context) (doctor.ResolverState, error) {
				return doctor.ResolverState{ManagedSuffix: "test", ActiveResolver: true, Nameservers: []string{"127.0.0.1"}, Evidence: "resolver active"}, nil
			},
			ResolveExampleHost: func(context.Context, string) (string, error) { return "127.0.0.1", nil },
			ReadNetworkHealth: func(context.Context) (admin.NetworkRuntimeHealth, error) {
				return admin.NetworkRuntimeHealth{HTTP: admin.ListenerStatus{Bound: true, BindAddress: "127.0.0.1:80"}, HTTPS: admin.ListenerStatus{Bound: true, BindAddress: "127.0.0.1:443"}, ManagedSuffix: "test"}, nil
			},
		})
	}
	loadedCfg = config.DefaultConfig()

	state := adminapi.StateSnapshot{
		Status: admin.StatusView{DNS: admin.DNSStatus{Healthy: true, ManagedSuffix: "test"}, HTTP: admin.ListenerStatus{Bound: true}, HTTPS: admin.ListenerStatus{Bound: true}, CertificateReady: true},
		Doctor: admin.DoctorView{
			ConflictCount: 1,
			WarningCount:  1,
			Warnings:      []routing.Warning{{Code: "invalid_label", Message: "ignored invalid label", Container: "acme-api-2", Field: "domain"}},
			Conflicts:     []routing.Conflict{{Hostname: "api.acme.test", Winner: routing.Candidate{ContainerName: "acme-api-1"}, Losers: []routing.Candidate{{ContainerName: "acme-api-2"}}, Reason: "higher priority winner kept route"}},
		},
	}

	socketPath, shutdown := startCommandTestServer(t, state)
	defer shutdown()

	out := &bytes.Buffer{}
	cmd := newDoctorCommand()
	cmd.SetOut(out)
	cmd.SetArgs([]string{"--admin-socket", socketPath, "--example-host", "api.acme.test"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("doctor command failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "docker\tok\tok") {
		t.Fatalf("expected runtime checks in doctor output, got %q", output)
	}
	if !strings.Contains(output, "snapshot_conflicts:") || !strings.Contains(output, "acme-api-2") {
		t.Fatalf("expected shared doctor conflict detail in output, got %q", output)
	}
	if !strings.Contains(output, "snapshot_warnings:") || !strings.Contains(output, "ignored invalid label") {
		t.Fatalf("expected shared doctor warning detail in output, got %q", output)
	}
}
