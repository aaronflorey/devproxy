package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/mochaka/devproxy/internal/dns"
	"github.com/mochaka/devproxy/internal/routing"
)

type HTTPHandlerConfig struct {
	ManagedSuffix string
	Snapshot      func() routing.Snapshot
	RoutingPaused func() bool
	Transport     http.RoundTripper
}

type HTTPHandler struct {
	dnsLookup     *dns.Server
	readSnapshot  func() routing.Snapshot
	routingPaused func() bool
	transport     http.RoundTripper
}

func NewHTTPHandler(cfg HTTPHandlerConfig) *HTTPHandler {
	readSnapshot := cfg.Snapshot
	if readSnapshot == nil {
		readSnapshot = func() routing.Snapshot { return routing.Snapshot{Routes: map[string]routing.Route{}} }
	}
	routingPaused := cfg.RoutingPaused
	if routingPaused == nil {
		routingPaused = func() bool { return false }
	}
	transport := cfg.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	return &HTTPHandler{
		dnsLookup:     dns.NewServer(cfg.ManagedSuffix, readSnapshot),
		readSnapshot:  readSnapshot,
		routingPaused: routingPaused,
		transport:     transport,
	}
}

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.HandleHTTP(w, r) {
		return
	}
	http.NotFound(w, r)
}

func (h *HTTPHandler) HandleHTTP(w http.ResponseWriter, r *http.Request) bool {
	host := canonicalHost(r.Host)
	lookup := h.dnsLookup.LookupHostname(host)
	if !lookup.Managed {
		return false
	}

	if h.routingPaused() {
		writeFriendlyPaused(w, host)
		return true
	}

	if !lookup.ActiveRoute {
		writeFriendlyNoRoute(w, host)
		return true
	}

	target := upstreamURL(lookup.Route.Upstream)
	if target == nil {
		writeFriendlyNoRoute(w, host)
		return true
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = h.transport
	proxy.ServeHTTP(w, r)
	return true
}

func upstreamURL(upstream routing.Upstream) *url.URL {
	scheme := strings.TrimSpace(upstream.Scheme)
	if scheme == "" {
		scheme = "http"
	}
	host := strings.TrimSpace(upstream.Host)
	if host == "" || upstream.Port <= 0 {
		return nil
	}
	return &url.URL{Scheme: scheme, Host: fmt.Sprintf("%s:%d", host, upstream.Port)}
}

func canonicalHost(hostPort string) string {
	host := strings.TrimSpace(hostPort)
	if i := strings.Index(host, ":"); i >= 0 {
		host = host[:i]
	}
	return strings.Trim(strings.ToLower(host), ".")
}

func writeFriendlyNoRoute(w http.ResponseWriter, host string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(fmt.Sprintf("No route is active for %s in devproxy.", host)))
}

func writeFriendlyPaused(w http.ResponseWriter, host string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(fmt.Sprintf("Routing is paused for %s in devproxy.", host)))
}
