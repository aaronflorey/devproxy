package daemon

import (
	"context"
	"errors"
	"strings"
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

func configFixture() config.Config {
	return config.Config{DomainSuffix: "test", RootServices: []string{"app", "web"}, Serving: config.ServingConfig{ManagedSuffix: "test"}}
}
