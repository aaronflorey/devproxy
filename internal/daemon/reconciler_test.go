package daemon

import (
	"testing"

	"github.com/mochaka/devproxy/internal/discovery"
)

func TestReconcilerStartupRefreshAndEvents(t *testing.T) {
	containers := []ContainerState{{ID: "1", Name: "acme-api-1", Running: true, Labels: map[string]string{"com.docker.compose.project": "acme", "com.docker.compose.service": "api"}, Ports: []discovery.PublishedPort{{HostPort: 8080, Protocol: "tcp"}}}}
	rec := NewReconciler(ReconcilerOptions{Suffix: "test", RootServices: []string{"app", "web", "nginx", "laravel.test"}})

	if err := rec.RebuildSnapshot(containers); err != nil {
		t.Fatalf("rebuild failed: %v", err)
	}
	if rec.LastSync().IsZero() {
		t.Fatalf("expected sync timestamp")
	}

	watcher := NewWatcher(rec)
	watcher.OnDisconnect()
	if watcher.Health().Connected {
		t.Fatalf("expected degraded health after disconnect")
	}

	watcher.OnReconnect(containers)
	if !watcher.Health().Connected || watcher.Health().LastReconnectSync.IsZero() {
		t.Fatalf("expected reconnect and full resync")
	}
}
