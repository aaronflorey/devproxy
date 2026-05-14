package daemon

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mochaka/devproxy/internal/adminapi"
	"github.com/mochaka/devproxy/internal/config"
	"github.com/mochaka/devproxy/internal/discovery"
)

func TestDaemonAppStartDegradesWhenDependenciesUnavailable(t *testing.T) {
	t.Run("docker unavailable still starts admin socket", func(t *testing.T) {
		app := NewApp(AppConfig{
			Config:           configFixture(),
			AdminSocketPath:  filepath.Join(t.TempDir(), "admin.sock"),
			DashboardAddress: "127.0.0.1:0",
			DNSAddress:       "127.0.0.1:0",
			HTTPAddress:      "127.0.0.1:0",
			HTTPSAddress:     "127.0.0.1:0",
			DockerPing:       func(context.Context) error { return errors.New("daemon unreachable") },
			DockerScan:       func(context.Context) ([]ContainerState, error) { return nil, errors.New("docker inspect failed") },
			EnsureMKCert: func(context.Context) error {
				return nil
			},
			BuildNetworkRuntime: func(context.Context) error {
				return nil
			},
		})
		defer func() { _ = app.Close(context.Background()) }()

		if err := app.Start(context.Background()); err != nil {
			t.Fatalf("expected degraded startup, got %v", err)
		}

		client := adminapi.NewClient(app.cfg.AdminSocketPath)
		status, err := client.Status(context.Background())
		if err != nil {
			t.Fatalf("status: %v", err)
		}
		if status.Watcher.Connected {
			t.Fatalf("expected watcher to report disconnected when docker is unavailable")
		}

		issues, err := client.Issues(context.Background())
		if err != nil {
			t.Fatalf("issues: %v", err)
		}
		if len(issues) < 2 {
			t.Fatalf("expected docker startup issues to be recorded, got %+v", issues)
		}
		messages := []string{issues[0].Message, issues[1].Message}
		joined := strings.Join(messages, " ")
		if !strings.Contains(joined, "docker reachability check failed") || !strings.Contains(joined, "docker inspect failed") {
			t.Fatalf("expected docker startup failures in issues, got %+v", issues)
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

	t.Run("listener bind validation failure is recorded", func(t *testing.T) {
		app := NewApp(AppConfig{
			Config:           configFixture(),
			AdminSocketPath:  filepath.Join(t.TempDir(), "admin.sock"),
			DashboardAddress: "127.0.0.1:0",
			DNSAddress:       "127.0.0.1:0",
			HTTPAddress:      "127.0.0.1:0",
			HTTPSAddress:     "127.0.0.1:0",
			DockerPing:       func(context.Context) error { return nil },
			EnsureMKCert:     func(context.Context) error { return nil },
			BuildNetworkRuntime: func(context.Context) error {
				return errors.New("listen tcp 127.0.0.1:80: bind: permission denied")
			},
		})
		defer func() { _ = app.Close(context.Background()) }()

		if err := app.Start(context.Background()); err != nil {
			t.Fatalf("expected degraded startup, got %v", err)
		}

		issues := app.stateSnapshot().Issues
		if len(issues) == 0 || issues[len(issues)-1].Role != "network" {
			t.Fatalf("expected network validation issue to be recorded, got %+v", issues)
		}
		if !strings.Contains(issues[len(issues)-1].Message, "listener bind validation failed") {
			t.Fatalf("expected validation failure message, got %+v", issues[len(issues)-1])
		}
	})

	t.Run("http bind failure still starts admin socket", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("reserve http address: %v", err)
		}
		defer func() { _ = listener.Close() }()

		app := NewApp(AppConfig{
			Config:           configFixture(),
			AdminSocketPath:  filepath.Join(t.TempDir(), "admin.sock"),
			DashboardAddress: "127.0.0.1:0",
			DNSAddress:       "127.0.0.1:0",
			HTTPAddress:      listener.Addr().String(),
			HTTPSAddress:     "127.0.0.1:0",
			EnsureMKCert:     func(context.Context) error { return nil },
		})
		defer func() { _ = app.Close(context.Background()) }()

		if err := app.Start(context.Background()); err != nil {
			t.Fatalf("expected degraded startup, got %v", err)
		}

		client := adminapi.NewClient(app.cfg.AdminSocketPath)
		status, err := client.Status(context.Background())
		if err != nil {
			t.Fatalf("status: %v", err)
		}
		if status.HTTP.Bound {
			t.Fatalf("expected http listener to remain unbound after startup conflict")
		}
		if !strings.Contains(status.HTTP.LastError, "address already in use") {
			t.Fatalf("expected explicit http bind error, got %+v", status.HTTP)
		}

		issues, err := client.Issues(context.Background())
		if err != nil {
			t.Fatalf("issues: %v", err)
		}
		if len(issues) == 0 || issues[0].Role != "http" {
			t.Fatalf("expected http bind issue, got %+v", issues)
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
