package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mochaka/devproxy/internal/certs"
	"github.com/mochaka/devproxy/internal/routing"
)

func TestHTTPSListenerBuildsTLSConfigFromPreparedInventory(t *testing.T) {
	leaf := mustTestCertificate(t, []string{"acme.test", "*.acme.test"})

	l, err := NewHTTPSListener(HTTPSListenerConfig{
		ManagedSuffix: "test",
		Snapshot: staticSnapshot(routing.Snapshot{Routes: map[string]routing.Route{
			"api.acme.test": {Hostname: "api.acme.test", Upstream: routing.Upstream{Host: "127.0.0.1", Port: 8080, Scheme: "http"}},
		}}),
		Certificates: map[string]tls.Certificate{"acme.test": leaf},
	})
	if err != nil {
		t.Fatalf("new https listener: %v", err)
	}

	if l.TLSConfig() == nil {
		t.Fatalf("expected tls config")
	}
	if l.TLSConfig().GetCertificate == nil {
		t.Fatalf("expected tls config with certificate selector")
	}
}

func TestHTTPSListenerSelectsCertificateForManagedActiveRoute(t *testing.T) {
	leaf := mustTestCertificate(t, []string{"acme.test", "*.acme.test"})

	l, err := NewHTTPSListener(HTTPSListenerConfig{
		ManagedSuffix: "test",
		Snapshot: staticSnapshot(routing.Snapshot{Routes: map[string]routing.Route{
			"api.acme.test": {Hostname: "api.acme.test", Upstream: routing.Upstream{Host: "127.0.0.1", Port: 8080, Scheme: "http"}},
		}}),
		Certificates: map[string]tls.Certificate{"acme.test": leaf},
	})
	if err != nil {
		t.Fatalf("new https listener: %v", err)
	}

	cert, err := l.TLSConfig().GetCertificate(&tls.ClientHelloInfo{ServerName: "api.acme.test"})
	if err != nil {
		t.Fatalf("get certificate: %v", err)
	}
	if cert == nil {
		t.Fatalf("expected certificate for managed active route")
	}
}

func TestHTTPSListenerSelectsCertificateForManagedNoRouteHost(t *testing.T) {
	leaf := mustTestCertificate(t, []string{"acme.test", "*.acme.test"})

	l, err := NewHTTPSListener(HTTPSListenerConfig{
		ManagedSuffix: "test",
		Snapshot:      staticSnapshot(routing.Snapshot{Routes: map[string]routing.Route{}}),
		Certificates:  map[string]tls.Certificate{"acme.test": leaf},
	})
	if err != nil {
		t.Fatalf("new https listener: %v", err)
	}

	cert, err := l.TLSConfig().GetCertificate(&tls.ClientHelloInfo{ServerName: "missing.acme.test"})
	if err != nil {
		t.Fatalf("expected certificate selection for managed no-route host: %v", err)
	}
	if cert == nil {
		t.Fatalf("expected certificate for managed no-route host")
	}

	req := httptest.NewRequest(http.MethodGet, "https://missing.acme.test/", nil)
	req.Host = "missing.acme.test"
	rr := httptest.NewRecorder()

	claimed := l.HandleHTTPS(rr, req)
	if !claimed {
		t.Fatalf("expected managed host to be claimed")
	}
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for managed no-route, got %d", rr.Code)
	}
}

func TestHTTPSListenerSharesFriendlyNoRouteAndPausedBehavior(t *testing.T) {
	t.Run("no route", func(t *testing.T) {
		l, err := NewHTTPSListener(HTTPSListenerConfig{
			ManagedSuffix: "test",
			Snapshot:      staticSnapshot(routing.Snapshot{Routes: map[string]routing.Route{}}),
			Certificates:  map[string]tls.Certificate{"acme.test": mustTestCertificate(t, []string{"acme.test", "*.acme.test"})},
		})
		if err != nil {
			t.Fatalf("new https listener: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "https://missing.acme.test/", nil)
		req.Host = "missing.acme.test"
		rr := httptest.NewRecorder()

		claimed := l.HandleHTTPS(rr, req)
		if !claimed {
			t.Fatalf("expected managed host to be claimed")
		}
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for managed no-route, got %d", rr.Code)
		}
	})

	t.Run("paused", func(t *testing.T) {
		l, err := NewHTTPSListener(HTTPSListenerConfig{
			ManagedSuffix: "test",
			Snapshot: staticSnapshot(routing.Snapshot{Routes: map[string]routing.Route{
				"api.acme.test": {Hostname: "api.acme.test", Upstream: routing.Upstream{Host: "127.0.0.1", Port: 8080, Scheme: "http"}},
			}}),
			RoutingPaused: func() bool { return true },
			Certificates:  map[string]tls.Certificate{"acme.test": mustTestCertificate(t, []string{"acme.test", "*.acme.test"})},
		})
		if err != nil {
			t.Fatalf("new https listener: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "https://api.acme.test/", nil)
		req.Host = "api.acme.test"
		rr := httptest.NewRecorder()

		claimed := l.HandleHTTPS(rr, req)
		if !claimed {
			t.Fatalf("expected managed host to be claimed")
		}
		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503 for paused routing, got %d", rr.Code)
		}
	})
}

func TestHTTPSListenerSelectsStoredWildcardCertificateForSubdomain(t *testing.T) {
	certPath, keyPath := mustWriteTestCertificateFiles(t, []string{"acme.test", "*.acme.test"})

	l, err := NewHTTPSListener(HTTPSListenerConfig{
		ManagedSuffix: "test",
		Snapshot: staticSnapshot(routing.Snapshot{Routes: map[string]routing.Route{
			"mailpit.acme.test": {Hostname: "mailpit.acme.test", Upstream: routing.Upstream{Host: "127.0.0.1", Port: 8025, Scheme: "http"}},
		}}),
		Stored: []certs.StoredCertificate{{ProjectRoot: "acme.test", SANs: []string{"acme.test", "*.acme.test"}, CertPath: certPath, KeyPath: keyPath}},
	})
	if err != nil {
		t.Fatalf("new https listener: %v", err)
	}

	cert, err := l.TLSConfig().GetCertificate(&tls.ClientHelloInfo{ServerName: "mailpit.acme.test"})
	if err != nil {
		t.Fatalf("expected stored wildcard certificate selection: %v", err)
	}
	if cert == nil {
		t.Fatalf("expected certificate for stored wildcard subdomain")
	}
}

func mustTestCertificate(t *testing.T, dnsNames []string) tls.Certificate {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate private key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: dnsNames[0]},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     dnsNames,
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: tmpl}
}

func mustWriteTestCertificateFiles(t *testing.T, dnsNames []string) (string, string) {
	t.Helper()

	cert := mustTestCertificate(t, dnsNames)
	key, ok := cert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		t.Fatalf("expected rsa private key, got %T", cert.PrivateKey)
	}

	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatalf("write cert pem: %v", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatalf("write key pem: %v", err)
	}

	return certPath, keyPath
}
