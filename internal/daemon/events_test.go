package daemon

import (
	"context"
	"testing"

	"github.com/mochaka/devproxy/internal/discovery"
)

func TestWatcherReconnectMarksHealthyAfterFullResync(t *testing.T) {
	rec := NewReconciler(ReconcilerOptions{Suffix: "test", RootServices: []string{"app", "web"}})
	watcher := NewWatcher(rec)
	watcher.OnDisconnect()

	err := watcher.OnReconnect([]ContainerState{testContainer("acme", "api", "acme-api-1", 8080)})
	if err != nil {
		t.Fatalf("reconnect failed: %v", err)
	}

	health := watcher.Health()
	if !health.Connected || health.LastReconnectSync.IsZero() {
		t.Fatalf("expected watcher to become healthy after full resync, got %+v", health)
	}
	if _, ok := rec.Snapshot().Routes["api.acme.test"]; !ok {
		t.Fatalf("expected reconnect sync to rebuild snapshot, got %+v", rec.Snapshot().Routes)
	}
}

func TestHandleDockerEventIgnoresUnsupportedActions(t *testing.T) {
	called := false
	app := NewApp(AppConfig{
		Config: configFixture(),
		DockerScan: func(context.Context) ([]ContainerState, error) {
			called = true
			return []ContainerState{{
				ID:      "1",
				Name:    "acme-api-1",
				Running: true,
				Labels: map[string]string{
					"com.docker.compose.project": "acme",
					"com.docker.compose.service": "api",
				},
				Ports: []discovery.PublishedPort{{HostPort: 8080, Protocol: "tcp"}},
			}}, nil
		},
	})

	before := app.reconciler.Snapshot()
	app.handleDockerEvent(context.Background(), DockerEvent{Action: "pause"})
	after := app.reconciler.Snapshot()

	if called {
		t.Fatalf("expected unsupported event to be ignored before scanning docker")
	}
	if len(after.Routes) != len(before.Routes) {
		t.Fatalf("expected unsupported event not to mutate snapshot, before=%+v after=%+v", before, after)
	}
}
