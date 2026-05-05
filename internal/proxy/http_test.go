package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mochaka/devproxy/internal/routing"
)

func TestHTTPProxyActiveRouteUsesReconciledUpstreamAndScheme(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hello" {
			t.Fatalf("expected upstream path /hello, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("proxied-ok"))
	}))
	defer upstream.Close()

	host, port := splitHostPort(t, upstream.URL)
	h := NewHTTPHandler(HTTPHandlerConfig{
		ManagedSuffix: "test",
		Snapshot: staticSnapshot(routing.Snapshot{Routes: map[string]routing.Route{
			"api.acme.test": {
				Hostname: "api.acme.test",
				Upstream: routing.Upstream{Host: host, Port: port, Scheme: "http"},
			},
		}}),
		RoutingPaused: func() bool { return false },
	})

	req := httptest.NewRequest(http.MethodGet, "http://api.acme.test/hello", nil)
	req.Host = "api.acme.test"
	rr := httptest.NewRecorder()

	claimed := h.HandleHTTP(rr, req)
	if !claimed {
		t.Fatalf("expected managed active route to be claimed by proxy")
	}

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if got := strings.TrimSpace(rr.Body.String()); got != "proxied-ok" {
		t.Fatalf("expected proxied response body, got %q", got)
	}
}

func TestNoRouteManagedHostReturnsFriendlyResponse(t *testing.T) {
	h := NewHTTPHandler(HTTPHandlerConfig{
		ManagedSuffix: "test",
		Snapshot:      staticSnapshot(routing.Snapshot{Routes: map[string]routing.Route{}}),
		RoutingPaused: func() bool { return false },
	})

	req := httptest.NewRequest(http.MethodGet, "http://missing.acme.test/", nil)
	req.Host = "missing.acme.test"
	rr := httptest.NewRecorder()

	claimed := h.HandleHTTP(rr, req)
	if !claimed {
		t.Fatalf("expected managed host with missing route to be claimed")
	}
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for managed no-route, got %d", rr.Code)
	}
	if body := rr.Body.String(); !strings.Contains(strings.ToLower(body), "no route") {
		t.Fatalf("expected friendly no-route response, got %q", body)
	}
}

func TestPausedManagedHostReturnsDistinctFriendlyResponse(t *testing.T) {
	h := NewHTTPHandler(HTTPHandlerConfig{
		ManagedSuffix: "test",
		Snapshot: staticSnapshot(routing.Snapshot{Routes: map[string]routing.Route{
			"api.acme.test": {Hostname: "api.acme.test"},
		}}),
		RoutingPaused: func() bool { return true },
	})

	req := httptest.NewRequest(http.MethodGet, "http://api.acme.test/", nil)
	req.Host = "api.acme.test"
	rr := httptest.NewRecorder()

	claimed := h.HandleHTTP(rr, req)
	if !claimed {
		t.Fatalf("expected paused managed host to be claimed")
	}
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for paused routing, got %d", rr.Code)
	}
	if body := strings.ToLower(rr.Body.String()); !strings.Contains(body, "paused") {
		t.Fatalf("expected paused response to mention paused state, got %q", rr.Body.String())
	}
}

func TestNoRouteUnmanagedHostBypassesFriendlyHandler(t *testing.T) {
	h := NewHTTPHandler(HTTPHandlerConfig{
		ManagedSuffix: "test",
		Snapshot:      staticSnapshot(routing.Snapshot{Routes: map[string]routing.Route{}}),
		RoutingPaused: func() bool { return false },
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Host = "example.com"
	rr := httptest.NewRecorder()

	claimed := h.HandleHTTP(rr, req)
	if claimed {
		t.Fatalf("expected unmanaged host not to be claimed")
	}
	if rr.Code != 200 || rr.Body.Len() != 0 {
		t.Fatalf("expected untouched response writer for unmanaged host, got status=%d body=%q", rr.Code, rr.Body.String())
	}
}

func TestWebSocketProxyPreservesUpgradeCapableBehavior(t *testing.T) {
	transport := &capturingTransport{}
	h := NewHTTPHandler(HTTPHandlerConfig{
		ManagedSuffix: "test",
		Snapshot: staticSnapshot(routing.Snapshot{Routes: map[string]routing.Route{
			"ws.acme.test": {
				Hostname: "ws.acme.test",
				Upstream: routing.Upstream{Host: "upstream.local", Port: 7443, Scheme: "https"},
			},
		}}),
		RoutingPaused: func() bool { return false },
		Transport:     transport,
	})

	req := httptest.NewRequest(http.MethodGet, "http://ws.acme.test/socket", nil)
	req.Host = "ws.acme.test"
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	rr := httptest.NewRecorder()

	claimed := h.HandleHTTP(rr, req)
	if !claimed {
		t.Fatalf("expected websocket host to be claimed")
	}

	if transport.gotScheme != "https" {
		t.Fatalf("expected proxy roundtrip to preserve configured upstream scheme, got %q", transport.gotScheme)
	}
	if transport.gotHost != "upstream.local:7443" {
		t.Fatalf("expected upstream host:port from route metadata, got %q", transport.gotHost)
	}
	if got := strings.ToLower(transport.gotUpgrade); got != "websocket" {
		t.Fatalf("expected websocket upgrade header to survive proxy setup, got %q", transport.gotUpgrade)
	}
}

type capturingTransport struct {
	gotScheme  string
	gotHost    string
	gotUpgrade string
}

func (c *capturingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.gotScheme = req.URL.Scheme
	c.gotHost = req.URL.Host
	c.gotUpgrade = req.Header.Get("Upgrade")
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}, nil
}

func staticSnapshot(s routing.Snapshot) func() routing.Snapshot {
	return func() routing.Snapshot { return s }
}

func splitHostPort(t *testing.T, url string) (string, int) {
	t.Helper()
	trimmed := strings.TrimPrefix(url, "http://")
	parts := strings.Split(trimmed, ":")
	if len(parts) != 2 {
		t.Fatalf("unexpected test server url format: %q", url)
	}
	port := 0
	_, err := fmt.Sscanf(parts[1], "%d", &port)
	if err != nil {
		t.Fatalf("parse upstream port: %v", err)
	}
	return parts[0], port
}
