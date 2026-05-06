package install

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

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
	if strings.Contains(string(data), "print system/com.devproxy.daemon") {
		t.Fatalf("did not expect print probe for non-missing-state failure")
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
