package install

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

const daemonProgramPath = "/usr/local/bin/devproxy"

type Options struct {
	Suffix      string
	WithMenubar bool
	Paths       InstallPaths
}

type Dependencies struct {
	CurrentEUID           func() int
	EnsurePaths           func(InstallPaths) error
	PrepareDaemonBinary   func(string) error
	WriteResolver         func(ResolverConfig) error
	BootstrapCertificates func(context.Context) error
	InstallDaemonService  func(LaunchdServiceConfig) error
	StartDaemonService    func(LaunchdServiceConfig) error
	InstallMenubarService func(LaunchdServiceConfig) error
	StartMenubarService   func(LaunchdServiceConfig) error
	ResolveGUIUser        func() (uid int, homeDir string, err error)
}

type Installer struct {
	deps Dependencies
}

func NewInstaller(deps Dependencies) *Installer {
	if deps.EnsurePaths == nil {
		deps.EnsurePaths = EnsurePaths
	}
	if deps.CurrentEUID == nil {
		deps.CurrentEUID = os.Geteuid
	}
	if deps.WriteResolver == nil {
		deps.WriteResolver = WriteResolver
	}
	if deps.PrepareDaemonBinary == nil {
		deps.PrepareDaemonBinary = stageCurrentExecutable
	}
	if deps.BootstrapCertificates == nil {
		deps.BootstrapCertificates = ensureMKCertInstalled
	}
	if deps.InstallDaemonService == nil {
		deps.InstallDaemonService = InstallService
	}
	if deps.StartDaemonService == nil {
		deps.StartDaemonService = StartService
	}
	if deps.InstallMenubarService == nil {
		deps.InstallMenubarService = InstallService
	}
	if deps.StartMenubarService == nil {
		deps.StartMenubarService = StartService
	}
	if deps.ResolveGUIUser == nil {
		deps.ResolveGUIUser = ResolveGUIUser
	}
	return &Installer{deps: deps}
}

func ensureMKCertInstalled(context.Context) error {
	if _, err := exec.LookPath("mkcert"); err != nil {
		return fmt.Errorf("mkcert not found: install mkcert before enabling HTTPS: %w", err)
	}
	return nil
}

func stageCurrentExecutable(targetPath string) error {
	currentPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("detect current executable path: %w", err)
	}

	source, err := os.Open(currentPath)
	if err != nil {
		return fmt.Errorf("open current executable %q: %w", currentPath, err)
	}
	defer source.Close()

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create daemon binary directory %q: %w", filepath.Dir(targetPath), err)
	}

	tempPath := targetPath + ".tmp"
	destination, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("create staged executable %q: %w", tempPath, err)
	}

	if _, err := io.Copy(destination, source); err != nil {
		destination.Close()
		_ = os.Remove(tempPath)
		return fmt.Errorf("copy executable to %q: %w", tempPath, err)
	}
	if err := destination.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("finalize staged executable %q: %w", tempPath, err)
	}

	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("activate staged executable %q: %w", targetPath, err)
	}

	if err := os.Chmod(targetPath, 0o755); err != nil {
		return fmt.Errorf("set executable permissions on %q: %w", targetPath, err)
	}

	return nil
}

func (i *Installer) Install(ctx context.Context, opts Options) error {
	if i.deps.CurrentEUID() != 0 {
		return fmt.Errorf("devproxy install requires root privileges; rerun with sudo (needs access to /usr/local/etc/devproxy, /var/lib/devproxy, /var/log/devproxy, /etc/resolver, and /Library/LaunchDaemons)")
	}

	paths := opts.Paths
	if paths.ConfigDir == "" {
		paths = DefaultPaths()
	}

	if err := i.deps.EnsurePaths(paths); err != nil {
		return fmt.Errorf("ensure install paths: %w", err)
	}
	if err := i.deps.WriteResolver(ResolverConfig{Suffix: opts.Suffix, ResolverDir: paths.ResolverDir}); err != nil {
		return fmt.Errorf("install managed resolver: %w", err)
	}
	if err := i.deps.BootstrapCertificates(ctx); err != nil {
		return fmt.Errorf("bootstrap certificates: %w", err)
	}
	if err := i.deps.PrepareDaemonBinary(daemonProgramPath); err != nil {
		return fmt.Errorf("stage daemon executable at %q: %w", daemonProgramPath, err)
	}

	daemonCfg := DaemonServiceConfig(paths)
	if err := i.deps.InstallDaemonService(daemonCfg); err != nil {
		return fmt.Errorf("install daemon service: %w", err)
	}
	if err := i.deps.StartDaemonService(daemonCfg); err != nil {
		return fmt.Errorf("start daemon service: %w", err)
	}

	if opts.WithMenubar {
		guiUID, guiHome, err := i.deps.ResolveGUIUser()
		if err != nil {
			return fmt.Errorf("resolve GUI user for menubar service: %w", err)
		}

		menubarPaths := paths
		menubarPaths.UserLibraryDir = filepath.Join(guiHome, "Library")
		menubarCfg := MenubarServiceConfig(menubarPaths, guiUID)
		if err := i.deps.InstallMenubarService(menubarCfg); err != nil {
			return fmt.Errorf("install menubar service: %w", err)
		}
		if err := i.deps.StartMenubarService(menubarCfg); err != nil {
			return fmt.Errorf("start menubar service: %w", err)
		}
	}

	return nil
}
