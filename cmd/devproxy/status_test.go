package devproxy

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mochaka/devproxy/internal/admin"
	"github.com/mochaka/devproxy/internal/adminapi"
	"github.com/mochaka/devproxy/internal/routing"
)

func TestStatusCommandPrintsConflictAndWarningDetail(t *testing.T) {
	now := time.Now().UTC()
	state := adminapi.StateSnapshot{
		Status: admin.StatusView{
			SnapshotVersion:  "v1",
			ActiveRoutes:     1,
			Conflicts:        1,
			Warnings:         1,
			ConflictDetails:  []routing.Conflict{{Hostname: "api.acme.test", Winner: routing.Candidate{ContainerName: "acme-api-1"}, Losers: []routing.Candidate{{ContainerName: "acme-api-2"}}, Reason: "higher priority winner kept route"}},
			WarningDetails:   []routing.Warning{{Code: "invalid_label", Message: "ignored invalid label", Container: "acme-api-2", Field: "domain"}},
			LastSync:         now,
			DNS:              admin.DNSStatus{Healthy: true, ManagedSuffix: "test"},
			HTTP:             admin.ListenerStatus{Enabled: true, Bound: true},
			HTTPS:            admin.ListenerStatus{Enabled: true, Bound: true},
			CertificateReady: true,
		},
	}

	socketPath, shutdown := startCommandTestServer(t, state)
	defer shutdown()

	out := &bytes.Buffer{}
	cmd := newStatusCommand()
	cmd.SetOut(out)
	cmd.SetArgs([]string{"--admin-socket", socketPath})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("status command failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "conflicts:") || !strings.Contains(output, "acme-api-2") {
		t.Fatalf("expected conflict loser detail in output, got %q", output)
	}
	if !strings.Contains(output, "warnings:") || !strings.Contains(output, "ignored invalid label") {
		t.Fatalf("expected warning detail in output, got %q", output)
	}
}

func startCommandTestServer(t *testing.T, state adminapi.StateSnapshot) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "devproxy-")
	if err != nil {
		t.Fatalf("create short temp dir: %v", err)
	}
	socketPath := filepath.Join(dir, "admin.sock")
	server, err := adminapi.NewServer(adminapi.ServerConfig{SocketPath: socketPath, State: func() adminapi.StateSnapshot { return state }})
	if err != nil {
		t.Fatalf("new admin server: %v", err)
	}
	if err := server.Start(); err != nil {
		t.Fatalf("start admin server: %v", err)
	}
	return socketPath, func() {
		_ = server.Close(context.Background())
		_ = os.RemoveAll(dir)
	}
}
