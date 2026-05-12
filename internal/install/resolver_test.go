package install

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteResolverBacksUpExistingResolverAndRemoveRestoresIt(t *testing.T) {
	t.Parallel()

	resolverDir := t.TempDir()
	stateDir := t.TempDir()
	resolverPath := filepath.Join(resolverDir, "test")
	original := "domain test\nnameserver 127.0.0.1\nport 53\n"
	if err := os.WriteFile(resolverPath, []byte(original), 0o644); err != nil {
		t.Fatalf("seed resolver: %v", err)
	}

	cfg := ResolverConfig{Suffix: "test", ResolverDir: resolverDir, StateDir: stateDir}
	if err := WriteResolver(cfg); err != nil {
		t.Fatalf("write resolver: %v", err)
	}

	got, err := os.ReadFile(resolverPath)
	if err != nil {
		t.Fatalf("read managed resolver: %v", err)
	}
	if string(got) != managedResolverContents("test") {
		t.Fatalf("unexpected managed resolver contents: %q", string(got))
	}

	backup, err := os.ReadFile(filepath.Join(stateDir, "resolver-backups", "test"))
	if err != nil {
		t.Fatalf("read resolver backup: %v", err)
	}
	if string(backup) != original {
		t.Fatalf("unexpected backup contents: %q", string(backup))
	}

	if err := RemoveResolver(context.Background(), cfg); err != nil {
		t.Fatalf("remove resolver: %v", err)
	}

	restored, err := os.ReadFile(resolverPath)
	if err != nil {
		t.Fatalf("read restored resolver: %v", err)
	}
	if string(restored) != original {
		t.Fatalf("unexpected restored resolver contents: %q", string(restored))
	}
}

func TestRemoveResolverDeletesManagedFileWhenNoBackupExists(t *testing.T) {
	t.Parallel()

	resolverDir := t.TempDir()
	stateDir := t.TempDir()
	resolverPath := filepath.Join(resolverDir, "test")
	if err := os.WriteFile(resolverPath, []byte(managedResolverContents("test")), 0o644); err != nil {
		t.Fatalf("seed managed resolver: %v", err)
	}

	if err := RemoveResolver(context.Background(), ResolverConfig{Suffix: "test", ResolverDir: resolverDir, StateDir: stateDir}); err != nil {
		t.Fatalf("remove resolver: %v", err)
	}
	if _, err := os.Stat(resolverPath); !os.IsNotExist(err) {
		t.Fatalf("expected resolver file removed, stat err=%v", err)
	}
}
