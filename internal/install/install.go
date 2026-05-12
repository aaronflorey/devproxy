package install

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const daemonProgramPath = "/usr/local/bin/devproxy"

var (
	mkcertLookPath   = exec.LookPath
	mkcertInstallRun = runMKCertInstall
)

type Options struct {
	Suffix      string
	WithMenubar bool
	Paths       InstallPaths
	Progress    func(string)
}

type Dependencies struct {
	CurrentEUID             func() int
	EnsurePaths             func(InstallPaths) error
	PrepareDaemonBinary     func(string) error
	PrepareMenubarBundle    func(sourceBinaryPath string, paths InstallPaths) error
	WriteResolver           func(ResolverConfig) error
	BootstrapCertificates   func(context.Context) error
	InstallDaemonService    func(LaunchdServiceConfig) error
	StartDaemonService      func(LaunchdServiceConfig) error
	InstallMenubarService   func(LaunchdServiceConfig) error
	StartMenubarService     func(LaunchdServiceConfig) error
	ResolveGUIUser          func() (uid int, homeDir string, err error)
	ResolveGUIUserOwnership func() (uid int, gid int, homeDir string, err error)
	EnsureMenubarOwnership  func(paths InstallPaths, cfg LaunchdServiceConfig, uid, gid int) error
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
	if deps.PrepareMenubarBundle == nil {
		deps.PrepareMenubarBundle = prepareMenubarBundle
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
	if deps.ResolveGUIUserOwnership == nil {
		deps.ResolveGUIUserOwnership = ResolveGUIUserOwnership
	}
	if deps.EnsureMenubarOwnership == nil {
		deps.EnsureMenubarOwnership = ensureMenubarOwnership
	}
	return &Installer{deps: deps}
}

func ensureMKCertInstalled(context.Context) error {
	if _, err := mkcertLookPath("mkcert"); err != nil {
		return fmt.Errorf("mkcert not found: install mkcert before enabling HTTPS: %w", err)
	}
	if err := mkcertInstallRun(); err != nil {
		return fmt.Errorf("mkcert trust store install failed: %w", err)
	}
	return nil
}

func runMKCertInstall() error {
	cmd := exec.Command("mkcert", "-install")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) == 0 {
			return err
		}
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
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

func prepareMenubarBundle(sourceBinaryPath string, paths InstallPaths) error {
	bundlePath := MenubarBundlePath(paths)
	contentsDir := filepath.Join(bundlePath, "Contents")
	macOSDir := filepath.Join(contentsDir, "MacOS")
	resourcesDir := filepath.Join(contentsDir, "Resources")
	executablePath := MenubarBundleExecutablePath(paths)

	for _, dir := range []string{macOSDir, resourcesDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create menubar bundle directory %q: %w", dir, err)
		}
	}

	if err := copyFile(sourceBinaryPath, executablePath, 0o755); err != nil {
		return fmt.Errorf("stage menubar bundle executable %q: %w", executablePath, err)
	}

	infoPlist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDisplayName</key>
	<string>DevProxy Menubar</string>
	<key>CFBundleExecutable</key>
	<string>%s</string>
	<key>CFBundleIdentifier</key>
	<string>com.devproxy.menubar.app</string>
	<key>CFBundleName</key>
	<string>DevProxy Menubar</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>1.0</string>
	<key>CFBundleVersion</key>
	<string>1</string>
	<key>LSUIElement</key>
	<string>1</string>
</dict>
</plist>
`, filepath.Base(executablePath))
	infoPlistPath := filepath.Join(contentsDir, "Info.plist")
	if err := os.WriteFile(infoPlistPath, []byte(infoPlist), 0o644); err != nil {
		return fmt.Errorf("write menubar bundle Info.plist %q: %w", infoPlistPath, err)
	}

	return nil
}

func copyFile(sourcePath, targetPath string, mode os.FileMode) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open source file %q: %w", sourcePath, err)
	}
	defer source.Close()

	tempPath := targetPath + ".tmp"
	destination, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("create target file %q: %w", tempPath, err)
	}

	if _, err := io.Copy(destination, source); err != nil {
		destination.Close()
		_ = os.Remove(tempPath)
		return fmt.Errorf("copy %q to %q: %w", sourcePath, tempPath, err)
	}
	if err := destination.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("finalize copied file %q: %w", tempPath, err)
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("activate copied file %q: %w", targetPath, err)
	}
	if err := os.Chmod(targetPath, mode); err != nil {
		return fmt.Errorf("set permissions on %q: %w", targetPath, err)
	}
	return nil
}

func ensureMenubarOwnership(paths InstallPaths, cfg LaunchdServiceConfig, uid, gid int) error {
	for _, path := range []string{MenubarBundlePath(paths), filepath.Dir(cfg.StdoutLog)} {
		if path == "" {
			continue
		}
		if err := chownRecursive(path, uid, gid); err != nil {
			return err
		}
	}
	if cfg.PlistPath != "" {
		if err := os.Chown(cfg.PlistPath, uid, gid); err != nil {
			return fmt.Errorf("chown menubar plist %q to uid=%d gid=%d: %w", cfg.PlistPath, uid, gid, err)
		}
	}
	return nil
}

func chownRecursive(root string, uid, gid int) error {
	if root == "" {
		return nil
	}
	if _, err := os.Stat(root); err != nil {
		return fmt.Errorf("stat %q before chown: %w", root, err)
	}
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if chownErr := os.Chown(path, uid, gid); chownErr != nil {
			return fmt.Errorf("chown %q to uid=%d gid=%d: %w", path, uid, gid, chownErr)
		}
		return nil
	})
}

func (i *Installer) Install(ctx context.Context, opts Options) error {
	if i.deps.CurrentEUID() != 0 {
		return fmt.Errorf("devproxy install requires root privileges; rerun with sudo (needs access to /usr/local/etc/devproxy, /var/lib/devproxy, /var/log/devproxy, /etc/resolver, and /Library/LaunchDaemons)")
	}

	progress := opts.Progress
	if progress == nil {
		progress = func(string) {}
	}

	paths := opts.Paths
	if paths.ConfigDir == "" {
		paths = DefaultPaths()
	}

	guiHome := ""
	if _, resolvedGUIHome, err := i.deps.ResolveGUIUser(); err == nil {
		guiHome = resolvedGUIHome
	}

	progress("Ensuring install paths")
	if err := i.deps.EnsurePaths(paths); err != nil {
		return fmt.Errorf("ensure install paths: %w", err)
	}
	progress("Writing resolver configuration")
	if err := i.deps.WriteResolver(ResolverConfig{Suffix: opts.Suffix, ResolverDir: paths.ResolverDir, StateDir: paths.StateDir}); err != nil {
		return fmt.Errorf("install managed resolver: %w", err)
	}
	progress("Bootstrapping TLS prerequisites")
	if err := i.deps.BootstrapCertificates(ctx); err != nil {
		return fmt.Errorf("bootstrap certificates: %w", err)
	}
	progress("Staging devproxy binary")
	if err := i.deps.PrepareDaemonBinary(daemonProgramPath); err != nil {
		return fmt.Errorf("stage daemon executable at %q: %w", daemonProgramPath, err)
	}

	daemonCfg := DaemonServiceConfig(paths, guiHome)
	progress("Installing daemon service")
	if err := i.deps.InstallDaemonService(daemonCfg); err != nil {
		return fmt.Errorf("install daemon service: %w", err)
	}
	progress("Starting daemon service")
	if err := i.deps.StartDaemonService(daemonCfg); err != nil {
		return fmt.Errorf("start daemon service: %w", err)
	}

	if opts.WithMenubar {
		progress("Resolving active GUI user")
		guiUID, guiGID, guiHome, err := i.deps.ResolveGUIUserOwnership()
		if err != nil {
			return fmt.Errorf("resolve GUI user for menubar service: %w", err)
		}

		menubarPaths := paths
		menubarPaths.UserLibraryDir = filepath.Join(guiHome, "Library")
		progress("Preparing menubar app bundle")
		if err := i.deps.PrepareMenubarBundle(daemonProgramPath, menubarPaths); err != nil {
			return fmt.Errorf("prepare menubar app bundle: %w", err)
		}
		menubarCfg := MenubarServiceConfig(menubarPaths, guiUID)
		progress("Installing menubar service")
		if err := i.deps.InstallMenubarService(menubarCfg); err != nil {
			return fmt.Errorf("install menubar service: %w", err)
		}
		progress("Fixing menubar file ownership")
		if err := i.deps.EnsureMenubarOwnership(menubarPaths, menubarCfg, guiUID, guiGID); err != nil {
			return fmt.Errorf("fix menubar file ownership: %w", err)
		}
		progress("Starting menubar service")
		if err := i.deps.StartMenubarService(menubarCfg); err != nil {
			return fmt.Errorf("start menubar service: %w", err)
		}
	}

	return nil
}
