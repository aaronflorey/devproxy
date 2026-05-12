package adminapi

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mochaka/devproxy/internal/admin"
	"github.com/mochaka/devproxy/internal/routing"
)

func TestClientStatusRoutesAndLogsDecodePayloads(t *testing.T) {
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
		},
		Status: admin.StatusView{SnapshotVersion: "v1", ActiveRoutes: 1},
		Logs:   []admin.LogEvent{{Type: "route", Message: "active route", Hostname: "api.acme.test"}},
	}

	server, socketPath := mustStartTestServer(t, state)
	t.Cleanup(func() { _ = server.Close(context.Background()) })

	client := NewClient(socketPath)

	status, err := client.Status(context.Background())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.SnapshotVersion != "v1" || status.ActiveRoutes != 1 {
		t.Fatalf("unexpected status payload: %+v", status)
	}

	routes, err := client.Routes(context.Background())
	if err != nil {
		t.Fatalf("routes: %v", err)
	}
	if len(routes) != 1 || routes[0].Hostname != "api.acme.test" {
		t.Fatalf("unexpected routes payload: %+v", routes)
	}

	logs, err := client.Logs(context.Background())
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	if len(logs) != 1 || logs[0].Hostname != "api.acme.test" {
		t.Fatalf("unexpected logs payload: %+v", logs)
	}
}

func TestClientRefreshReturnsErrorOnDaemonFailure(t *testing.T) {
	server, socketPath := mustStartTestServer(t, StateSnapshot{})
	t.Cleanup(func() { _ = server.Close(context.Background()) })

	server.SetRefreshFunc(func(context.Context, string) error {
		return errors.New("docker ping failed")
	})

	client := NewClient(socketPath)
	_, err := client.Refresh(context.Background(), "operator")
	if err == nil {
		t.Fatal("expected refresh to return an error")
	}
	if !strings.Contains(err.Error(), "/refresh") || !strings.Contains(err.Error(), "docker ping failed") {
		t.Fatalf("expected explicit refresh failure, got %v", err)
	}
}

func TestClientReturnsExplicitSocketAndMalformedResponseErrors(t *testing.T) {
	missing := NewClient(filepath.Join(t.TempDir(), "missing.sock"))
	_, err := missing.Status(context.Background())
	if err == nil || !strings.Contains(err.Error(), "connect admin socket") {
		t.Fatalf("expected explicit socket error, got %v", err)
	}

	socketPath := filepath.Join(t.TempDir(), "bad.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen bad socket: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		_ = drainHTTPHeaders(conn)
		_, _ = conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 8\r\n\r\nnot-json"))
	}()

	bad := NewClient(socketPath)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = bad.Status(ctx)
	if err == nil || !strings.Contains(err.Error(), "decode /status response") {
		t.Fatalf("expected explicit decode error, got %v", err)
	}
}

func TestClientReturnsErrorOnMalformedRefreshResponse(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "bad-refresh.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen bad socket: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		_ = drainHTTPHeaders(conn)
		_, _ = conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 8\r\n\r\nnot-json"))
	}()

	client := NewClient(socketPath)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = client.Refresh(ctx, "operator")
	if err == nil || !strings.Contains(err.Error(), "decode /refresh response") {
		t.Fatalf("expected explicit decode error, got %v", err)
	}
}

func TestClientUsesPostForRefresh(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "method.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen socket: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	gotMethod := make(chan string, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		client := &http.Client{Transport: &http.Transport{}}
		_ = client
		buf := make([]byte, 256)
		n, _ := conn.Read(buf)
		line := string(buf[:n])
		if strings.HasPrefix(line, "POST /refresh") {
			gotMethod <- "POST"
		} else {
			gotMethod <- line
		}
		_, _ = conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 61\r\n\r\n{\"accepted\":true,\"refreshed\":true,\"at\":\"2026-01-01T00:00:00Z\"}"))
	}()

	client := NewClient(socketPath)
	_, _ = client.Refresh(context.Background(), "operator")
	if got := <-gotMethod; got != "POST" {
		t.Fatalf("expected POST /refresh, got %q", got)
	}
}

func TestClientSupportsRoutingPauseResumeAndStartupEndpoints_D01_D02_D03(t *testing.T) {
	server, socketPath := mustStartTestServer(t, StateSnapshot{})
	t.Cleanup(func() { _ = server.Close(context.Background()) })

	client := NewClient(socketPath)

	pause, err := client.PauseRouting(context.Background())
	if err != nil {
		t.Fatalf("pause routing: %v", err)
	}
	if !pause.Paused {
		t.Fatalf("expected paused=true payload, got %+v", pause)
	}

	resume, err := client.ResumeRouting(context.Background())
	if err != nil {
		t.Fatalf("resume routing: %v", err)
	}
	if resume.Paused {
		t.Fatalf("expected paused=false payload, got %+v", resume)
	}

	startup, err := client.StartupStatus(context.Background())
	if err != nil {
		t.Fatalf("startup status: %v", err)
	}
	if len(startup.Roles) < 2 {
		t.Fatalf("expected daemon and menubar roles, got %+v", startup)
	}

	toggle, err := client.SetStartupEnabled(context.Background(), StartupToggleRequest{Role: "menubar", Enabled: true})
	if err != nil {
		t.Fatalf("set startup enabled: %v", err)
	}
	if toggle.Role != "menubar" || !toggle.Enabled {
		t.Fatalf("unexpected startup toggle response: %+v", toggle)
	}
}

func TestClientStartupStatusReturnsErrorOnNonSuccessResponse(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "startup-error.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen startup socket: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		_ = drainHTTPHeaders(conn)
		body := `{"error":"launchctl unavailable"}`
		_, _ = conn.Write([]byte(fmt.Sprintf("HTTP/1.1 503 Service Unavailable\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s", len(body), body)))
	}()

	client := NewClient(socketPath)
	_, err = client.StartupStatus(context.Background())
	if err == nil {
		t.Fatal("expected startup status to return an error")
	}
	if !strings.Contains(err.Error(), "/startup") || !strings.Contains(err.Error(), "launchctl unavailable") {
		t.Fatalf("expected explicit startup failure, got %v", err)
	}
}

func drainHTTPHeaders(conn net.Conn) error {
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		if line == "\r\n" {
			return nil
		}
	}
}
