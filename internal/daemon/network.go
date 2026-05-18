package daemon

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"

	mdns "github.com/miekg/dns"
	"github.com/mochaka/devproxy/internal/certs"
	devproxydns "github.com/mochaka/devproxy/internal/dns"
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
	ManagedSuffix        string
	Snapshot             func() routing.Snapshot
	RoutingPaused        func() bool
	Certificates         map[string]tls.Certificate
	StoredCertificates   map[string]certs.StoredCertificate
	CertificateOutputDir string
	IssueCertificate     func(outputDir string, sans []string) (certs.IssuedCertificate, error)
	DNSAddress           string
	HTTPAddress          string
	HTTPSAddress         string
}

type NetworkRuntime struct {
	mu            sync.RWMutex
	managedSuffix string
	readSnapshot  func() routing.Snapshot
	readPaused    func() bool
	certificates  map[string]tls.Certificate
	storedCerts   map[string]certs.StoredCertificate
	certOutputDir string
	issueCert     func(outputDir string, sans []string) (certs.IssuedCertificate, error)
	httpHandler   *proxy.HTTPHandler
	httpsHandler  *proxy.HTTPSListener
	health        NetworkRuntimeHealth
	dnsServer     *mdns.Server
	dnsPacketConn net.PacketConn
	httpServer    *http.Server
	httpsServer   *http.Server
	httpListener  net.Listener
	httpsListener net.Listener
}

func NewNetworkRuntime(cfg NetworkRuntimeConfig) (*NetworkRuntime, error) {
	snapshot := cfg.Snapshot
	if snapshot == nil {
		snapshot = func() routing.Snapshot { return routing.Snapshot{Routes: map[string]routing.Route{}} }
	}
	prepared, err := prepareStoredCertificates(snapshot(), cfg)
	if err != nil {
		return nil, err
	}
	httpHandler := proxy.NewHTTPHandler(proxy.HTTPHandlerConfig{ManagedSuffix: cfg.ManagedSuffix, Snapshot: snapshot, RoutingPaused: cfg.RoutingPaused})
	httpsHandler, err := proxy.NewHTTPSListener(proxy.HTTPSListenerConfig{ManagedSuffix: cfg.ManagedSuffix, Snapshot: snapshot, RoutingPaused: cfg.RoutingPaused, Certificates: cfg.Certificates, Stored: prepared})
	if err != nil {
		return nil, err
	}
	paused := cfg.RoutingPaused
	if paused == nil {
		paused = func() bool { return false }
	}
	httpAddress := cfg.HTTPAddress
	if httpAddress == "" {
		httpAddress = "127.0.0.1:80"
	}
	httpsAddress := cfg.HTTPSAddress
	if httpsAddress == "" {
		httpsAddress = "127.0.0.1:443"
	}
	dnsAddress := cfg.DNSAddress
	if dnsAddress == "" {
		dnsAddress = "127.0.0.1:53535"
	}
	runtime := &NetworkRuntime{
		managedSuffix: cfg.ManagedSuffix,
		readSnapshot:  snapshot,
		readPaused:    paused,
		certificates:  cloneTLSCertificates(cfg.Certificates),
		storedCerts:   indexStoredCertificates(prepared),
		certOutputDir: certificateOutputDir(cfg.CertificateOutputDir),
		issueCert:     certificateIssuer(cfg.IssueCertificate),
		httpHandler:   httpHandler,
		httpsHandler:  httpsHandler,
	}
	runtime.health = NetworkRuntimeHealth{
		DNS:              ListenerHealth{Enabled: true, Bound: false, BindAddress: dnsAddress},
		HTTP:             ListenerHealth{Enabled: true, Bound: false, BindAddress: httpAddress},
		HTTPS:            ListenerHealth{Enabled: true, Bound: false, BindAddress: httpsAddress},
		ManagedSuffix:    cfg.ManagedSuffix,
		CertificateReady: len(cfg.Certificates) > 0 || len(prepared) > 0,
	}
	return runtime, nil
}

func (n *NetworkRuntime) RefreshCertificates() error {
	prepared, err := prepareStoredCertificates(n.readSnapshot(), NetworkRuntimeConfig{
		ManagedSuffix:        n.managedSuffix,
		Certificates:         n.certificates,
		StoredCertificates:   cloneStoredCertificates(n.storedCerts),
		CertificateOutputDir: n.certOutputDir,
		IssueCertificate:     n.issueCert,
	})
	if err != nil {
		n.SetCertificateReady(false)
		return err
	}
	if err := n.httpsHandler.ReplaceCertificates(n.certificates, prepared); err != nil {
		n.SetCertificateReady(false)
		return err
	}

	n.mu.Lock()
	n.storedCerts = indexStoredCertificates(prepared)
	n.health.CertificateReady = len(n.certificates) > 0 || len(prepared) > 0
	n.mu.Unlock()
	return nil
}

func (n *NetworkRuntime) HTTPHandler() *proxy.HTTPHandler    { return n.httpHandler }
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

func (n *NetworkRuntime) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.dnsServer != nil || n.httpServer != nil || n.httpsServer != nil {
		return fmt.Errorf("network runtime already started")
	}

	dnsPacketConn, err := net.ListenPacket("udp", n.health.DNS.BindAddress)
	n.health.DNS.Bound = err == nil
	n.health.DNS.LastError = errorString(err)
	if err != nil {
		return err
	}
	n.health.DNS.BindAddress = dnsPacketConn.LocalAddr().String()
	dnsServer := &mdns.Server{PacketConn: dnsPacketConn, Handler: devproxydns.NewServer(n.managedSuffix, n.readSnapshot)}
	n.dnsPacketConn = dnsPacketConn
	n.dnsServer = dnsServer
	go serveDNSServer(dnsServer)

	httpListener, err := net.Listen("tcp", n.health.HTTP.BindAddress)
	n.health.HTTP.Bound = err == nil
	n.health.HTTP.LastError = errorString(err)
	if err == nil {
		n.health.HTTP.BindAddress = httpListener.Addr().String()
		n.httpListener = httpListener
		n.httpServer = &http.Server{Handler: n.httpHandler}
		go serveListener(n.httpServer, httpListener)
	}

	httpsListener, err := tls.Listen("tcp", n.health.HTTPS.BindAddress, n.httpsHandler.TLSConfig())
	n.health.HTTPS.Bound = err == nil
	n.health.HTTPS.LastError = errorString(err)
	if err == nil {
		n.health.HTTPS.BindAddress = httpsListener.Addr().String()
		n.httpsListener = httpsListener
		n.httpsServer = &http.Server{Handler: n.httpsHandler}
		go serveListener(n.httpsServer, httpsListener)
	}
	return nil
}

func (n *NetworkRuntime) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	var errs []error
	if err := n.stopDNSServerLocked(); err != nil {
		errs = append(errs, err)
	}
	if n.httpServer != nil {
		err := n.httpServer.Shutdown(context.Background())
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errs = append(errs, err)
		}
		n.httpServer = nil
		n.httpListener = nil
		n.health.HTTP.Bound = false
		n.health.HTTP.LastError = ""
	}
	if n.httpsServer != nil {
		err := n.httpsServer.Shutdown(context.Background())
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errs = append(errs, err)
		}
		n.httpsServer = nil
		n.httpsListener = nil
		n.health.HTTPS.Bound = false
		n.health.HTTPS.LastError = ""
	}
	return errors.Join(errs...)
}

func (n *NetworkRuntime) stopDNSServerLocked() error {
	if n.dnsServer == nil {
		n.health.DNS.Bound = false
		n.health.DNS.LastError = ""
		return nil
	}
	err := n.dnsServer.Shutdown()
	if err != nil && err.Error() == "dns: server not started" {
		err = nil
	}
	n.dnsServer = nil
	n.dnsPacketConn = nil
	n.health.DNS.Bound = false
	n.health.DNS.LastError = ""
	return err
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

func cloneTLSCertificates(in map[string]tls.Certificate) map[string]tls.Certificate {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]tls.Certificate, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneStoredCertificates(in map[string]certs.StoredCertificate) map[string]certs.StoredCertificate {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]certs.StoredCertificate, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func indexStoredCertificates(stored []certs.StoredCertificate) map[string]certs.StoredCertificate {
	if len(stored) == 0 {
		return nil
	}
	indexed := make(map[string]certs.StoredCertificate, len(stored))
	for _, cert := range stored {
		if cert.ProjectRoot == "" {
			continue
		}
		indexed[cert.ProjectRoot] = cert
	}
	if len(indexed) == 0 {
		return nil
	}
	return indexed
}

func certificateOutputDir(dir string) string {
	if dir == "" {
		return "."
	}
	return dir
}

func certificateIssuer(issue func(outputDir string, sans []string) (certs.IssuedCertificate, error)) func(outputDir string, sans []string) (certs.IssuedCertificate, error) {
	if issue == nil {
		return certs.MKCertIssuer{}.Issue
	}
	return issue
}

func prepareStoredCertificates(snapshot routing.Snapshot, cfg NetworkRuntimeConfig) ([]certs.StoredCertificate, error) {
	issuer := cfg.IssueCertificate
	if issuer == nil {
		issuer = certs.MKCertIssuer{}.Issue
	}
	existing := cfg.StoredCertificates
	if existing == nil {
		existing = map[string]certs.StoredCertificate{}
	}
	outputDir := cfg.CertificateOutputDir
	if outputDir == "" {
		outputDir = "."
	}

	decisions := certs.BuildCertificateInventory(snapshot, cfg.ManagedSuffix, existing)
	prepared := make([]certs.StoredCertificate, 0, len(decisions))
	var certErrors []string
	for _, decision := range decisions {
		if decision.ReuseExisting {
			prepared = append(prepared, existing[decision.ProjectRoot])
			continue
		}
		issued, err := issuer(outputDir, decision.RequiredSANs)
		if err != nil {
			certErrors = append(certErrors, fmt.Sprintf("%s: %v", decision.ProjectRoot, err))
			continue
		}
		projectRoot := issued.ProjectRoot
		if projectRoot == "" {
			projectRoot = decision.ProjectRoot
		}
		prepared = append(prepared, certs.StoredCertificate{
			ProjectRoot: projectRoot,
			SANs:        append([]string(nil), issued.SANs...),
			CertPath:    issued.CertPath,
			KeyPath:     issued.KeyPath,
		})
	}
	if len(certErrors) > 0 {
		return prepared, fmt.Errorf("certificate issuance skipped for %d project(s): %s", len(certErrors), certErrors[0])
	}
	return prepared, nil
}

func serveListener(server *http.Server, listener net.Listener) {
	if server == nil || listener == nil {
		return
	}
	_ = server.Serve(listener)
}

func serveDNSServer(server *mdns.Server) {
	if server == nil {
		return
	}
	_ = server.ActivateAndServe()
}
