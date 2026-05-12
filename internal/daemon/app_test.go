package daemon

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mochaka/devproxy/internal/config"
	"github.com/mochaka/devproxy/internal/discovery"
)

func TestDaemonAppBootstrapFailsClearlyWhenDependenciesUnavailable(t *testing.T) {
	t.Run("docker ping fails", func(t *testing.T) {
		app := NewApp(AppConfig{
			DockerPing: func(context.Context) error { return errors.New("daemon unreachable") },
			EnsureMKCert: func(context.Context) error {
				t.Fatal("expected bootstrap to stop before mkcert when docker is unavailable")
				return nil
			},
			BuildNetworkRuntime: func(context.Context) error {
				t.Fatal("expected bootstrap to stop before network runtime when docker is unavailable")
				return nil
			},
		})

		err := app.Start(context.Background())
		if err == nil || !strings.Contains(err.Error(), "docker reachability") {
			t.Fatalf("expected explicit docker reachability failure, got %v", err)
		}
	})

	t.Run("mkcert prerequisite fails", func(t *testing.T) {
		app := NewApp(AppConfig{
			DockerPing:   func(context.Context) error { return nil },
			EnsureMKCert: func(context.Context) error { return errors.New("mkcert not found") },
			BuildNetworkRuntime: func(context.Context) error {
				t.Fatal("expected bootstrap to stop before network runtime when mkcert check fails")
				return nil
			},
		})

		err := app.Start(context.Background())
		if err == nil || !strings.Contains(err.Error(), "mkcert prerequisites") {
			t.Fatalf("expected explicit mkcert prerequisite failure, got %v", err)
		}
	})

	t.Run("listener bind fails", func(t *testing.T) {
		app := NewApp(AppConfig{
			DockerPing:   func(context.Context) error { return nil },
			EnsureMKCert: func(context.Context) error { return nil },
			BuildNetworkRuntime: func(context.Context) error {
				return errors.New("listen tcp 127.0.0.1:80: bind: permission denied")
			},
		})

		err := app.Start(context.Background())
		if err == nil || !strings.Contains(err.Error(), "listener bind") {
			t.Fatalf("expected explicit listener bind failure, got %v", err)
		}
	})
}

func TestRefreshUsesDockerScanSnapshot(t *testing.T) {
	t.Parallel()

	app := NewApp(AppConfig{
		Config: configFixture(),
		DockerScan: func(context.Context) ([]ContainerState, error) {
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

	if err := app.Refresh(context.Background()); err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if got := len(app.reconciler.Snapshot().Routes); got != 1 {
		t.Fatalf("expected one route after docker scan refresh, got %d", got)
	}
	if !app.watcher.Health().Connected {
		t.Fatalf("expected watcher health to report connected after successful scan")
	}
}

func TestRefreshRecordsDockerScanFailures(t *testing.T) {
	t.Parallel()

	app := NewApp(AppConfig{
		Config:     configFixture(),
		DockerScan: func(context.Context) ([]ContainerState, error) { return nil, errors.New("docker inspect failed") },
	})

	err := app.Refresh(context.Background())
	if err == nil || !strings.Contains(err.Error(), "docker container sync failed") {
		t.Fatalf("expected docker sync failure, got %v", err)
	}
	if app.watcher.Health().Connected {
		t.Fatalf("expected watcher health to report disconnected after scan failure")
	}
	issues := app.stateSnapshot().Issues
	if len(issues) == 0 || issues[0].Role != "docker" {
		t.Fatalf("expected docker issue to be recorded, got %+v", issues)
	}
	if issues[0].Timestamp.Before(time.Now().Add(-time.Minute)) {
		t.Fatalf("expected recent issue timestamp, got %+v", issues[0])
	}
}

func TestAppStartProcessesLiveDockerEvents(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eventCh := make(chan DockerEvent, 1)
	errCh := make(chan error)
	var mu sync.RWMutex
	containers := []ContainerState{testContainer("acme", "api", "acme-api-1", 8080)}
	scanCalls := 0
	app := NewApp(AppConfig{
		Config:           configFixture(),
		AdminSocketPath:  filepath.Join(t.TempDir(), "admin.sock"),
		DashboardAddress: "127.0.0.1:0",
		DNSAddress:       "127.0.0.1:0",
		HTTPAddress:      "127.0.0.1:0",
		HTTPSAddress:     "127.0.0.1:0",
		DockerScan: func(context.Context) ([]ContainerState, error) {
			mu.Lock()
			defer mu.Unlock()
			scanCalls++
			return append([]ContainerState(nil), containers...), nil
		},
		DockerEvents: func(context.Context) (*EventStream, error) {
			return &EventStream{Events: eventCh, Errors: errCh}, nil
		},
	})
	defer func() { _ = app.Close(context.Background()) }()

	if err := app.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	mu.Lock()
	containers = []ContainerState{
		testContainer("acme", "api", "acme-api-1", 8080),
		testContainer("acme", "docs", "acme-docs-1", 8081),
	}
	mu.Unlock()
	eventCh <- DockerEvent{Action: "start"}

	waitFor(t, func() bool {
		_, ok := app.reconciler.Snapshot().Routes["docs.acme.test"]
		return ok && scanCalls >= 2
	}, "live docker event to rebuild snapshot")
}

func TestAppWatcherReconnectsWithFullResync(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.RWMutex
	containers := []ContainerState{testContainer("acme", "api", "acme-api-1", 8080)}
	scanCalls := 0
	streamErrors := []chan error{make(chan error, 1), make(chan error)}
	streamEvents := []chan DockerEvent{make(chan DockerEvent), make(chan DockerEvent)}
	connectCount := 0
	app := NewApp(AppConfig{
		Config:           configFixture(),
		AdminSocketPath:  filepath.Join(t.TempDir(), "admin.sock"),
		DashboardAddress: "127.0.0.1:0",
		DNSAddress:       "127.0.0.1:0",
		HTTPAddress:      "127.0.0.1:0",
		HTTPSAddress:     "127.0.0.1:0",
		DockerScan: func(context.Context) ([]ContainerState, error) {
			mu.Lock()
			defer mu.Unlock()
			scanCalls++
			return append([]ContainerState(nil), containers...), nil
		},
		DockerEvents: func(context.Context) (*EventStream, error) {
			stream := &EventStream{Events: streamEvents[connectCount], Errors: streamErrors[connectCount]}
			connectCount++
			return stream, nil
		},
	})
	defer func() { _ = app.Close(context.Background()) }()

	if err := app.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	streamErrors[0] <- errors.New("docker events dropped")
	waitFor(t, func() bool { return !app.watcher.Health().Connected }, "watcher disconnect after stream error")

	mu.Lock()
	containers = []ContainerState{testContainer("acme", "docs", "acme-docs-1", 8081)}
	mu.Unlock()

	waitFor(t, func() bool {
		health := app.watcher.Health()
		_, ok := app.reconciler.Snapshot().Routes["docs.acme.test"]
		return health.Connected && !health.LastReconnectSync.IsZero() && ok && scanCalls >= 2
	}, "watcher reconnect with full resync")
}

func testContainer(project, service, name string, port int) ContainerState {
	return ContainerState{
		ID:      name,
		Name:    name,
		Running: true,
		Labels: map[string]string{
			"com.docker.compose.project": project,
			"com.docker.compose.service": service,
		},
		Ports: []discovery.PublishedPort{{HostPort: port, Protocol: "tcp"}},
	}
}

func waitFor(t *testing.T, check func() bool, description string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", description)
}

func configFixture() config.Config {
	return config.Config{DomainSuffix: "test", RootServices: []string{"app", "web"}, Serving: config.ServingConfig{ManagedSuffix: "test"}}
}
