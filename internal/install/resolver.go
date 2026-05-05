package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ResolverConfig struct {
	Suffix      string
	ResolverDir string
}

func WriteResolver(cfg ResolverConfig) error {
	suffix := strings.TrimSpace(strings.TrimPrefix(cfg.Suffix, "."))
	if suffix == "" {
		return fmt.Errorf("resolver suffix is required")
	}
	resolverDir := cfg.ResolverDir
	if resolverDir == "" {
		resolverDir = "/etc/resolver"
	}
	contents := fmt.Sprintf("domain %s\nnameserver 127.0.0.1\nport 53535\n", suffix)
	resolverPath := filepath.Join(resolverDir, suffix)
	if err := os.WriteFile(resolverPath, []byte(contents), 0o644); err != nil {
		return fmt.Errorf("write resolver file %q: %w", resolverPath, err)
	}
	return nil
}
