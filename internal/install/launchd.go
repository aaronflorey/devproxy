package install

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type LaunchdDomain string

const (
	DomainSystem LaunchdDomain = "system"
	DomainAgent  LaunchdDomain = "gui"
)

type LaunchdServiceConfig struct {
	Label     string
	Domain    LaunchdDomain
	PlistPath string
	Program   string
	Arguments []string
}

func DaemonServiceConfig(paths InstallPaths) LaunchdServiceConfig {
	return LaunchdServiceConfig{
		Label:     "com.devproxy.daemon",
		Domain:    DomainSystem,
		PlistPath: filepath.Join(paths.LaunchDaemons, "com.devproxy.daemon.plist"),
		Program:   "/usr/local/bin/devproxy",
		Arguments: []string{"daemon"},
	}
}

func MenubarServiceConfig(paths InstallPaths) LaunchdServiceConfig {
	return LaunchdServiceConfig{
		Label:     "com.devproxy.menubar",
		Domain:    DomainAgent,
		PlistPath: filepath.Join(paths.UserLibraryDir, "LaunchAgents", "com.devproxy.menubar.plist"),
		Program:   "/usr/local/bin/devproxy",
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
	if err != nil && isKnownLaunchdMissingState(err.Error()) {
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
		uid := os.Getuid()
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
