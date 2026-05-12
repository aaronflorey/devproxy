package install

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ResolverConfig struct {
	Suffix      string
	ResolverDir string
	StateDir    string
}

func WriteResolver(cfg ResolverConfig) error {
	suffix, resolverDir, err := normalizeResolverConfig(cfg)
	if err != nil {
		return err
	}
	contents := managedResolverContents(suffix)
	resolverPath := filepath.Join(resolverDir, suffix)

	existing, err := os.ReadFile(resolverPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read resolver file %q: %w", resolverPath, err)
	}
	if err == nil && string(existing) != contents {
		if backupErr := writeResolverBackup(cfg, existing); backupErr != nil {
			return backupErr
		}
	}

	if err := os.WriteFile(resolverPath, []byte(contents), 0o644); err != nil {
		return fmt.Errorf("write resolver file %q: %w", resolverPath, err)
	}
	return nil
}

func RemoveResolver(_ context.Context, cfg ResolverConfig) error {
	suffix, resolverDir, err := normalizeResolverConfig(cfg)
	if err != nil {
		return err
	}

	resolverPath := filepath.Join(resolverDir, suffix)
	backupPath := resolverBackupPath(cfg, suffix)
	if backupPath != "" {
		if backup, backupErr := os.ReadFile(backupPath); backupErr == nil {
			if err := os.WriteFile(resolverPath, backup, 0o644); err != nil {
				return fmt.Errorf("restore resolver file %q: %w", resolverPath, err)
			}
			if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove resolver backup %q: %w", backupPath, err)
			}
			return nil
		} else if !os.IsNotExist(backupErr) {
			return fmt.Errorf("read resolver backup %q: %w", backupPath, backupErr)
		}
	}

	path := resolverPath
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove resolver file %q: %w", path, err)
	}
	return nil
}

func normalizeResolverConfig(cfg ResolverConfig) (string, string, error) {
	suffix := strings.TrimSpace(strings.TrimPrefix(cfg.Suffix, "."))
	if suffix == "" {
		return "", "", fmt.Errorf("resolver suffix is required")
	}
	resolverDir := cfg.ResolverDir
	if resolverDir == "" {
		resolverDir = "/etc/resolver"
	}
	return suffix, resolverDir, nil
}

func managedResolverContents(suffix string) string {
	return fmt.Sprintf("domain %s\nnameserver 127.0.0.1\nport 53535\n", suffix)
}

func resolverBackupPath(cfg ResolverConfig, suffix string) string {
	stateDir := strings.TrimSpace(cfg.StateDir)
	if stateDir == "" {
		return ""
	}
	return filepath.Join(stateDir, "resolver-backups", suffix)
}

func writeResolverBackup(cfg ResolverConfig, contents []byte) error {
	suffix := strings.TrimSpace(strings.TrimPrefix(cfg.Suffix, "."))
	backupPath := resolverBackupPath(cfg, suffix)
	if backupPath == "" {
		return nil
	}
	if _, err := os.Stat(backupPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat resolver backup %q: %w", backupPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		return fmt.Errorf("create resolver backup directory %q: %w", filepath.Dir(backupPath), err)
	}
	if err := os.WriteFile(backupPath, contents, 0o644); err != nil {
		return fmt.Errorf("write resolver backup %q: %w", backupPath, err)
	}
	return nil
}
