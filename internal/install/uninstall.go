package install

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type CleanupScope struct {
	Config       bool
	State        bool
	Logs         bool
	Certificates bool
}

type UninstallOptions struct {
	Suffix      string
	WithMenubar bool
	Paths       InstallPaths
	Cleanup     CleanupScope
}

type UninstallDependencies struct {
	CurrentEUID             func() int
	StopDaemonService       func(context.Context, LaunchdServiceConfig) error
	UninstallDaemonService  func(context.Context, LaunchdServiceConfig) error
	StopMenubarService      func(context.Context, LaunchdServiceConfig) error
	UninstallMenubarService func(context.Context, LaunchdServiceConfig) error
	RemoveResolver          func(context.Context, ResolverConfig) error
	RemoveConfig            func(context.Context, InstallPaths) error
	RemoveState             func(context.Context, InstallPaths) error
	RemoveLogs              func(context.Context, InstallPaths) error
	RemoveCertificates      func(context.Context, InstallPaths) error
}

type Uninstaller struct{ deps UninstallDependencies }

func NewUninstaller(deps UninstallDependencies) *Uninstaller {
	if deps.StopDaemonService == nil {
		deps.StopDaemonService = StopService
	}
	if deps.CurrentEUID == nil {
		deps.CurrentEUID = os.Geteuid
	}
	if deps.UninstallDaemonService == nil {
		deps.UninstallDaemonService = UninstallService
	}
	if deps.StopMenubarService == nil {
		deps.StopMenubarService = StopService
	}
	if deps.UninstallMenubarService == nil {
		deps.UninstallMenubarService = UninstallService
	}
	if deps.RemoveResolver == nil {
		deps.RemoveResolver = RemoveResolver
	}
	if deps.RemoveConfig == nil {
		deps.RemoveConfig = removeConfig
	}
	if deps.RemoveState == nil {
		deps.RemoveState = removeState
	}
	if deps.RemoveLogs == nil {
		deps.RemoveLogs = removeLogs
	}
	if deps.RemoveCertificates == nil {
		deps.RemoveCertificates = removeCertificates
	}
	return &Uninstaller{deps: deps}
}

func (u *Uninstaller) Uninstall(ctx context.Context, opts UninstallOptions) error {
	if u.deps.CurrentEUID() != 0 {
		return fmt.Errorf("devproxy uninstall requires root privileges; rerun with sudo")
	}

	paths := opts.Paths
	if paths.ConfigDir == "" {
		paths = DefaultPaths()
	}

	daemonCfg := DaemonServiceConfig(paths)
	if err := u.deps.StopDaemonService(ctx, daemonCfg); err != nil && !isAlreadyRemovedServiceState(err) {
		return fmt.Errorf("stop daemon service: %w", err)
	}
	if err := u.deps.UninstallDaemonService(ctx, daemonCfg); err != nil && !isAlreadyRemovedServiceState(err) {
		return fmt.Errorf("uninstall daemon service: %w", err)
	}
	if opts.WithMenubar {
		menubarCfg := MenubarServiceConfig(paths, os.Getuid())
		if err := u.deps.StopMenubarService(ctx, menubarCfg); err != nil && !isAlreadyRemovedServiceState(err) {
			return fmt.Errorf("stop menubar service: %w", err)
		}
		if err := u.deps.UninstallMenubarService(ctx, menubarCfg); err != nil && !isAlreadyRemovedServiceState(err) {
			return fmt.Errorf("uninstall menubar service: %w", err)
		}
	}

	if err := u.deps.RemoveResolver(ctx, ResolverConfig{Suffix: opts.Suffix, ResolverDir: paths.ResolverDir}); err != nil {
		return fmt.Errorf("remove resolver: %w", err)
	}

	if opts.Cleanup.Config {
		if err := u.deps.RemoveConfig(ctx, paths); err != nil {
			return fmt.Errorf("remove config: %w", err)
		}
	}
	if opts.Cleanup.State {
		if err := u.deps.RemoveState(ctx, paths); err != nil {
			return fmt.Errorf("remove state: %w", err)
		}
	}
	if opts.Cleanup.Logs {
		if err := u.deps.RemoveLogs(ctx, paths); err != nil {
			return fmt.Errorf("remove logs: %w", err)
		}
	}
	if opts.Cleanup.Certificates {
		if err := u.deps.RemoveCertificates(ctx, paths); err != nil {
			return fmt.Errorf("remove certificates: %w", err)
		}
	}

	return nil
}

func RemoveResolver(_ context.Context, cfg ResolverConfig) error {
	suffix := cfg.Suffix
	if suffix == "" {
		suffix = "test"
	}
	resolverDir := cfg.ResolverDir
	if resolverDir == "" {
		resolverDir = "/etc/resolver"
	}
	path := filepath.Join(resolverDir, suffix)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove resolver file %q: %w", path, err)
	}
	return nil
}

func removeConfig(_ context.Context, paths InstallPaths) error { return os.RemoveAll(paths.ConfigDir) }
func removeState(_ context.Context, paths InstallPaths) error  { return os.RemoveAll(paths.StateDir) }
func removeLogs(_ context.Context, paths InstallPaths) error   { return os.RemoveAll(paths.LogDir) }
func removeCertificates(_ context.Context, paths InstallPaths) error {
	return os.RemoveAll(filepath.Join(paths.StateDir, "certs"))
}

func isAlreadyRemovedServiceState(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return isKnownLaunchdMissingState(msg) || isBootoutExitFiveIOError(msg)
}
