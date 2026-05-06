package install

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestUninstallHonorsIndependentCleanupChoices(t *testing.T) {
	t.Parallel()

	var removed []string
	u := NewUninstaller(UninstallDependencies{
		CurrentEUID:            func() int { return 0 },
		StopDaemonService:      func(context.Context, LaunchdServiceConfig) error { return nil },
		UninstallDaemonService: func(context.Context, LaunchdServiceConfig) error { return nil },
		RemoveResolver:         func(context.Context, ResolverConfig) error { return nil },
		RemoveConfig:           func(context.Context, InstallPaths) error { removed = append(removed, "config"); return nil },
		RemoveState:            func(context.Context, InstallPaths) error { removed = append(removed, "state"); return nil },
		RemoveLogs:             func(context.Context, InstallPaths) error { removed = append(removed, "logs"); return nil },
		RemoveCertificates:     func(context.Context, InstallPaths) error { removed = append(removed, "certs"); return nil },
	})

	err := u.Uninstall(context.Background(), UninstallOptions{
		Suffix: "test",
		Cleanup: CleanupScope{
			Config:       false,
			State:        true,
			Logs:         false,
			Certificates: true,
		},
	})
	if err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}

	got := strings.Join(removed, ",")
	if got != "state,certs" {
		t.Fatalf("expected only selected categories removed, got %q", got)
	}
}

func TestUninstallAlwaysStopsAndUnregistersBeforeOptionalCleanup(t *testing.T) {
	t.Parallel()

	var steps []string
	u := NewUninstaller(UninstallDependencies{
		CurrentEUID:       func() int { return 0 },
		StopDaemonService: func(context.Context, LaunchdServiceConfig) error { steps = append(steps, "stop-daemon"); return nil },
		UninstallDaemonService: func(context.Context, LaunchdServiceConfig) error {
			steps = append(steps, "uninstall-daemon")
			return nil
		},
		StopMenubarService: func(context.Context, LaunchdServiceConfig) error { steps = append(steps, "stop-menubar"); return nil },
		UninstallMenubarService: func(context.Context, LaunchdServiceConfig) error {
			steps = append(steps, "uninstall-menubar")
			return nil
		},
		RemoveResolver:     func(context.Context, ResolverConfig) error { steps = append(steps, "remove-resolver"); return nil },
		RemoveConfig:       func(context.Context, InstallPaths) error { steps = append(steps, "remove-config"); return nil },
		RemoveState:        func(context.Context, InstallPaths) error { steps = append(steps, "remove-state"); return nil },
		RemoveLogs:         func(context.Context, InstallPaths) error { steps = append(steps, "remove-logs"); return nil },
		RemoveCertificates: func(context.Context, InstallPaths) error { steps = append(steps, "remove-certs"); return nil },
	})

	err := u.Uninstall(context.Background(), UninstallOptions{
		Suffix:      "test",
		WithMenubar: true,
		Cleanup:     CleanupScope{Config: true, State: true, Logs: true, Certificates: true},
	})
	if err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}

	joined := strings.Join(steps, ",")
	if !strings.HasPrefix(joined, "stop-daemon,uninstall-daemon") {
		t.Fatalf("expected daemon stop/uninstall first, got %q", joined)
	}
	resolverIndex := strings.Index(joined, "remove-resolver")
	cleanupIndex := strings.Index(joined, "remove-config")
	if resolverIndex == -1 || cleanupIndex == -1 || resolverIndex > cleanupIndex {
		t.Fatalf("expected resolver removal before optional cleanup, got %q", joined)
	}
}

func TestUninstallRequiresRootBeforeTeardown(t *testing.T) {
	t.Parallel()

	started := false
	u := NewUninstaller(UninstallDependencies{
		CurrentEUID: func() int { return 501 },
		StopDaemonService: func(context.Context, LaunchdServiceConfig) error {
			started = true
			return nil
		},
	})

	err := u.Uninstall(context.Background(), UninstallOptions{Suffix: "test"})
	if err == nil {
		t.Fatalf("expected root preflight error")
	}
	if !strings.Contains(err.Error(), "devproxy uninstall requires root privileges; rerun with sudo") {
		t.Fatalf("unexpected error: %v", err)
	}
	if started {
		t.Fatalf("expected no teardown before root preflight")
	}
}

func TestUninstallTreatsMissingServiceStateAsNonFatal(t *testing.T) {
	t.Parallel()

	var removedResolver bool
	u := NewUninstaller(UninstallDependencies{
		CurrentEUID: func() int { return 0 },
		StopDaemonService: func(context.Context, LaunchdServiceConfig) error {
			return errors.New("launchctl bootout failed: Could not find service")
		},
		UninstallDaemonService:  func(context.Context, LaunchdServiceConfig) error { return errors.New("no such file") },
		StopMenubarService:      func(context.Context, LaunchdServiceConfig) error { return errors.New("service already unloaded") },
		UninstallMenubarService: func(context.Context, LaunchdServiceConfig) error { return errors.New("No such process") },
		RemoveResolver: func(context.Context, ResolverConfig) error {
			removedResolver = true
			return nil
		},
	})

	err := u.Uninstall(context.Background(), UninstallOptions{Suffix: "test", WithMenubar: true})
	if err != nil {
		t.Fatalf("expected missing service state to be non-fatal, got %v", err)
	}
	if !removedResolver {
		t.Fatalf("expected uninstall to continue to resolver removal")
	}
}
