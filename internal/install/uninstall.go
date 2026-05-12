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
	Progress    func(string)
}

type UninstallDependencies struct {
	CurrentEUID             func() int
	ResolveGUIUser          func() (uid int, homeDir string, err error)
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
	if deps.ResolveGUIUser == nil {
		deps.ResolveGUIUser = ResolveGUIUser
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

	progress := opts.Progress
	if progress == nil {
		progress = func(string) {}
	}

	paths := opts.Paths
	if paths.ConfigDir == "" {
		paths = DefaultPaths()
	}

	progress("Stopping daemon service")
	daemonCfg := DaemonServiceConfig(paths, "")
	if err := u.deps.StopDaemonService(ctx, daemonCfg); err != nil && !isAlreadyRemovedServiceState(err) {
		return fmt.Errorf("stop daemon service: %w", err)
	}
	progress("Removing daemon service")
	if err := u.deps.UninstallDaemonService(ctx, daemonCfg); err != nil && !isAlreadyRemovedServiceState(err) {
		return fmt.Errorf("uninstall daemon service: %w", err)
	}

	// Always attempt menubar removal best-effort; WithMenubar makes it required.
	menubarErr := func() error {
		guiUID, guiHome, err := u.deps.ResolveGUIUser()
		if err != nil {
			if opts.WithMenubar {
				return fmt.Errorf("resolve GUI user for menubar uninstall: %w", err)
			}
			return nil
		}
		menubarPaths := paths
		menubarPaths.UserLibraryDir = filepath.Join(guiHome, "Library")
		menubarCfg := MenubarServiceConfig(menubarPaths, guiUID)
		progress("Stopping menubar service")
		if err := u.deps.StopMenubarService(ctx, menubarCfg); err != nil && !isAlreadyRemovedServiceState(err) {
			if opts.WithMenubar {
				return fmt.Errorf("stop menubar service: %w", err)
			}
		}
		progress("Removing menubar service")
		if err := u.deps.UninstallMenubarService(ctx, menubarCfg); err != nil && !isAlreadyRemovedServiceState(err) {
			if opts.WithMenubar {
				return fmt.Errorf("uninstall menubar service: %w", err)
			}
		}
		progress("Removing menubar app bundle")
		if err := os.RemoveAll(MenubarBundlePath(menubarPaths)); err != nil {
			if opts.WithMenubar {
				return fmt.Errorf("remove menubar app bundle: %w", err)
			}
		}
		return nil
	}()
	if menubarErr != nil {
		return menubarErr
	}

	progress("Removing resolver configuration")
	if err := u.deps.RemoveResolver(ctx, ResolverConfig{Suffix: opts.Suffix, ResolverDir: paths.ResolverDir, StateDir: paths.StateDir}); err != nil {
		return fmt.Errorf("remove resolver: %w", err)
	}

	if opts.Cleanup.Config {
		progress("Removing config")
		if err := u.deps.RemoveConfig(ctx, paths); err != nil {
			return fmt.Errorf("remove config: %w", err)
		}
	}
	if opts.Cleanup.State {
		progress("Removing state")
		if err := u.deps.RemoveState(ctx, paths); err != nil {
			return fmt.Errorf("remove state: %w", err)
		}
	}
	if opts.Cleanup.Logs {
		progress("Removing logs")
		if err := u.deps.RemoveLogs(ctx, paths); err != nil {
			return fmt.Errorf("remove logs: %w", err)
		}
	}
	if opts.Cleanup.Certificates {
		progress("Removing certificates")
		if err := u.deps.RemoveCertificates(ctx, paths); err != nil {
			return fmt.Errorf("remove certificates: %w", err)
		}
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
