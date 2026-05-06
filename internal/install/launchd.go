package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

type StartupRoleStatus struct {
	Role          string
	Domain        string
	Label         string
	Installed     bool
	Running       bool
	Toggleable    bool
	StatusMessage string
}

type LaunchdDomain string

const (
	DomainSystem LaunchdDomain = "system"
	DomainAgent  LaunchdDomain = "gui"
)

type LaunchdServiceConfig struct {
	Label     string
	Domain    LaunchdDomain
	AgentUID  int
	PlistPath string
	Program   string
	Arguments []string
	StdoutLog string
	StderrLog string
}

const launchdDefaultPath = "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"

func DaemonServiceConfig(paths InstallPaths) LaunchdServiceConfig {
	return LaunchdServiceConfig{
		Label:     "com.devproxy.daemon",
		Domain:    DomainSystem,
		PlistPath: filepath.Join(paths.LaunchDaemons, "com.devproxy.daemon.plist"),
		Program:   daemonProgramPath,
		Arguments: []string{"daemon"},
		StdoutLog: filepath.Join(paths.LogDir, "daemon.stdout.log"),
		StderrLog: filepath.Join(paths.LogDir, "daemon.stderr.log"),
	}
}

func MenubarServiceConfig(paths InstallPaths, agentUID int) LaunchdServiceConfig {
	return LaunchdServiceConfig{
		Label:     "com.devproxy.menubar",
		Domain:    DomainAgent,
		AgentUID:  agentUID,
		PlistPath: filepath.Join(paths.UserLibraryDir, "LaunchAgents", "com.devproxy.menubar.plist"),
		Program:   daemonProgramPath,
		Arguments: []string{"menubar"},
		StdoutLog: filepath.Join(paths.LogDir, "menubar.stdout.log"),
		StderrLog: filepath.Join(paths.LogDir, "menubar.stderr.log"),
	}
}

func InstallService(cfg LaunchdServiceConfig) error {
	if err := os.MkdirAll(filepath.Dir(cfg.PlistPath), 0o755); err != nil {
		return fmt.Errorf("create launchd plist directory: %w", err)
	}
	if err := os.WriteFile(cfg.PlistPath, []byte(plistFor(cfg)), 0o644); err != nil {
		return fmt.Errorf("write launchd plist %q: %w", cfg.PlistPath, err)
	}
	return nil
}

func StartService(cfg LaunchdServiceConfig) error {
	if err := validateLaunchdPreflight(cfg); err != nil {
		return err
	}

	if err := stopServiceBestEffort(cfg); err != nil {
		return err
	}

	if err := runLaunchctl("bootstrap", domainTarget(cfg), cfg.PlistPath); err != nil {
		return bootstrapDiagnosticError(cfg, err)
	}

	if err := runLaunchctl("kickstart", "-k", fmt.Sprintf("%s/%s", domainTarget(cfg), cfg.Label)); err != nil {
		return fmt.Errorf("launchd service bootstrapped but kickstart failed for %s/%s: %w", domainTarget(cfg), cfg.Label, err)
	}

	return nil
}

func StopService(_ context.Context, cfg LaunchdServiceConfig) error {
	err := runLaunchctl("bootout", domainTarget(cfg), cfg.PlistPath)
	if err == nil {
		return nil
	}
	errMsg := err.Error()
	if isKnownLaunchdMissingState(errMsg) {
		return nil
	}
	if !isBootoutExitFiveIOError(errMsg) {
		return err
	}
	if serviceAlreadyMissing(cfg) {
		return nil
	}
	return err
}

func UninstallService(_ context.Context, cfg LaunchdServiceConfig) error {
	if err := os.Remove(cfg.PlistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove launchd plist %q: %w", cfg.PlistPath, err)
	}
	return nil
}

func plistFor(cfg LaunchdServiceConfig) string {
	args := ""
	for _, arg := range cfg.Arguments {
		args += fmt.Sprintf("\n        <string>%s</string>", arg)
	}
	stdoutLog := ""
	if cfg.StdoutLog != "" {
		stdoutLog = fmt.Sprintf("\n    <key>StandardOutPath</key>\n    <string>%s</string>", cfg.StdoutLog)
	}
	stderrLog := ""
	if cfg.StderrLog != "" {
		stderrLog = fmt.Sprintf("\n    <key>StandardErrorPath</key>\n    <string>%s</string>", cfg.StderrLog)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>%s
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>%s</string>
    </dict>%s%s
</dict>
</plist>
`, cfg.Label, cfg.Program, args, launchdDefaultPath, stdoutLog, stderrLog)
}

func domainTarget(cfg LaunchdServiceConfig) string {
	if cfg.Domain == DomainSystem {
		return "system"
	}
	if cfg.Domain == DomainAgent {
		uid := cfg.AgentUID
		if uid <= 0 {
			uid = os.Getuid()
		}
		return fmt.Sprintf("gui/%d", uid)
	}
	return string(cfg.Domain)
}

func runLaunchctl(args ...string) error {
	_, err := runCommand("launchctl", args...)
	return err
}

func runCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	trimmed := strings.TrimSpace(string(output))
	if err != nil {
		return trimmed, fmt.Errorf("%s %s failed: %w: %s", command, strings.Join(args, " "), err, trimmed)
	}
	return trimmed, nil
}

func stopServiceBestEffort(cfg LaunchdServiceConfig) error {
	err := runLaunchctl("bootout", domainTarget(cfg), cfg.PlistPath)
	if err == nil {
		return nil
	}
	errMsg := err.Error()
	if isKnownLaunchdMissingState(errMsg) {
		return nil
	}
	if !isBootoutExitFiveIOError(errMsg) {
		return err
	}
	if serviceAlreadyMissing(cfg) {
		return nil
	}
	return err
}

func validateLaunchdPreflight(cfg LaunchdServiceConfig) error {
	if _, err := os.Stat(cfg.PlistPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("launchd preflight failed: plist %q does not exist; reinstall and retry", cfg.PlistPath)
		}
		return fmt.Errorf("launchd preflight failed: cannot stat plist %q: %w", cfg.PlistPath, err)
	}

	if _, err := runCommand("plutil", "-lint", cfg.PlistPath); err != nil {
		return fmt.Errorf("launchd preflight failed: plist validation failed for %q: %w", cfg.PlistPath, err)
	}

	if cfg.Program == "" {
		return fmt.Errorf("launchd preflight failed: program path is empty in service config")
	}
	programInfo, err := os.Stat(cfg.Program)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("launchd preflight failed: program %q does not exist; reinstall devproxy binary", cfg.Program)
		}
		return fmt.Errorf("launchd preflight failed: cannot stat program %q: %w", cfg.Program, err)
	}
	if programInfo.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("launchd preflight failed: program %q is not executable; run chmod 755 %q", cfg.Program, cfg.Program)
	}

	if cfg.Domain == DomainSystem {
		if err := validateSystemDaemonPlistPerms(cfg.PlistPath); err != nil {
			return err
		}
	}

	return nil
}

func validateSystemDaemonPlistPerms(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("launchd preflight failed: cannot stat daemon plist %q: %w", path, err)
	}

	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		if runtime.GOOS == "darwin" && (stat.Uid != 0 || stat.Gid != 0) {
			return fmt.Errorf("launchd preflight failed: daemon plist %q must be owned by root:wheel (uid 0 gid 0), found uid %d gid %d; run sudo chown root:wheel %q", path, stat.Uid, stat.Gid, path)
		}
	}

	perm := info.Mode().Perm()
	if perm&0o022 != 0 {
		return fmt.Errorf("launchd preflight failed: daemon plist %q permissions are %#o; group/other write bits must be disabled (recommended 0644)", path, perm)
	}

	return nil
}

func bootstrapDiagnosticError(cfg LaunchdServiceConfig, bootstrapErr error) error {
	serviceTarget := fmt.Sprintf("%s/%s", domainTarget(cfg), cfg.Label)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("launchd bootstrap failed for %s using %q: %v", serviceTarget, cfg.PlistPath, bootstrapErr))

	hints := []string{
		"Verify plist ownership/perms (root:wheel, mode 0644) and binary permissions (0755)",
		"Validate plist with: plutil -lint " + strconv.Quote(cfg.PlistPath),
		"Confirm program path exists: " + strconv.Quote(cfg.Program),
	}
	sb.WriteString("\nLikely causes:\n- " + strings.Join(hints, "\n- "))

	printOutput, printErr := runCommand("launchctl", "print", serviceTarget)
	if printErr != nil {
		sb.WriteString("\nlaunchctl print diagnostics: " + printErr.Error())
	} else {
		sb.WriteString("\nlaunchctl print diagnostics:\n" + printOutput)
	}

	return errors.New(sb.String())
}

func isKnownLaunchdMissingState(message string) bool {
	msg := strings.ToLower(message)
	return strings.Contains(msg, "could not find service") ||
		strings.Contains(msg, "service already unloaded") ||
		strings.Contains(msg, "no such process") ||
		strings.Contains(msg, "no such file")
}

func isBootoutExitFiveIOError(message string) bool {
	msg := strings.ToLower(message)
	return strings.Contains(msg, "boot-out failed: 5") && strings.Contains(msg, "input/output error")
}

func serviceAlreadyMissing(cfg LaunchdServiceConfig) bool {
	err := runLaunchctl("print", fmt.Sprintf("%s/%s", domainTarget(cfg), cfg.Label))
	if err == nil {
		return false
	}
	return isKnownLaunchdMissingState(err.Error())
}

func StartupStatuses(paths InstallPaths) []StartupRoleStatus {
	daemonCfg := DaemonServiceConfig(paths)
	menubarCfg := MenubarServiceConfig(paths, os.Getuid())

	daemonInstalled := fileExists(daemonCfg.PlistPath)
	menubarInstalled := fileExists(menubarCfg.PlistPath)

	daemonRunning := serviceRunning(daemonCfg)
	menubarRunning := serviceRunning(menubarCfg)

	statuses := []StartupRoleStatus{
		{
			Role:          "daemon",
			Domain:        domainTarget(daemonCfg),
			Label:         daemonCfg.Label,
			Installed:     daemonInstalled,
			Running:       daemonRunning,
			Toggleable:    false,
			StatusMessage: daemonStatusMessage(daemonInstalled, daemonRunning),
		},
		{
			Role:          "menubar",
			Domain:        domainTarget(menubarCfg),
			Label:         menubarCfg.Label,
			Installed:     menubarInstalled,
			Running:       menubarRunning,
			Toggleable:    true,
			StatusMessage: menubarStatusMessage(menubarInstalled, menubarRunning),
		},
	}

	return statuses
}

func SetMenubarStartupEnabled(ctx context.Context, paths InstallPaths, enabled bool) error {
	menubarCfg := MenubarServiceConfig(paths, os.Getuid())
	if enabled {
		if err := InstallService(menubarCfg); err != nil {
			return err
		}
		if err := StartService(menubarCfg); err != nil {
			return err
		}
		return nil
	}

	if err := StopService(ctx, menubarCfg); err != nil {
		return err
	}
	if err := UninstallService(ctx, menubarCfg); err != nil {
		return err
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func serviceRunning(cfg LaunchdServiceConfig) bool {
	err := runLaunchctl("print", fmt.Sprintf("%s/%s", domainTarget(cfg), cfg.Label))
	return err == nil
}

func daemonStatusMessage(installed, running bool) string {
	if !installed {
		return "Daemon launchd service is not installed"
	}
	if running {
		return "Managed by system launchd"
	}
	return "Installed but not currently running"
}

func menubarStatusMessage(installed, running bool) string {
	if !installed {
		return "Does not start at login"
	}
	if running {
		return "Starts at login"
	}
	return "Installed for login but not currently running"
}
