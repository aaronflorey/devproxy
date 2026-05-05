package install

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCreatesPathsResolverBootstrapsCertsAndStartsDaemon(t *testing.T) {
	t.Parallel()

	var calls []string
	installer := NewInstaller(Dependencies{
		EnsurePaths: func(InstallPaths) error {
			calls = append(calls, "paths")
			return nil
		},
		WriteResolver: func(ResolverConfig) error {
			calls = append(calls, "resolver")
			return nil
		},
		BootstrapCertificates: func(context.Context) error {
			calls = append(calls, "certs")
			return nil
		},
		InstallDaemonService: func(serviceConfig LaunchdServiceConfig) error {
			calls = append(calls, "daemon-install")
			if got, want := serviceConfig.Domain, DomainSystem; got != want {
				t.Fatalf("expected daemon domain %q, got %q", want, got)
			}
			if got, want := serviceConfig.Label, "com.devproxy.daemon"; got != want {
				t.Fatalf("expected daemon label %q, got %q", want, got)
			}
			return nil
		},
		StartDaemonService: func(serviceConfig LaunchdServiceConfig) error {
			calls = append(calls, "daemon-start")
			if got, want := serviceConfig.Domain, DomainSystem; got != want {
				t.Fatalf("expected daemon start in %q, got %q", want, got)
			}
			return nil
		},
	})

	err := installer.Install(context.Background(), Options{Suffix: "test"})
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	joined := strings.Join(calls, ",")
	for _, want := range []string{"paths", "resolver", "certs", "daemon-install", "daemon-start"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected call %q in flow, got %q", want, joined)
		}
	}
}

func TestInstallInstallsMenubarOnlyWhenOptedIn(t *testing.T) {
	t.Parallel()

	var menubarInstallCount int
	installer := NewInstaller(Dependencies{
		EnsurePaths:            func(InstallPaths) error { return nil },
		WriteResolver:          func(ResolverConfig) error { return nil },
		BootstrapCertificates:  func(context.Context) error { return nil },
		InstallDaemonService:   func(LaunchdServiceConfig) error { return nil },
		StartDaemonService:     func(LaunchdServiceConfig) error { return nil },
		InstallMenubarService:  func(LaunchdServiceConfig) error { menubarInstallCount++; return nil },
		StartMenubarService:    func(LaunchdServiceConfig) error { return nil },
	})

	if err := installer.Install(context.Background(), Options{Suffix: "test", WithMenubar: false}); err != nil {
		t.Fatalf("default install failed: %v", err)
	}
	if menubarInstallCount != 0 {
		t.Fatalf("expected no menubar install for default flow, got %d", menubarInstallCount)
	}

	if err := installer.Install(context.Background(), Options{Suffix: "test", WithMenubar: true}); err != nil {
		t.Fatalf("opt-in menubar install failed: %v", err)
	}
	if menubarInstallCount != 1 {
		t.Fatalf("expected one menubar install when opted in, got %d", menubarInstallCount)
	}
}

func TestLaunchdRolesUseSeparateDomainsAndLabels(t *testing.T) {
	t.Parallel()

	paths := DefaultPaths()
	daemonCfg := DaemonServiceConfig(paths)
	if daemonCfg.Domain != DomainSystem {
		t.Fatalf("expected daemon domain %q, got %q", DomainSystem, daemonCfg.Domain)
	}
	if daemonCfg.Label == "" || !strings.Contains(daemonCfg.Label, "daemon") {
		t.Fatalf("expected daemon label to include daemon role, got %q", daemonCfg.Label)
	}
	if !strings.Contains(daemonCfg.PlistPath, "/Library/LaunchDaemons/") {
		t.Fatalf("expected daemon plist under LaunchDaemons, got %q", daemonCfg.PlistPath)
	}

	agentCfg := MenubarServiceConfig(paths)
	if agentCfg.Domain != DomainAgent {
		t.Fatalf("expected menubar domain %q, got %q", DomainAgent, agentCfg.Domain)
	}
	if agentCfg.Label == "" || !strings.Contains(agentCfg.Label, "menubar") {
		t.Fatalf("expected menubar label to include menubar role, got %q", agentCfg.Label)
	}
	if !strings.Contains(agentCfg.PlistPath, filepath.Join(paths.UserLibraryDir, "LaunchAgents")) {
		t.Fatalf("expected menubar plist under user LaunchAgents, got %q", agentCfg.PlistPath)
	}
}

func TestInstallWrapsResolverErrors(t *testing.T) {
	t.Parallel()

	installer := NewInstaller(Dependencies{
		EnsurePaths:           func(InstallPaths) error { return nil },
		WriteResolver:         func(ResolverConfig) error { return errors.New("permission denied") },
		BootstrapCertificates: func(context.Context) error { return nil },
		InstallDaemonService:  func(LaunchdServiceConfig) error { return nil },
		StartDaemonService:    func(LaunchdServiceConfig) error { return nil },
	})

	err := installer.Install(context.Background(), Options{Suffix: "test"})
	if err == nil || !strings.Contains(err.Error(), "resolver") {
		t.Fatalf("expected explicit resolver failure, got %v", err)
	}
}
