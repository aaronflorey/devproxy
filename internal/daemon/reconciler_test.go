package daemon

import (
	"testing"

	"github.com/mochaka/devproxy/internal/config"
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
				"devproxy.scheme":            "https",
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

func TestReconcilerAppliesMergedOverrides(t *testing.T) {
	port := 3000
	priority := 7
	rec := NewReconciler(ReconcilerOptions{
		Suffix:       "test",
		RootServices: []string{"app", "web", "nginx", "laravel.test"},
		Overrides: map[string]config.ProjectConfig{
			"acme": {
				Services: map[string]config.ServiceOverride{
					"api": {
						Domains:  []string{"edge.acme.test"},
						Root:     boolPtr(true),
						Port:     &port,
						Scheme:   "https",
						Priority: &priority,
					},
				},
			},
		},
	})

	containers := []ContainerState{{
		ID:      "1",
		Name:    "acme-api-1",
		Running: true,
		Labels: map[string]string{
			"com.docker.compose.project": "acme",
			"com.docker.compose.service": "api",
		},
		Ports: []discovery.PublishedPort{{HostPort: 8080, Protocol: "tcp"}, {HostPort: 3000, Protocol: "tcp"}},
	}}

	if err := rec.RebuildSnapshot(containers); err != nil {
		t.Fatalf("rebuild failed: %v", err)
	}

	snap := rec.Snapshot()
	route, ok := snap.Routes["acme.test"]
	if !ok {
		t.Fatalf("expected root override hostname to be published")
	}
	if _, ok := snap.Routes["edge.acme.test"]; !ok {
		t.Fatalf("expected extra override domain to be published")
	}
	if route.Upstream.Port != 3000 {
		t.Fatalf("expected override port 3000, got %d", route.Upstream.Port)
	}
	if route.Upstream.Scheme != "https" {
		t.Fatalf("expected override scheme https, got %q", route.Upstream.Scheme)
	}
	if route.Priority != 7 {
		t.Fatalf("expected override priority 7, got %d", route.Priority)
	}
	if len(route.ServedHostnames) != 2 {
		t.Fatalf("expected served hostnames from merged domains, got %v", route.ServedHostnames)
	}
}

func TestReconcilerLabelOverridesConfigForPortAndScheme(t *testing.T) {
	configPort := 3000
	rec := NewReconciler(ReconcilerOptions{
		Suffix:       "test",
		RootServices: []string{"app", "web", "nginx", "laravel.test"},
		Overrides: map[string]config.ProjectConfig{
			"acme": {
				Services: map[string]config.ServiceOverride{
					"api": {Port: &configPort, Scheme: "https"},
				},
			},
		},
	})

	containers := []ContainerState{{
		ID:      "1",
		Name:    "acme-api-1",
		Running: true,
		Labels: map[string]string{
			"com.docker.compose.project": "acme",
			"com.docker.compose.service": "api",
			"devproxy.port":              "8080",
			"devproxy.scheme":            "http",
		},
		Ports: []discovery.PublishedPort{{HostPort: 8080, Protocol: "tcp"}, {HostPort: 3000, Protocol: "tcp"}},
	}}

	if err := rec.RebuildSnapshot(containers); err != nil {
		t.Fatalf("rebuild failed: %v", err)
	}

	route, ok := rec.Snapshot().Routes["api.acme.test"]
	if !ok {
		t.Fatalf("expected default hostname route")
	}
	if route.Upstream.Port != 8080 {
		t.Fatalf("expected label port 8080 to override config, got %d", route.Upstream.Port)
	}
	if route.Upstream.Scheme != "http" {
		t.Fatalf("expected label scheme http to override config, got %q", route.Upstream.Scheme)
	}
	if route.Provenance.PortSource != "label" {
		t.Fatalf("expected label port provenance, got %q", route.Provenance.PortSource)
	}
}

func TestReconcilerExplicitRootFalseSuppressesDefaultRootHostname(t *testing.T) {
	rootFalse := false
	rec := NewReconciler(ReconcilerOptions{
		Suffix:       "test",
		RootServices: []string{"app", "web", "nginx", "laravel.test"},
		Overrides: map[string]config.ProjectConfig{
			"acme": {
				Services: map[string]config.ServiceOverride{
					"app": {Root: &rootFalse},
				},
			},
		},
	})

	containers := []ContainerState{{
		ID:      "1",
		Name:    "acme-app-1",
		Running: true,
		Labels: map[string]string{
			"com.docker.compose.project": "acme",
			"com.docker.compose.service": "app",
		},
		Ports: []discovery.PublishedPort{{HostPort: 8080, Protocol: "tcp"}},
	}}

	if err := rec.RebuildSnapshot(containers); err != nil {
		t.Fatalf("rebuild failed: %v", err)
	}

	snap := rec.Snapshot()
	if _, ok := snap.Routes["acme.test"]; ok {
		t.Fatalf("expected explicit root false to suppress root hostname, got %+v", snap.Routes)
	}
	if _, ok := snap.Routes["app.acme.test"]; !ok {
		t.Fatalf("expected default service hostname to remain published, got %+v", snap.Routes)
	}
}

func boolPtr(v bool) *bool { return &v }
