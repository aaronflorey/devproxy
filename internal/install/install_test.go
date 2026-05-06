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
		CurrentEUID: func() int { return 0 },
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
		CurrentEUID:           func() int { return 0 },
		EnsurePaths:           func(InstallPaths) error { return nil },
		WriteResolver:         func(ResolverConfig) error { return nil },
		BootstrapCertificates: func(context.Context) error { return nil },
		InstallDaemonService:  func(LaunchdServiceConfig) error { return nil },
		StartDaemonService:    func(LaunchdServiceConfig) error { return nil },
		InstallMenubarService: func(LaunchdServiceConfig) error { menubarInstallCount++; return nil },
		StartMenubarService:   func(LaunchdServiceConfig) error { return nil },
		ResolveGUIUser:        func() (int, string, error) { return 501, "/Users/dev", nil },
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

	agentCfg := MenubarServiceConfig(paths, 501)
	if agentCfg.Domain != DomainAgent {
		t.Fatalf("expected menubar domain %q, got %q", DomainAgent, agentCfg.Domain)
	}
	if agentCfg.AgentUID != 501 {
		t.Fatalf("expected menubar uid %d, got %d", 501, agentCfg.AgentUID)
	}
	if agentCfg.Label == "" || !strings.Contains(agentCfg.Label, "menubar") {
		t.Fatalf("expected menubar label to include menubar role, got %q", agentCfg.Label)
	}
	if !strings.Contains(agentCfg.PlistPath, filepath.Join(paths.UserLibraryDir, "LaunchAgents")) {
		t.Fatalf("expected menubar plist under user LaunchAgents, got %q", agentCfg.PlistPath)
	}
}

func TestInstallWithMenubarTargetsResolvedGUIUser(t *testing.T) {
	t.Parallel()

	paths := DefaultPaths()
	var gotInstallCfg LaunchdServiceConfig
	var gotStartCfg LaunchdServiceConfig

	installer := NewInstaller(Dependencies{
		CurrentEUID:           func() int { return 0 },
		EnsurePaths:           func(InstallPaths) error { return nil },
		WriteResolver:         func(ResolverConfig) error { return nil },
		BootstrapCertificates: func(context.Context) error { return nil },
		InstallDaemonService:  func(LaunchdServiceConfig) error { return nil },
		StartDaemonService:    func(LaunchdServiceConfig) error { return nil },
		ResolveGUIUser:        func() (int, string, error) { return 502, "/Users/alice", nil },
		InstallMenubarService: func(cfg LaunchdServiceConfig) error { gotInstallCfg = cfg; return nil },
		StartMenubarService:   func(cfg LaunchdServiceConfig) error { gotStartCfg = cfg; return nil },
	})

	err := installer.Install(context.Background(), Options{Suffix: "test", WithMenubar: true, Paths: paths})
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	if gotInstallCfg.AgentUID != 502 || gotStartCfg.AgentUID != 502 {
		t.Fatalf("expected menubar agent uid 502 for install/start, got install=%d start=%d", gotInstallCfg.AgentUID, gotStartCfg.AgentUID)
	}
	wantPlistPath := filepath.Join("/Users/alice", "Library", "LaunchAgents", "com.devproxy.menubar.plist")
	if gotInstallCfg.PlistPath != wantPlistPath || gotStartCfg.PlistPath != wantPlistPath {
		t.Fatalf("expected menubar plist path %q, got install=%q start=%q", wantPlistPath, gotInstallCfg.PlistPath, gotStartCfg.PlistPath)
	}
}

func TestInstallWithMenubarFailsExplicitlyWithoutGUIUser(t *testing.T) {
	t.Parallel()

	installer := NewInstaller(Dependencies{
		CurrentEUID:           func() int { return 0 },
		EnsurePaths:           func(InstallPaths) error { return nil },
		WriteResolver:         func(ResolverConfig) error { return nil },
		BootstrapCertificates: func(context.Context) error { return nil },
		InstallDaemonService:  func(LaunchdServiceConfig) error { return nil },
		StartDaemonService:    func(LaunchdServiceConfig) error { return nil },
		ResolveGUIUser: func() (int, string, error) {
			return 0, "", errors.New("no active GUI user session found; log into macOS desktop and re-run install --with-menubar as sudo")
		},
	})

	err := installer.Install(context.Background(), Options{Suffix: "test", WithMenubar: true})
	if err == nil {
		t.Fatalf("expected install error when GUI user cannot be resolved")
	}
	if !strings.Contains(err.Error(), "resolve GUI user for menubar service") {
		t.Fatalf("expected explicit menubar GUI resolution context, got %v", err)
	}
	if !strings.Contains(err.Error(), "no active GUI user session found") {
		t.Fatalf("expected actionable no-GUI-user message, got %v", err)
	}
}

func TestInstallWrapsResolverErrors(t *testing.T) {
	t.Parallel()

	installer := NewInstaller(Dependencies{
		CurrentEUID:           func() int { return 0 },
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

func TestInstallRequiresRootBeforeMutations(t *testing.T) {
	t.Parallel()

	mutated := false
	installer := NewInstaller(Dependencies{
		CurrentEUID: func() int { return 501 },
		EnsurePaths: func(InstallPaths) error {
			mutated = true
			return nil
		},
	})

	err := installer.Install(context.Background(), Options{Suffix: "test"})
	if err == nil {
		t.Fatalf("expected root preflight error")
	}
	if !strings.Contains(err.Error(), "devproxy install requires root privileges; rerun with sudo") {
		t.Fatalf("unexpected error: %v", err)
	}
	if mutated {
		t.Fatalf("expected no install mutations before root preflight")
	}
}
