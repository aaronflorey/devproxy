package daemon

import (
	"context"
	"errors"
	"strings"
	"testing"
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
			DockerPing:    func(context.Context) error { return nil },
			EnsureMKCert:  func(context.Context) error { return errors.New("mkcert not found") },
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
			DockerPing: func(context.Context) error { return nil },
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
