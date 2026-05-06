package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
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

	if info.Mode().Perm() != 0o660 {
		t.Fatalf("expected socket permissions 0660, got %o", info.Mode().Perm())
	}
}

func TestSetAdminSocketAccess_DarwinLookupAndChownBestEffort(t *testing.T) {
	path := filepath.Join(t.TempDir(), "admin.sock")
	if err := os.WriteFile(path, []byte("socket-marker"), 0o600); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	origLookupGroup := lookupGroup
	origChown := osChown
	t.Cleanup(func() {
		lookupGroup = origLookupGroup
		osChown = origChown
	})

	lookupCalled := false
	chownCalled := false

	lookupGroup = func(name string) (*user.Group, error) {
		lookupCalled = true
		if name != "admin" {
			t.Fatalf("expected admin group lookup, got %q", name)
		}
		return &user.Group{Gid: "80"}, nil
	}
	osChown = func(name string, uid, gid int) error {
		chownCalled = true
		if name != path {
			t.Fatalf("expected chown path %q, got %q", path, name)
		}
		if uid != -1 || gid != 80 {
			t.Fatalf("expected chown uid=-1 gid=80, got uid=%d gid=%d", uid, gid)
		}
		return errors.New("simulated chown failure")
	}

	if err := setAdminSocketAccess(path); err != nil {
		t.Fatalf("set admin socket access should succeed despite best-effort chown: %v", err)
	}

	if runtime.GOOS == "darwin" {
		if !lookupCalled {
			t.Fatalf("expected group lookup on darwin")
		}
		if !chownCalled {
			t.Fatalf("expected chown attempt on darwin")
		}
	} else {
		if lookupCalled || chownCalled {
			t.Fatalf("expected no lookup/chown outside darwin")
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat marker: %v", err)
	}
	if info.Mode().Perm() != 0o660 {
		t.Fatalf("expected mode 0660, got %o", info.Mode().Perm())
	}
}

func TestSetAdminSocketAccess_DarwinLookupUnavailableFallsBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "admin.sock")
	if err := os.WriteFile(path, []byte("socket-marker"), 0o600); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	origLookupGroup := lookupGroup
	origChown := osChown
	t.Cleanup(func() {
		lookupGroup = origLookupGroup
		osChown = origChown
	})

	lookupCalled := false
	chownCalled := false

	lookupGroup = func(name string) (*user.Group, error) {
		lookupCalled = true
		return nil, errors.New("group lookup unavailable")
	}
	osChown = func(name string, uid, gid int) error {
		chownCalled = true
		return nil
	}

	if err := setAdminSocketAccess(path); err != nil {
		t.Fatalf("set admin socket access with lookup fallback: %v", err)
	}

	if runtime.GOOS == "darwin" {
		if !lookupCalled {
			t.Fatalf("expected lookup attempt on darwin")
		}
		if chownCalled {
			t.Fatalf("did not expect chown when lookup fails")
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat marker: %v", err)
	}
	if info.Mode().Perm() != 0o660 {
		t.Fatalf("expected mode 0660, got %o", info.Mode().Perm())
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
					Winner:   routing.Candidate{ContainerName: "acme-api-1"},
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

func TestServerRoutingPauseResumeAndStartupEndpoints_D02_D03(t *testing.T) {
	server, socketPath := mustStartTestServer(t, StateSnapshot{})
	t.Cleanup(func() { _ = server.Close(context.Background()) })

	client := unixSocketClient(socketPath)

	t.Run("POST /routing/pause returns explicit paused state", func(t *testing.T) {
		resp := postJSONRequest(t, client, "http://unix/routing/pause", `{}`)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 from /routing/pause, got %d", resp.StatusCode)
		}
		defer resp.Body.Close()
		var payload RoutingPauseResumeResponse
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatalf("decode /routing/pause response: %v", err)
		}
		if !payload.Paused {
			t.Fatalf("expected paused=true from /routing/pause, got %+v", payload)
		}
	})

	t.Run("POST /routing/resume returns explicit paused=false state", func(t *testing.T) {
		resp := postJSONRequest(t, client, "http://unix/routing/resume", `{}`)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 from /routing/resume, got %d", resp.StatusCode)
		}
		defer resp.Body.Close()
		var payload RoutingPauseResumeResponse
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatalf("decode /routing/resume response: %v", err)
		}
		if payload.Paused {
			t.Fatalf("expected paused=false from /routing/resume, got %+v", payload)
		}
	})

	t.Run("GET /startup exposes daemon and menubar role entries", func(t *testing.T) {
		resp, err := client.Get("http://unix/startup")
		if err != nil {
			t.Fatalf("get /startup: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 from /startup, got %d", resp.StatusCode)
		}
		var payload StartupStatusResponse
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatalf("decode /startup response: %v", err)
		}
		if len(payload.Roles) != 2 {
			t.Fatalf("expected daemon+menubar role entries, got %+v", payload)
		}
		assertStartupRoleContainsFields(t, payload, "daemon")
		assertStartupRoleContainsFields(t, payload, "menubar")
	})

	t.Run("POST /startup toggles only requested role", func(t *testing.T) {
		resp := postJSONRequest(t, client, "http://unix/startup", `{"role":"menubar","enabled":true}`)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 from /startup toggle, got %d", resp.StatusCode)
		}
		defer resp.Body.Close()
		var payload StartupToggleResponse
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatalf("decode /startup toggle response: %v", err)
		}
		if payload.Role != "menubar" || !payload.Enabled {
			t.Fatalf("expected menubar enabled toggle result, got %+v", payload)
		}
		if payload.AffectedRole != "menubar" {
			t.Fatalf("expected only requested role to be affected, got %+v", payload)
		}
	})
}

func postJSONRequest(t *testing.T, client *http.Client, url, body string) *http.Response {
	t.Helper()
	resp, err := client.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("post %s: %v", url, err)
	}
	return resp
}

func assertStartupRoleContainsFields(t *testing.T, payload StartupStatusResponse, role string) {
	t.Helper()
	for _, item := range payload.Roles {
		if item.Role != role {
			continue
		}
		if item.Domain == "" || item.Label == "" || item.StatusMessage == "" {
			t.Fatalf("startup role %q missing required fields: %+v", role, item)
		}
		_ = item.Installed
		_ = item.Running
		_ = item.Toggleable
		return
	}
	t.Fatalf("role %q not found in payload: %+v", role, payload)
}

func TestServerRoutingPauseResumeFailureReturnsStructuredError_D02(t *testing.T) {
	server, socketPath := mustStartTestServer(t, StateSnapshot{})
	t.Cleanup(func() { _ = server.Close(context.Background()) })

	server.SetRoutingPauseResume(func(context.Context, bool) error {
		return errors.New("pause/resume unavailable")
	})

	client := unixSocketClient(socketPath)
	resp := postJSONRequest(t, client, "http://unix/routing/pause", `{}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 from /routing/pause failure, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "pause/resume unavailable") {
		t.Fatalf("expected structured pause failure text, got %s", string(body))
	}
}
