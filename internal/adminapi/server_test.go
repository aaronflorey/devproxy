package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mochaka/devproxy/internal/admin"
	"github.com/mochaka/devproxy/internal/routing"
)

func TestAdminAPIRemovesStaleSocketBeforeBind(t *testing.T) {
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "devproxy.sock")
	if err := os.WriteFile(socketPath, []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale socket marker: %v", err)
	}

	server, err := NewServer(ServerConfig{
		SocketPath: socketPath,
		State:      staticStateProvider(),
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	t.Cleanup(func() { _ = server.Close(context.Background()) })

	if err := server.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}

	info, err := os.Stat(socketPath)
	if err != nil {
		t.Fatalf("stat socket: %v", err)
	}
	if info.Mode().Type() != os.ModeSocket {
		t.Fatalf("expected unix socket at %q, got mode %s", socketPath, info.Mode())
	}

	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected socket permissions 0600, got %o", info.Mode().Perm())
	}
}

func TestAdminAPIUsesSharedReadModelsAndSerializesJSON(t *testing.T) {
	network := admin.NetworkDoctorStatus{DNSHealthy: true, HTTPBound: true, HTTPSBound: false, ManagedSuffix: "test"}
	state := StateSnapshot{
		Snapshot: routing.Snapshot{
			Version: "v1",
			Routes: map[string]routing.Route{
				"api.acme.test": {
					Hostname: "api.acme.test",
					Winner: routing.Candidate{ContainerName: "acme-api-1"},
					Upstream: routing.Upstream{Scheme: "http", Host: "127.0.0.1", Port: 8080},
				},
			},
			Warnings: []routing.Warning{{Code: "W1", Message: "warning"}},
		},
		Status: admin.StatusView{SnapshotVersion: "v1", LastSync: time.Now().UTC(), DNS: admin.DNSStatus{Healthy: true, ManagedSuffix: "test"}},
		Doctor: admin.BuildDoctor(routing.Snapshot{Warnings: []routing.Warning{{Code: "W1", Message: "warning"}}}, network),
		Logs:   admin.BuildSessionEvents(routing.Snapshot{Warnings: []routing.Warning{{Message: "warning"}}}),
	}

	server, socketPath := mustStartTestServer(t, state)
	t.Cleanup(func() { _ = server.Close(context.Background()) })

	client := unixSocketClient(socketPath)

	statusResp := getJSON[StatusResponse](t, client, "http://unix/status")
	if statusResp.Status.SnapshotVersion != "v1" {
		t.Fatalf("expected status snapshot version v1, got %q", statusResp.Status.SnapshotVersion)
	}

	routesResp := getJSON[RoutesResponse](t, client, "http://unix/routes")
	if len(routesResp.Routes) != 1 || routesResp.Routes[0].Hostname != "api.acme.test" {
		t.Fatalf("expected routes payload from snapshot read model, got %+v", routesResp.Routes)
	}

	doctorResp := getJSON[DoctorResponse](t, client, "http://unix/doctor")
	if !doctorResp.Doctor.Network.DNSHealthy {
		t.Fatalf("expected doctor payload network dns health true")
	}

	logsResp := getJSON[LogsResponse](t, client, "http://unix/logs")
	if len(logsResp.Events) == 0 {
		t.Fatalf("expected logs payload events")
	}
}

func TestRegisterCoreCommandsIncludesDaemonAndPrintConfig(t *testing.T) {
	var names []string
	RegisterCoreCommands(func(factory CommandFactory) {
		names = append(names, factory().Name())
	})

	joined := strings.Join(names, ",")
	if !strings.Contains(joined, "print-config") {
		t.Fatalf("expected print-config command in registry, got %v", names)
	}
	if !strings.Contains(joined, "daemon") {
		t.Fatalf("expected daemon command in registry, got %v", names)
	}
}

func mustStartTestServer(t *testing.T, state StateSnapshot) (*Server, string) {
	t.Helper()
	socketPath := filepath.Join(t.TempDir(), "admin.sock")
	server, err := NewServer(ServerConfig{SocketPath: socketPath, State: staticStateProviderWith(state)})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	if err := server.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	return server, socketPath
}

func unixSocketClient(socketPath string) *http.Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}
	return &http.Client{Transport: transport}
}

func getJSON[T any](t *testing.T, client *http.Client, url string) T {
	t.Helper()
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("http get %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from %s, got %d", url, resp.StatusCode)
	}
	var out T
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode json %s: %v", url, err)
	}
	return out
}

func staticStateProvider() func() StateSnapshot {
	return staticStateProviderWith(StateSnapshot{})
}

func staticStateProviderWith(state StateSnapshot) func() StateSnapshot {
	return func() StateSnapshot { return state }
}

func TestAdminAPIRefreshReturnsFailurePayloadFromDaemonError(t *testing.T) {
	errRefresh := errors.New("docker ping failed")
	state := StateSnapshot{}
	server, socketPath := mustStartTestServer(t, state)
	t.Cleanup(func() { _ = server.Close(context.Background()) })

	server.SetRefreshFunc(func(context.Context, string) error {
		return errRefresh
	})

	client := unixSocketClient(socketPath)
	body := strings.NewReader(`{"reason":"operator"}`)
	resp, err := client.Post("http://unix/refresh", "application/json", body)
	if err != nil {
		t.Fatalf("post refresh: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected refresh failure status 503, got %d", resp.StatusCode)
	}
	var payload RefreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode refresh payload: %v", err)
	}
	if payload.Error == "" || !strings.Contains(payload.Error, "docker ping failed") {
		t.Fatalf("expected refresh error payload to expose bootstrap failure, got %+v", payload)
	}
}
