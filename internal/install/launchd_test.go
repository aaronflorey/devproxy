package install

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
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
		"if [ \"$1\" = \"bootout\" ]; then\n" +
		"  echo \"Boot-out failed: 5: Input/output error\" >&2\n" +
		"  exit 5\n" +
		"fi\n" +
		"if [ \"$1\" = \"print\" ]; then\n" +
		"  echo \"Could not find service\" >&2\n" +
		"  exit 113\n" +
		"fi\n" +
		"exit 0\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake launchctl: %v", err)
	}
}
