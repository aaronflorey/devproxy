package daemon

import (
	"crypto/tls"
	"sync"

	"github.com/mochaka/devproxy/internal/proxy"
	"github.com/mochaka/devproxy/internal/routing"
)

type ListenerHealth struct {
	Enabled     bool
	Bound       bool
	BindAddress string
	LastError   string
}

type NetworkRuntimeHealth struct {
	DNS              ListenerHealth
	HTTP             ListenerHealth
	HTTPS            ListenerHealth
	Paused           bool
	CertificateReady bool
	ManagedSuffix    string
}

type NetworkRuntimeConfig struct {
	ManagedSuffix string
	Snapshot      func() routing.Snapshot
	RoutingPaused func() bool
	Certificates  map[string]tls.Certificate
}

type NetworkRuntime struct {
	mu           sync.RWMutex
	managedSuffix string
	readPaused   func() bool
	httpHandler  *proxy.HTTPHandler
	httpsHandler *proxy.HTTPSListener
	health       NetworkRuntimeHealth
}

func NewNetworkRuntime(cfg NetworkRuntimeConfig) (*NetworkRuntime, error) {
	httpHandler := proxy.NewHTTPHandler(proxy.HTTPHandlerConfig{ManagedSuffix: cfg.ManagedSuffix, Snapshot: cfg.Snapshot, RoutingPaused: cfg.RoutingPaused})
	httpsHandler, err := proxy.NewHTTPSListener(proxy.HTTPSListenerConfig{ManagedSuffix: cfg.ManagedSuffix, Snapshot: cfg.Snapshot, RoutingPaused: cfg.RoutingPaused, Certificates: cfg.Certificates})
	if err != nil {
		return nil, err
	}
	paused := cfg.RoutingPaused
	if paused == nil {
		paused = func() bool { return false }
	}
	runtime := &NetworkRuntime{managedSuffix: cfg.ManagedSuffix, readPaused: paused, httpHandler: httpHandler, httpsHandler: httpsHandler}
	runtime.health = NetworkRuntimeHealth{
		DNS: ListenerHealth{Enabled: true, Bound: true, BindAddress: "127.0.0.1:53535"},
		HTTP: ListenerHealth{Enabled: true, Bound: false, BindAddress: "127.0.0.1:80"},
		HTTPS: ListenerHealth{Enabled: true, Bound: false, BindAddress: "127.0.0.1:443"},
		ManagedSuffix: cfg.ManagedSuffix,
		CertificateReady: len(cfg.Certificates) > 0,
	}
	return runtime, nil
}

func (n *NetworkRuntime) HTTPHandler() *proxy.HTTPHandler { return n.httpHandler }
func (n *NetworkRuntime) HTTPSHandler() *proxy.HTTPSListener { return n.httpsHandler }

func (n *NetworkRuntime) SetDNSBindResult(addr string, err error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.health.DNS.BindAddress = addr
	n.health.DNS.Bound = err == nil
	n.health.DNS.LastError = errorString(err)
}

func (n *NetworkRuntime) SetHTTPBindResult(addr string, err error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.health.HTTP.BindAddress = addr
	n.health.HTTP.Bound = err == nil
	n.health.HTTP.LastError = errorString(err)
}

func (n *NetworkRuntime) SetHTTPSBindResult(addr string, err error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.health.HTTPS.BindAddress = addr
	n.health.HTTPS.Bound = err == nil
	n.health.HTTPS.LastError = errorString(err)
}

func (n *NetworkRuntime) SetCertificateReady(ready bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.health.CertificateReady = ready
}

func (n *NetworkRuntime) Health() NetworkRuntimeHealth {
	n.mu.RLock()
	defer n.mu.RUnlock()
	health := n.health
	health.Paused = n.readPaused()
	health.ManagedSuffix = n.managedSuffix
	return health
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
