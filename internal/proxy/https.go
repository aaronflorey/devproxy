package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/mochaka/devproxy/internal/certs"
	"github.com/mochaka/devproxy/internal/dns"
	"github.com/mochaka/devproxy/internal/routing"
)

type HTTPSListenerConfig struct {
	ManagedSuffix string
	Snapshot      func() routing.Snapshot
	RoutingPaused func() bool
	Transport     http.RoundTripper
	Certificates  map[string]tls.Certificate
	Stored        []certs.StoredCertificate
}

type HTTPSListener struct {
	mu           sync.RWMutex
	handler      *HTTPHandler
	dnsLookup    *dns.Server
	tlsConfig    *tls.Config
	certificates map[string]tls.Certificate
}

func NewHTTPSListener(cfg HTTPSListenerConfig) (*HTTPSListener, error) {
	h := NewHTTPHandler(HTTPHandlerConfig{ManagedSuffix: cfg.ManagedSuffix, Snapshot: cfg.Snapshot, RoutingPaused: cfg.RoutingPaused, Transport: cfg.Transport})
	lookup := dns.NewServer(cfg.ManagedSuffix, cfg.Snapshot)

	inventory, err := buildCertificateMap(cfg.Certificates, cfg.Stored)
	if err != nil {
		return nil, err
	}

	l := &HTTPSListener{handler: h, dnsLookup: lookup, certificates: inventory}
	l.tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12, GetCertificate: l.getCertificate}
	return l, nil
}

func (l *HTTPSListener) ReplaceCertificates(certificates map[string]tls.Certificate, stored []certs.StoredCertificate) error {
	inventory, err := buildCertificateMap(certificates, stored)
	if err != nil {
		return err
	}

	l.mu.Lock()
	l.certificates = inventory
	l.mu.Unlock()
	return nil
}

func (l *HTTPSListener) HandleHTTPS(w http.ResponseWriter, r *http.Request) bool {
	return l.handler.HandleHTTP(w, r)
}

func (l *HTTPSListener) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l.handler.ServeHTTP(w, r)
}

func (l *HTTPSListener) TLSConfig() *tls.Config {
	return l.tlsConfig
}

func (l *HTTPSListener) getCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if hello == nil {
		return nil, fmt.Errorf("tls handshake missing client hello")
	}
	host := normalizeHostname(hello.ServerName)
	lookup := l.dnsLookup.LookupHostname(host)
	if !lookup.Managed {
		return nil, fmt.Errorf("no managed route for %s", host)
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	if cert, ok := l.certificates[host]; ok {
		return &cert, nil
	}
	for san, cert := range l.certificates {
		if san == host {
			return &cert, nil
		}
		if strings.HasPrefix(san, "*.") && wildcardMatches(host, strings.TrimPrefix(san, "*.")) {
			return &cert, nil
		}
		if certificateCoversHost(&cert, host) {
			return &cert, nil
		}
	}

	return nil, fmt.Errorf("no certificate available for %s", host)
}

func buildCertificateMap(certificates map[string]tls.Certificate, stored []certs.StoredCertificate) (map[string]tls.Certificate, error) {
	inventory := map[string]tls.Certificate{}
	for k, v := range certificates {
		inventory[normalizeHostname(k)] = ensureParsedLeaf(v)
	}

	if len(inventory) > 0 || len(stored) == 0 {
		return inventory, nil
	}

	for _, storedCert := range stored {
		if storedCert.ProjectRoot == "" {
			continue
		}
		loaded, err := tls.LoadX509KeyPair(storedCert.CertPath, storedCert.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("load certificate %s: %w", storedCert.ProjectRoot, err)
		}
		inventory[normalizeHostname(storedCert.ProjectRoot)] = ensureParsedLeaf(loaded)
	}

	return inventory, nil
}

func certificateCoversHost(cert *tls.Certificate, host string) bool {
	if cert == nil {
		return false
	}
	if cert.Leaf != nil {
		for _, san := range cert.Leaf.DNSNames {
			n := normalizeHostname(san)
			if n == host {
				return true
			}
			if strings.HasPrefix(n, "*.") && wildcardMatches(host, strings.TrimPrefix(n, "*.")) {
				return true
			}
		}
	}
	return false
}

func normalizeHostname(host string) string {
	return strings.Trim(strings.ToLower(strings.TrimSpace(host)), ".")
}

func wildcardMatches(host, root string) bool {
	if !strings.HasSuffix(host, "."+root) {
		return false
	}
	hostLabels := strings.Split(host, ".")
	rootLabels := strings.Split(root, ".")
	return len(hostLabels) == len(rootLabels)+1
}

func ensureParsedLeaf(cert tls.Certificate) tls.Certificate {
	if cert.Leaf != nil || len(cert.Certificate) == 0 {
		return cert
	}

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return cert
	}
	cert.Leaf = leaf
	return cert
}
