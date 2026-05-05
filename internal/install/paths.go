package install

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

type InstallPaths struct {
	ConfigDir      string
	StateDir       string
	LogDir         string
	ResolverDir    string
	LaunchDaemons  string
	UserLibraryDir string
}

func DefaultPaths() InstallPaths {
	userLibraryDir := "/Library"
	if u, err := user.Current(); err == nil && u.HomeDir != "" {
		userLibraryDir = filepath.Join(u.HomeDir, "Library")
	}

	return InstallPaths{
		ConfigDir:      "/usr/local/etc/devproxy",
		StateDir:       "/var/lib/devproxy",
		LogDir:         "/var/log/devproxy",
		ResolverDir:    "/etc/resolver",
		LaunchDaemons:  "/Library/LaunchDaemons",
		UserLibraryDir: userLibraryDir,
	}
}

func EnsurePaths(paths InstallPaths) error {
	for _, dir := range []string{paths.ConfigDir, paths.StateDir, paths.LogDir, paths.ResolverDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create install path %q: %w", dir, err)
		}
	}
	return nil
}
