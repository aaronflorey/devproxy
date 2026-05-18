package install

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestStartServiceFailsPreflightWhenProgramNotExecutable(t *testing.T) {
	tmp := t.TempDir()
	plistPath := filepath.Join(tmp, "com.devproxy.daemon.plist")
	if err := os.WriteFile(plistPath, []byte(plistFor(LaunchdServiceConfig{Label: "com.devproxy.daemon", Program: "/tmp/devproxy", Arguments: []string{"daemon"}})), 0o644); err != nil {
		t.Fatalf("write plist: %v", err)
	}
	programPath := filepath.Join(tmp, "devproxy")
	if err := os.WriteFile(programPath, []byte("#!/bin/sh\nexit 0\n"), 0o644); err != nil {
		t.Fatalf("write program: %v", err)
	}

	binDir := t.TempDir()
	makeFakePlutil(t, binDir)
	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+":"+originalPath)

	err := StartService(LaunchdServiceConfig{
		Label:     "com.devproxy.daemon",
		Domain:    DomainSystem,
		PlistPath: plistPath,
		Program:   programPath,
	})
	if err == nil {
		t.Fatalf("expected preflight executable failure")
	}
	if !strings.Contains(err.Error(), "not executable") {
		t.Fatalf("expected actionable executable error, got %v", err)
	}
}

func TestStartServiceIncludesBootstrapDiagnostics(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("launchctl script test is unix-only")
	}

	tmp := t.TempDir()
	stateFile := filepath.Join(tmp, "state")
	plistPath := filepath.Join(tmp, "com.devproxy.daemon.plist")
	programPath := filepath.Join(tmp, "devproxy")
	if err := os.WriteFile(plistPath, []byte(plistFor(LaunchdServiceConfig{Label: "com.devproxy.daemon", Program: programPath, Arguments: []string{"daemon"}})), 0o644); err != nil {
		t.Fatalf("write plist: %v", err)
	}
	if err := os.WriteFile(programPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write program: %v", err)
	}

	binDir := t.TempDir()
	makeFakePlutil(t, binDir)
	makeFailingBootstrapLaunchctl(t, binDir, stateFile)
	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+":"+originalPath)

	err := StartService(LaunchdServiceConfig{
		Label:     "com.devproxy.daemon",
		Domain:    DomainSystem,
		PlistPath: plistPath,
		Program:   programPath,
	})
	if err == nil {
		t.Fatalf("expected bootstrap failure")
	}
	msg := err.Error()
	if !strings.Contains(msg, "Likely causes") {
		t.Fatalf("expected likely causes hints, got %v", err)
	}
	if !strings.Contains(msg, "launchctl print diagnostics") {
		t.Fatalf("expected print diagnostics, got %v", err)
	}
	if !strings.Contains(msg, "state = waiting") {
		t.Fatalf("expected launchctl print output in diagnostics, got %v", err)
	}
}

func TestStartServiceBootoutIsIdempotentBeforeBootstrap(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("launchctl script test is unix-only")
	}

	tmp := t.TempDir()
	stateFile := filepath.Join(tmp, "state")
	plistPath := filepath.Join(tmp, "com.devproxy.daemon.plist")
	programPath := filepath.Join(tmp, "devproxy")
	if err := os.WriteFile(plistPath, []byte(plistFor(LaunchdServiceConfig{Label: "com.devproxy.daemon", Program: programPath, Arguments: []string{"daemon"}})), 0o644); err != nil {
		t.Fatalf("write plist: %v", err)
	}
	if err := os.WriteFile(programPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write program: %v", err)
	}

	binDir := t.TempDir()
	makeFakePlutil(t, binDir)
	makeIdempotentLaunchctl(t, binDir, stateFile)
	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+":"+originalPath)

	err := StartService(LaunchdServiceConfig{
		Label:     "com.devproxy.daemon",
		Domain:    DomainSystem,
		PlistPath: plistPath,
		Program:   programPath,
	})
	if err != nil {
		t.Fatalf("expected start success, got %v", err)
	}

	data, readErr := os.ReadFile(stateFile)
	if readErr != nil {
		t.Fatalf("read fake launchctl state: %v", readErr)
	}
	calls := string(data)
	if !strings.Contains(calls, "bootout system "+plistPath) {
		t.Fatalf("expected bootout call, got %q", calls)
	}
	if !strings.Contains(calls, "print system/com.devproxy.daemon") {
		t.Fatalf("expected missing-state print probe, got %q", calls)
	}
	if !strings.Contains(calls, "bootstrap system "+plistPath) {
		t.Fatalf("expected bootstrap call, got %q", calls)
	}
	if !strings.Contains(calls, "kickstart -k system/com.devproxy.daemon") {
		t.Fatalf("expected kickstart call, got %q", calls)
	}
	if !strings.Contains(calls, "print system/com.devproxy.daemon") {
		t.Fatalf("expected running-state print probe, got %q", calls)
	}
}

func TestStartServiceFailsWhenServiceDoesNotReachRunningState(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("launchctl script test is unix-only")
	}

	tmp := t.TempDir()
	stateFile := filepath.Join(tmp, "state")
	plistPath := filepath.Join(tmp, "com.devproxy.daemon.plist")
	programPath := filepath.Join(tmp, "devproxy")
	if err := os.WriteFile(plistPath, []byte(plistFor(LaunchdServiceConfig{Label: "com.devproxy.daemon", Program: programPath, Arguments: []string{"daemon"}, StderrLog: "/var/log/devproxy/daemon.stderr.log"})), 0o644); err != nil {
		t.Fatalf("write plist: %v", err)
	}
	if err := os.WriteFile(programPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write program: %v", err)
	}

	binDir := t.TempDir()
	makeFakePlutil(t, binDir)
	makeNonRunningLaunchctl(t, binDir, stateFile)
	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+":"+originalPath)

	err := StartService(LaunchdServiceConfig{
		Label:     "com.devproxy.daemon",
		Domain:    DomainSystem,
		PlistPath: plistPath,
		Program:   programPath,
		StderrLog: "/var/log/devproxy/daemon.stderr.log",
	})
	if err == nil {
		t.Fatalf("expected running-state failure")
	}
	if !strings.Contains(err.Error(), "failed to reach running state") {
		t.Fatalf("expected running-state failure context, got %v", err)
	}
	if !strings.Contains(err.Error(), "inspect stderr log: /var/log/devproxy/daemon.stderr.log") {
		t.Fatalf("expected stderr log hint, got %v", err)
	}
}

func TestStopServiceTreatsBootoutExitFiveAsMissingOnlyWithMissingStateProbe(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("launchctl script test is unix-only")
	}

	stateFile := filepath.Join(t.TempDir(), "state")
	fakeLaunchctlDir := makeFakeLaunchctl(t, stateFile, true)
	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", fakeLaunchctlDir+":"+originalPath)

	err := StopService(context.Background(), LaunchdServiceConfig{
		Label:     "com.devproxy.daemon",
		Domain:    DomainSystem,
		PlistPath: "/Library/LaunchDaemons/com.devproxy.daemon.plist",
	})
	if err != nil {
		t.Fatalf("expected missing-state bootout to be treated as success, got %v", err)
	}

	data, readErr := os.ReadFile(stateFile)
	if readErr != nil {
		t.Fatalf("read fake launchctl state: %v", readErr)
	}
	calls := string(data)
	if !strings.Contains(calls, "bootout system /Library/LaunchDaemons/com.devproxy.daemon.plist") {
		t.Fatalf("expected bootout call, got %q", calls)
	}
	if !strings.Contains(calls, "print system/com.devproxy.daemon") {
		t.Fatalf("expected fallback print probe, got %q", calls)
	}
}

func TestStopServicePreservesNonMissingBootoutFailures(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("launchctl script test is unix-only")
	}

	stateFile := filepath.Join(t.TempDir(), "state")
	fakeLaunchctlDir := makeFakeLaunchctl(t, stateFile, false)
	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", fakeLaunchctlDir+":"+originalPath)

	err := StopService(context.Background(), LaunchdServiceConfig{
		Label:     "com.devproxy.daemon",
		Domain:    DomainSystem,
		PlistPath: "/Library/LaunchDaemons/com.devproxy.daemon.plist",
	})
	if err == nil {
		t.Fatalf("expected non-missing bootout failure to be returned")
	}
	if !strings.Contains(err.Error(), "Input/output error") {
		t.Fatalf("expected original launchctl error, got %v", err)
	}

	data, readErr := os.ReadFile(stateFile)
	if readErr != nil {
		t.Fatalf("read fake launchctl state: %v", readErr)
	}
	if !strings.Contains(string(data), "print system/com.devproxy.daemon") {
		t.Fatalf("expected print probe before preserving non-missing failure")
	}
}

func TestDomainTargetUsesAgentUIDWhenProvided(t *testing.T) {
	t.Parallel()

	cfg := LaunchdServiceConfig{Domain: DomainAgent, AgentUID: 502}
	if got, want := domainTarget(cfg), "gui/502"; got != want {
		t.Fatalf("expected domain target %q, got %q", want, got)
	}
}

func TestPlistIncludesEnvironmentPath(t *testing.T) {
	t.Parallel()

	plist := plistFor(LaunchdServiceConfig{
		Label:     "com.devproxy.daemon",
		Program:   "/usr/local/bin/devproxy",
		Arguments: []string{"daemon"},
		StdoutLog: "/var/log/devproxy/daemon.stdout.log",
		StderrLog: "/var/log/devproxy/daemon.stderr.log",
	})

	if !strings.Contains(plist, "<key>EnvironmentVariables</key>") {
		t.Fatalf("expected launchd plist to include environment variables")
	}
	if !strings.Contains(plist, "<key>PATH</key>") {
		t.Fatalf("expected launchd plist to include PATH environment variable")
	}
	if !strings.Contains(plist, launchdDefaultPath) {
		t.Fatalf("expected launchd plist to include default PATH %q", launchdDefaultPath)
	}
	if !strings.Contains(plist, "<key>StandardOutPath</key>") || !strings.Contains(plist, "/var/log/devproxy/daemon.stdout.log") {
		t.Fatalf("expected launchd plist to include stdout log path")
	}
	if !strings.Contains(plist, "<key>StandardErrorPath</key>") || !strings.Contains(plist, "/var/log/devproxy/daemon.stderr.log") {
		t.Fatalf("expected launchd plist to include stderr log path")
	}
}

func TestPlistIncludesDaemonMKCertEnvironmentWhenProvided(t *testing.T) {
	t.Parallel()

	plist := plistFor(LaunchdServiceConfig{
		Label:     "com.devproxy.daemon",
		Program:   "/usr/local/bin/devproxy",
		Arguments: []string{"daemon"},
		Env: map[string]string{
			"HOME":   "/Users/alice",
			"CAROOT": "/Users/alice/Library/Application Support/mkcert",
		},
	})
	if !strings.Contains(plist, "<key>HOME</key>") || !strings.Contains(plist, "/Users/alice") {
		t.Fatalf("expected launchd plist to include HOME environment, got %q", plist)
	}
	if !strings.Contains(plist, "<key>CAROOT</key>") || !strings.Contains(plist, "/Users/alice/Library/Application Support/mkcert") {
		t.Fatalf("expected launchd plist to include CAROOT environment, got %q", plist)
	}
}

func TestSetMenubarStartupEnabledInstallsAndUninstallsLaunchAgent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("launchctl script test is unix-only")
	}

	originalResolveGUIUser := resolveGUIUser
	t.Cleanup(func() { resolveGUIUser = originalResolveGUIUser })

	root := t.TempDir()
	homeDir := filepath.Join(root, "Users", "dev")
	paths := InstallPaths{UserLibraryDir: filepath.Join(root, "Library")}
	uid := os.Getuid()
	resolveGUIUser = func() (int, string, error) { return uid, homeDir, nil }

	programPath := MenubarBundleExecutablePath(InstallPaths{UserLibraryDir: filepath.Join(homeDir, "Library")})
	if err := os.MkdirAll(filepath.Dir(programPath), 0o755); err != nil {
		t.Fatalf("create menubar bundle dir: %v", err)
	}
	if err := os.WriteFile(programPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write menubar program: %v", err)
	}

	stateFile := filepath.Join(root, "state")
	binDir := t.TempDir()
	makeFakePlutil(t, binDir)
	makeMenubarToggleLaunchctl(t, binDir, stateFile)
	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+":"+originalPath)

	if err := SetMenubarStartupEnabled(context.Background(), paths, true); err != nil {
		t.Fatalf("enable menubar startup: %v", err)
	}

	menubarCfg := MenubarServiceConfig(InstallPaths{UserLibraryDir: filepath.Join(homeDir, "Library")}, uid)
	if _, err := os.Stat(menubarCfg.PlistPath); err != nil {
		t.Fatalf("expected menubar plist to be installed: %v", err)
	}

	if err := SetMenubarStartupEnabled(context.Background(), paths, false); err != nil {
		t.Fatalf("disable menubar startup: %v", err)
	}
	if _, err := os.Stat(menubarCfg.PlistPath); !os.IsNotExist(err) {
		t.Fatalf("expected menubar plist to be removed, got %v", err)
	}

	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read fake launchctl state: %v", err)
	}
	calls := string(data)
	wantDomain := "gui/" + strconv.Itoa(uid)
	if !strings.Contains(calls, "bootstrap "+wantDomain+" "+menubarCfg.PlistPath) {
		t.Fatalf("expected menubar bootstrap call, got %q", calls)
	}
	if !strings.Contains(calls, "bootout "+wantDomain+" "+menubarCfg.PlistPath) {
		t.Fatalf("expected menubar bootout call, got %q", calls)
	}
}

func TestMenubarServiceConfigWritesLogsToUserLibrary(t *testing.T) {
	t.Parallel()

	paths := InstallPaths{UserLibraryDir: "/Users/alice/Library"}
	cfg := MenubarServiceConfig(paths, 502)
	if got, want := cfg.StdoutLog, "/Users/alice/Library/Logs/DevProxy/menubar.stdout.log"; got != want {
		t.Fatalf("expected stdout log %q, got %q", want, got)
	}
	if got, want := cfg.StderrLog, "/Users/alice/Library/Logs/DevProxy/menubar.stderr.log"; got != want {
		t.Fatalf("expected stderr log %q, got %q", want, got)
	}
}

func TestMenubarPlistIncludesInteractiveAquaSession(t *testing.T) {
	t.Parallel()

	plist := plistFor(LaunchdServiceConfig{
		Label:     "com.devproxy.menubar",
		Domain:    DomainAgent,
		Program:   "/usr/local/bin/devproxy",
		Arguments: []string{"menubar"},
	})

	if !strings.Contains(plist, "<key>LimitLoadToSessionType</key>") || !strings.Contains(plist, "<string>Aqua</string>") {
		t.Fatalf("expected menubar plist to limit load to Aqua session")
	}
	if !strings.Contains(plist, "<key>ProcessType</key>") || !strings.Contains(plist, "<string>Interactive</string>") {
		t.Fatalf("expected menubar plist to set interactive process type")
	}
}

func makeFakeLaunchctl(t *testing.T, stateFile string, printMissing bool) string {
	t.Helper()
	binDir := t.TempDir()
	scriptPath := filepath.Join(binDir, "launchctl")
	printBody := "service is loaded"
	printExit := "1"
	if printMissing {
		printBody = "Could not find service"
		printExit = "113"
	}
	script := "#!/bin/sh\n" +
		"echo \"$*\" >> \"" + stateFile + "\"\n" +
		"if [ \"$1\" = \"bootout\" ]; then\n" +
		"  echo \"Boot-out failed: 5: Input/output error\" >&2\n" +
		"  exit 5\n" +
		"fi\n" +
		"if [ \"$1\" = \"print\" ]; then\n" +
		"  echo \"" + printBody + "\" >&2\n" +
		"  exit " + printExit + "\n" +
		"fi\n" +
		"exit 0\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake launchctl: %v", err)
	}
	return binDir
}

func makeFakePlutil(t *testing.T, binDir string) {
	t.Helper()
	scriptPath := filepath.Join(binDir, "plutil")
	script := "#!/bin/sh\n" +
		"echo \"OK\"\n" +
		"exit 0\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake plutil: %v", err)
	}
}

func makeFailingBootstrapLaunchctl(t *testing.T, binDir, stateFile string) {
	t.Helper()
	scriptPath := filepath.Join(binDir, "launchctl")
	script := "#!/bin/sh\n" +
		"echo \"$*\" >> \"" + stateFile + "\"\n" +
		"if [ \"$1\" = \"bootout\" ]; then\n" +
		"  echo \"Could not find service\" >&2\n" +
		"  exit 113\n" +
		"fi\n" +
		"if [ \"$1\" = \"bootstrap\" ]; then\n" +
		"  echo \"Bootstrap failed: 5: Input/output error\" >&2\n" +
		"  exit 5\n" +
		"fi\n" +
		"if [ \"$1\" = \"print\" ]; then\n" +
		"  echo \"state = waiting\" >&2\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 0\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake launchctl: %v", err)
	}
}

func makeIdempotentLaunchctl(t *testing.T, binDir, stateFile string) {
	t.Helper()
	scriptPath := filepath.Join(binDir, "launchctl")
	script := "#!/bin/sh\n" +
		"echo \"$*\" >> \"" + stateFile + "\"\n" +
		"if [ \"$1\" = \"bootstrap\" ]; then\n" +
		"  echo running > \"" + stateFile + ".status\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"bootout\" ]; then\n" +
		"  echo \"Boot-out failed: 5: Input/output error\" >&2\n" +
		"  exit 5\n" +
		"fi\n" +
		"if [ \"$1\" = \"print\" ]; then\n" +
		"  if [ -f \"" + stateFile + ".status\" ]; then\n" +
		"    echo \"state = running\"\n" +
		"    exit 0\n" +
		"  fi\n" +
		"  echo \"Could not find service\" >&2\n" +
		"  exit 113\n" +
		"fi\n" +
		"exit 0\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake launchctl: %v", err)
	}
}

func makeNonRunningLaunchctl(t *testing.T, binDir, stateFile string) {
	t.Helper()
	scriptPath := filepath.Join(binDir, "launchctl")
	script := "#!/bin/sh\n" +
		"echo \"$*\" >> \"" + stateFile + "\"\n" +
		"if [ \"$1\" = \"bootout\" ]; then\n" +
		"  echo \"Could not find service\" >&2\n" +
		"  exit 113\n" +
		"fi\n" +
		"if [ \"$1\" = \"print\" ]; then\n" +
		"  echo \"state = spawn scheduled\"\n" +
		"  echo \"last exit code = 1\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 0\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake launchctl: %v", err)
	}
}

func makeMenubarToggleLaunchctl(t *testing.T, binDir, stateFile string) {
	t.Helper()
	scriptPath := filepath.Join(binDir, "launchctl")
	statusFile := stateFile + ".status"
	script := "#!/bin/sh\n" +
		"echo \"$*\" >> \"" + stateFile + "\"\n" +
		"if [ \"$1\" = \"bootstrap\" ]; then\n" +
		"  echo running > \"" + statusFile + "\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"bootout\" ]; then\n" +
		"  if [ -f \"" + statusFile + "\" ]; then\n" +
		"    rm -f \"" + statusFile + "\"\n" +
		"    exit 0\n" +
		"  fi\n" +
		"  echo \"Could not find service\" >&2\n" +
		"  exit 113\n" +
		"fi\n" +
		"if [ \"$1\" = \"print\" ]; then\n" +
		"  if [ -f \"" + statusFile + "\" ]; then\n" +
		"    echo \"state = running\"\n" +
		"    exit 0\n" +
		"  fi\n" +
		"  echo \"Could not find service\" >&2\n" +
		"  exit 113\n" +
		"fi\n" +
		"exit 0\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake launchctl: %v", err)
	}
}
