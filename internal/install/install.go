package install

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

type Options struct {
	Suffix      string
	WithMenubar bool
	Paths       InstallPaths
}

type Dependencies struct {
	CurrentEUID           func() int
	EnsurePaths           func(InstallPaths) error
	WriteResolver         func(ResolverConfig) error
	BootstrapCertificates func(context.Context) error
	InstallDaemonService  func(LaunchdServiceConfig) error
	StartDaemonService    func(LaunchdServiceConfig) error
	InstallMenubarService func(LaunchdServiceConfig) error
	StartMenubarService   func(LaunchdServiceConfig) error
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
	return &Installer{deps: deps}
}

func ensureMKCertInstalled(context.Context) error {
	if _, err := exec.LookPath("mkcert"); err != nil {
		return fmt.Errorf("mkcert not found: install mkcert before enabling HTTPS: %w", err)
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

	daemonCfg := DaemonServiceConfig(paths)
	if err := i.deps.InstallDaemonService(daemonCfg); err != nil {
		return fmt.Errorf("install daemon service: %w", err)
	}
	if err := i.deps.StartDaemonService(daemonCfg); err != nil {
		return fmt.Errorf("start daemon service: %w", err)
	}

	if opts.WithMenubar {
		menubarCfg := MenubarServiceConfig(paths)
		if err := i.deps.InstallMenubarService(menubarCfg); err != nil {
			return fmt.Errorf("install menubar service: %w", err)
		}
		if err := i.deps.StartMenubarService(menubarCfg); err != nil {
			return fmt.Errorf("start menubar service: %w", err)
		}
	}

	return nil
}
