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

func TestReconcilerBuildsUpstreamMetadataAndServedHostInventory(t *testing.T) {
	containers := []ContainerState{
		{
			ID:      "1",
			Name:    "acme-api-1",
			Running: true,
			Labels: map[string]string{
				"com.docker.compose.project": "acme",
				"com.docker.compose.service": "api",
				"devproxy.scheme":          "https",
			},
			Ports: []discovery.PublishedPort{
				{HostPort: 8080, Protocol: "tcp"},
			},
		},
	}

	rec := NewReconciler(ReconcilerOptions{Suffix: "test", RootServices: []string{"app", "web", "nginx", "laravel.test"}})
	if err := rec.RebuildSnapshot(containers); err != nil {
		t.Fatalf("rebuild failed: %v", err)
	}

	snap := rec.Snapshot()
	route, ok := snap.Routes["api.acme.test"]
	if !ok {
		t.Fatalf("expected winning route for api.acme.test")
	}

	if route.Upstream.Scheme != "https" {
		t.Fatalf("expected scheme from effective metadata, got %q", route.Upstream.Scheme)
	}

	if route.Upstream.Port != 8080 {
		t.Fatalf("expected upstream port 8080, got %d", route.Upstream.Port)
	}

	if len(route.ServedHostnames) == 0 {
		t.Fatalf("expected served hostname inventory for winner route")
	}
}

func TestReconcilerPauseStateDoesNotDeleteSnapshotRoutes(t *testing.T) {
	containers := []ContainerState{{
		ID:      "1",
		Name:    "acme-api-1",
		Running: true,
		Labels: map[string]string{
			"com.docker.compose.project": "acme",
			"com.docker.compose.service": "api",
		},
		Ports: []discovery.PublishedPort{{HostPort: 8080, Protocol: "tcp"}},
	}}

	rec := NewReconciler(ReconcilerOptions{Suffix: "test", RootServices: []string{"app", "web", "nginx", "laravel.test"}})
	if err := rec.RebuildSnapshot(containers); err != nil {
		t.Fatalf("rebuild failed: %v", err)
	}

	before := rec.Snapshot()
	if len(before.Routes) == 0 {
		t.Fatalf("expected non-empty routes before pause")
	}

	rec.SetRoutingPaused(true)

	after := rec.Snapshot()
	if len(after.Routes) != len(before.Routes) {
		t.Fatalf("expected pause state not to mutate snapshot routes, got %d want %d", len(after.Routes), len(before.Routes))
	}

	if !rec.IsRoutingPaused() {
		t.Fatalf("expected paused state to be explicit runtime flag")
	}
}
