package install

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
}

func DaemonServiceConfig(paths InstallPaths) LaunchdServiceConfig {
	return LaunchdServiceConfig{
		Label:     "com.devproxy.daemon",
		Domain:    DomainSystem,
		PlistPath: filepath.Join(paths.LaunchDaemons, "com.devproxy.daemon.plist"),
		Program:   daemonProgramPath,
		Arguments: []string{"daemon"},
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
	return runLaunchctl("bootstrap", domainTarget(cfg), cfg.PlistPath)
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
</dict>
</plist>
`, cfg.Label, cfg.Program, args)
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
	cmd := exec.Command("launchctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
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
