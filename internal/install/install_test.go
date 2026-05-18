package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureMKCertInstalledRunsTrustStoreBootstrap(t *testing.T) {
	t.Parallel()

	origLookPath := mkcertLookPath
	origInstallRun := mkcertInstallRun
	t.Cleanup(func() {
		mkcertLookPath = origLookPath
		mkcertInstallRun = origInstallRun
	})

	lookedUp := false
	installed := false
	mkcertLookPath = func(file string) (string, error) {
		lookedUp = true
		if file != "mkcert" {
			t.Fatalf("expected mkcert lookup, got %q", file)
		}
		return "/opt/homebrew/bin/mkcert", nil
	}
	mkcertInstallRun = func() error {
		installed = true
		return nil
	}

	if err := ensureMKCertInstalled(context.Background()); err != nil {
		t.Fatalf("ensure mkcert installed: %v", err)
	}
	if !lookedUp {
		t.Fatalf("expected mkcert lookup")
	}
	if !installed {
		t.Fatalf("expected mkcert -install bootstrap")
	}
}

func TestEnsureMKCertInstalledReturnsTrustStoreInstallError(t *testing.T) {
	t.Parallel()

	origLookPath := mkcertLookPath
	origInstallRun := mkcertInstallRun
	t.Cleanup(func() {
		mkcertLookPath = origLookPath
		mkcertInstallRun = origInstallRun
	})

	mkcertLookPath = func(string) (string, error) { return "/opt/homebrew/bin/mkcert", nil }
	mkcertInstallRun = func() error { return errors.New("permission denied") }

	err := ensureMKCertInstalled(context.Background())
	if err == nil {
		t.Fatalf("expected mkcert trust store install error")
	}
	if !strings.Contains(err.Error(), "mkcert trust store install failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallCreatesPathsResolverBootstrapsCertsAndStartsDaemon(t *testing.T) {
	t.Parallel()

	var calls []string
	installer := NewInstaller(Dependencies{
		CurrentEUID:    func() int { return 0 },
		ResolveGUIUser: func() (int, string, error) { return 501, "/Users/dev", nil },
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
		PrepareDaemonBinary: func(path string) error {
			calls = append(calls, "stage-binary")
			if got, want := path, daemonProgramPath; got != want {
				t.Fatalf("expected daemon program path %q, got %q", want, got)
			}
			return nil
		},
		PrepareMenubarBundle: func(string, InstallPaths) error { return nil },
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
	for _, want := range []string{"paths", "resolver", "certs", "stage-binary", "daemon-install", "daemon-start"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected call %q in flow, got %q", want, joined)
		}
	}
}

func TestInstallFailsWithActionableErrorWhenDaemonBinaryStagingFails(t *testing.T) {
	t.Parallel()

	daemonInstalled := false
	installer := NewInstaller(Dependencies{
		CurrentEUID:           func() int { return 0 },
		ResolveGUIUser:        func() (int, string, error) { return 501, "/Users/dev", nil },
		EnsurePaths:           func(InstallPaths) error { return nil },
		WriteResolver:         func(ResolverConfig) error { return nil },
		BootstrapCertificates: func(context.Context) error { return nil },
		PrepareDaemonBinary: func(string) error {
			return errors.New("source executable is not readable")
		},
		PrepareMenubarBundle: func(string, InstallPaths) error { return nil },
		InstallDaemonService: func(LaunchdServiceConfig) error {
			daemonInstalled = true
			return nil
		},
		StartDaemonService: func(LaunchdServiceConfig) error { return nil },
	})

	err := installer.Install(context.Background(), Options{Suffix: "test"})
	if err == nil {
		t.Fatalf("expected staging failure")
	}
	if !strings.Contains(err.Error(), "stage daemon executable") {
		t.Fatalf("expected explicit staging context in error, got %v", err)
	}
	if !strings.Contains(err.Error(), daemonProgramPath) {
		t.Fatalf("expected target daemon path in error, got %v", err)
	}
	if daemonInstalled {
		t.Fatalf("expected daemon service install to stop when binary staging fails")
	}
}

func TestInstallInstallsMenubarOnlyWhenOptedIn(t *testing.T) {
	t.Parallel()

	var menubarInstallCount int
	installer := NewInstaller(Dependencies{
		CurrentEUID:             func() int { return 0 },
		ResolveGUIUser:          func() (int, string, error) { return 501, "/Users/dev", nil },
		EnsurePaths:             func(InstallPaths) error { return nil },
		WriteResolver:           func(ResolverConfig) error { return nil },
		BootstrapCertificates:   func(context.Context) error { return nil },
		PrepareDaemonBinary:     func(string) error { return nil },
		PrepareMenubarBundle:    func(string, InstallPaths) error { return nil },
		InstallDaemonService:    func(LaunchdServiceConfig) error { return nil },
		StartDaemonService:      func(LaunchdServiceConfig) error { return nil },
		InstallMenubarService:   func(LaunchdServiceConfig) error { menubarInstallCount++; return nil },
		StartMenubarService:     func(LaunchdServiceConfig) error { return nil },
		ResolveGUIUserOwnership: func() (int, int, string, error) { return 501, 20, "/Users/dev", nil },
		EnsureMenubarOwnership:  func(InstallPaths, LaunchdServiceConfig, int, int) error { return nil },
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
	daemonCfg := DaemonServiceConfig(paths, "")
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

func TestDaemonServiceConfigUsesGUIUserMkcertHome(t *testing.T) {
	t.Parallel()

	cfg := DaemonServiceConfig(DefaultPaths(), "/Users/alice")
	if got, want := cfg.Env["HOME"], "/Users/alice"; got != want {
		t.Fatalf("expected HOME %q, got %q", want, got)
	}
	if got, want := cfg.Env["CAROOT"], "/Users/alice/Library/Application Support/mkcert"; got != want {
		t.Fatalf("expected CAROOT %q, got %q", want, got)
	}
}

func TestInstallWithMenubarTargetsResolvedGUIUser(t *testing.T) {
	t.Parallel()

	paths := DefaultPaths()
	var gotInstallCfg LaunchdServiceConfig
	var gotStartCfg LaunchdServiceConfig
	var gotOwnershipPaths InstallPaths
	var gotOwnershipCfg LaunchdServiceConfig
	var gotOwnershipUID int
	var gotOwnershipGID int

	installer := NewInstaller(Dependencies{
		CurrentEUID:             func() int { return 0 },
		ResolveGUIUser:          func() (int, string, error) { return 502, "/Users/alice", nil },
		EnsurePaths:             func(InstallPaths) error { return nil },
		WriteResolver:           func(ResolverConfig) error { return nil },
		BootstrapCertificates:   func(context.Context) error { return nil },
		PrepareDaemonBinary:     func(string) error { return nil },
		PrepareMenubarBundle:    func(string, InstallPaths) error { return nil },
		InstallDaemonService:    func(LaunchdServiceConfig) error { return nil },
		StartDaemonService:      func(LaunchdServiceConfig) error { return nil },
		ResolveGUIUserOwnership: func() (int, int, string, error) { return 502, 20, "/Users/alice", nil },
		InstallMenubarService:   func(cfg LaunchdServiceConfig) error { gotInstallCfg = cfg; return nil },
		StartMenubarService:     func(cfg LaunchdServiceConfig) error { gotStartCfg = cfg; return nil },
		EnsureMenubarOwnership: func(paths InstallPaths, cfg LaunchdServiceConfig, uid, gid int) error {
			gotOwnershipPaths = paths
			gotOwnershipCfg = cfg
			gotOwnershipUID = uid
			gotOwnershipGID = gid
			return nil
		},
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
	wantProgramPath := filepath.Join("/Users/alice", "Library", "Application Support", "DevProxy", "DevProxy Menubar.app", "Contents", "MacOS", "devproxy-menubar")
	if gotInstallCfg.Program != wantProgramPath || gotStartCfg.Program != wantProgramPath {
		t.Fatalf("expected menubar program path %q, got install=%q start=%q", wantProgramPath, gotInstallCfg.Program, gotStartCfg.Program)
	}
	if len(gotInstallCfg.Arguments) != 1 || gotInstallCfg.Arguments[0] != "menubar" {
		t.Fatalf("expected install arguments [menubar], got %v", gotInstallCfg.Arguments)
	}
	if len(gotStartCfg.Arguments) != 1 || gotStartCfg.Arguments[0] != "menubar" {
		t.Fatalf("expected start arguments [menubar], got %v", gotStartCfg.Arguments)
	}
	if gotOwnershipUID != 502 || gotOwnershipGID != 20 {
		t.Fatalf("expected ownership fix for uid/gid 502:20, got %d:%d", gotOwnershipUID, gotOwnershipGID)
	}
	if gotOwnershipCfg.PlistPath != wantPlistPath {
		t.Fatalf("expected ownership fix to target plist %q, got %q", wantPlistPath, gotOwnershipCfg.PlistPath)
	}
	if wantUserLibrary := filepath.Join("/Users/alice", "Library"); gotOwnershipPaths.UserLibraryDir != wantUserLibrary {
		t.Fatalf("expected ownership fix to target user library %q, got %q", wantUserLibrary, gotOwnershipPaths.UserLibraryDir)
	}
}

func TestInstallWithMenubarFailsExplicitlyWithoutGUIUser(t *testing.T) {
	t.Parallel()

	installer := NewInstaller(Dependencies{
		CurrentEUID:           func() int { return 0 },
		ResolveGUIUser:        func() (int, string, error) { return 501, "/Users/dev", nil },
		EnsurePaths:           func(InstallPaths) error { return nil },
		WriteResolver:         func(ResolverConfig) error { return nil },
		BootstrapCertificates: func(context.Context) error { return nil },
		PrepareDaemonBinary:   func(string) error { return nil },
		PrepareMenubarBundle:  func(string, InstallPaths) error { return nil },
		InstallDaemonService:  func(LaunchdServiceConfig) error { return nil },
		StartDaemonService:    func(LaunchdServiceConfig) error { return nil },
		ResolveGUIUserOwnership: func() (int, int, string, error) {
			return 0, 0, "", errors.New("no active GUI user session found; log into macOS desktop and re-run install --with-menubar as sudo")
		},
		EnsureMenubarOwnership: func(InstallPaths, LaunchdServiceConfig, int, int) error { return nil },
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
		ResolveGUIUser:        func() (int, string, error) { return 501, "/Users/dev", nil },
		EnsurePaths:           func(InstallPaths) error { return nil },
		WriteResolver:         func(ResolverConfig) error { return errors.New("permission denied") },
		BootstrapCertificates: func(context.Context) error { return nil },
		PrepareDaemonBinary:   func(string) error { return nil },
		PrepareMenubarBundle:  func(string, InstallPaths) error { return nil },
		InstallDaemonService:  func(LaunchdServiceConfig) error { return nil },
		StartDaemonService:    func(LaunchdServiceConfig) error { return nil },
	})

	err := installer.Install(context.Background(), Options{Suffix: "test"})
	if err == nil || !strings.Contains(err.Error(), "resolver") {
		t.Fatalf("expected explicit resolver failure, got %v", err)
	}
}

func TestInstallWrapsDaemonStartupVerificationErrors(t *testing.T) {
	t.Parallel()

	installer := NewInstaller(Dependencies{
		CurrentEUID:           func() int { return 0 },
		ResolveGUIUser:        func() (int, string, error) { return 501, "/Users/dev", nil },
		EnsurePaths:           func(InstallPaths) error { return nil },
		WriteResolver:         func(ResolverConfig) error { return nil },
		BootstrapCertificates: func(context.Context) error { return nil },
		PrepareDaemonBinary:   func(string) error { return nil },
		PrepareMenubarBundle:  func(string, InstallPaths) error { return nil },
		InstallDaemonService:  func(LaunchdServiceConfig) error { return nil },
		StartDaemonService: func(LaunchdServiceConfig) error {
			return errors.New("launchd service system/com.devproxy.daemon failed to reach running state within 2s")
		},
	})

	err := installer.Install(context.Background(), Options{Suffix: "test"})
	if err == nil {
		t.Fatalf("expected daemon startup verification failure")
	}
	if !strings.Contains(err.Error(), "start daemon service") {
		t.Fatalf("expected daemon startup context, got %v", err)
	}
	if !strings.Contains(err.Error(), "failed to reach running state") {
		t.Fatalf("expected running-state verification detail, got %v", err)
	}
}

func TestInstallWrapsMenubarStartupVerificationErrors(t *testing.T) {
	t.Parallel()

	installer := NewInstaller(Dependencies{
		CurrentEUID:             func() int { return 0 },
		ResolveGUIUser:          func() (int, string, error) { return 501, "/Users/dev", nil },
		EnsurePaths:             func(InstallPaths) error { return nil },
		WriteResolver:           func(ResolverConfig) error { return nil },
		BootstrapCertificates:   func(context.Context) error { return nil },
		PrepareDaemonBinary:     func(string) error { return nil },
		PrepareMenubarBundle:    func(string, InstallPaths) error { return nil },
		InstallDaemonService:    func(LaunchdServiceConfig) error { return nil },
		StartDaemonService:      func(LaunchdServiceConfig) error { return nil },
		ResolveGUIUserOwnership: func() (int, int, string, error) { return 501, 20, "/Users/dev", nil },
		InstallMenubarService:   func(LaunchdServiceConfig) error { return nil },
		EnsureMenubarOwnership:  func(InstallPaths, LaunchdServiceConfig, int, int) error { return nil },
		StartMenubarService: func(LaunchdServiceConfig) error {
			return errors.New("launchd service gui/501/com.devproxy.menubar failed to reach running state within 2s")
		},
	})

	err := installer.Install(context.Background(), Options{Suffix: "test", WithMenubar: true})
	if err == nil {
		t.Fatalf("expected menubar startup verification failure")
	}
	if !strings.Contains(err.Error(), "start menubar service") {
		t.Fatalf("expected menubar startup context, got %v", err)
	}
	if !strings.Contains(err.Error(), "failed to reach running state") {
		t.Fatalf("expected running-state verification detail, got %v", err)
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
		PrepareMenubarBundle: func(string, InstallPaths) error { return nil },
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

func TestPrepareMenubarBundleCreatesAppBundle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	sourceBinary := filepath.Join(root, "devproxy")
	if err := os.WriteFile(sourceBinary, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write source binary: %v", err)
	}
	paths := InstallPaths{UserLibraryDir: filepath.Join(root, "Library")}

	if err := prepareMenubarBundle(sourceBinary, paths); err != nil {
		t.Fatalf("prepare menubar bundle: %v", err)
	}

	if _, err := os.Stat(MenubarBundleExecutablePath(paths)); err != nil {
		t.Fatalf("expected bundle executable, got %v", err)
	}
	infoPlistPath := filepath.Join(MenubarBundlePath(paths), "Contents", "Info.plist")
	data, err := os.ReadFile(infoPlistPath)
	if err != nil {
		t.Fatalf("read Info.plist: %v", err)
	}
	plist := string(data)
	if !strings.Contains(plist, "<key>LSUIElement</key>") {
		t.Fatalf("expected LSUIElement in bundle Info.plist")
	}
	if !strings.Contains(plist, "<key>LSUIElement</key>\n\t<true/>") {
		t.Fatalf("expected LSUIElement to be written as a boolean true, got %q", plist)
	}
	if !strings.Contains(plist, "devproxy-menubar") {
		t.Fatalf("expected bundle executable name in Info.plist")
	}
}

func TestInstallFailsWhenMenubarOwnershipFixFails(t *testing.T) {
	t.Parallel()

	installer := NewInstaller(Dependencies{
		CurrentEUID:             func() int { return 0 },
		EnsurePaths:             func(InstallPaths) error { return nil },
		WriteResolver:           func(ResolverConfig) error { return nil },
		BootstrapCertificates:   func(context.Context) error { return nil },
		PrepareDaemonBinary:     func(string) error { return nil },
		PrepareMenubarBundle:    func(string, InstallPaths) error { return nil },
		InstallDaemonService:    func(LaunchdServiceConfig) error { return nil },
		StartDaemonService:      func(LaunchdServiceConfig) error { return nil },
		ResolveGUIUserOwnership: func() (int, int, string, error) { return 501, 20, "/Users/dev", nil },
		InstallMenubarService:   func(LaunchdServiceConfig) error { return nil },
		EnsureMenubarOwnership:  func(InstallPaths, LaunchdServiceConfig, int, int) error { return errors.New("permission denied") },
	})

	err := installer.Install(context.Background(), Options{Suffix: "test", WithMenubar: true})
	if err == nil {
		t.Fatalf("expected ownership fix failure")
	}
	if !strings.Contains(err.Error(), "fix menubar file ownership") {
		t.Fatalf("expected explicit ownership failure context, got %v", err)
	}
}
