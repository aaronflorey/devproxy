package daemon

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	mdns "github.com/miekg/dns"
	"github.com/mochaka/devproxy/internal/certs"
	"github.com/mochaka/devproxy/internal/routing"
)

func TestNewNetworkRuntimePreparesCertificates(t *testing.T) {
	t.Run("reuses existing inventory", func(t *testing.T) {
		certPath, keyPath := mustWriteTestCertificateFiles(t, []string{"acme.test", "*.acme.test"})
		runtime, err := NewNetworkRuntime(NetworkRuntimeConfig{
			ManagedSuffix: "test",
			Snapshot: staticDaemonSnapshot(routing.Snapshot{Routes: map[string]routing.Route{
				"api.acme.test": {Hostname: "api.acme.test", ServedHostnames: []string{"acme.test", "api.acme.test"}},
			}}),
			StoredCertificates: map[string]certs.StoredCertificate{
				"acme.test": {
					ProjectRoot: "acme.test",
					SANs:        []string{"acme.test", "*.acme.test"},
					CertPath:    certPath,
					KeyPath:     keyPath,
				},
			},
			IssueCertificate: func(string, []string) (certs.IssuedCertificate, error) {
				t.Fatal("did not expect issuance for reusable certificate")
				return certs.IssuedCertificate{}, nil
			},
		})
		if err != nil {
			t.Fatalf("new network runtime: %v", err)
		}
		if !runtime.Health().CertificateReady {
			t.Fatalf("expected certificate readiness from reused inventory")
		}
		if _, err := runtime.HTTPSHandler().TLSConfig().GetCertificate(&tls.ClientHelloInfo{ServerName: "api.acme.test"}); err != nil {
			t.Fatalf("expected reused certificate to be loaded: %v", err)
		}
	})

	t.Run("issues certificate when coverage changes", func(t *testing.T) {
		certPath, keyPath := mustWriteTestCertificateFiles(t, []string{"acme.test", "*.acme.test"})
		var gotOutputDir string
		var gotSANs []string
		runtime, err := NewNetworkRuntime(NetworkRuntimeConfig{
			ManagedSuffix:        "test",
			CertificateOutputDir: t.TempDir(),
			Snapshot: staticDaemonSnapshot(routing.Snapshot{Routes: map[string]routing.Route{
				"api.foo.acme.test": {Hostname: "api.foo.acme.test", ServedHostnames: []string{"foo.acme.test", "api.foo.acme.test"}},
			}}),
			IssueCertificate: func(outputDir string, sans []string) (certs.IssuedCertificate, error) {
				gotOutputDir = outputDir
				gotSANs = append([]string(nil), sans...)
				return certs.IssuedCertificate{
					ProjectRoot: "acme.test",
					SANs:        append([]string(nil), sans...),
					CertPath:    certPath,
					KeyPath:     keyPath,
				}, nil
			},
		})
		if err != nil {
			t.Fatalf("new network runtime: %v", err)
		}
		if gotOutputDir == "" {
			t.Fatalf("expected issuance to receive output dir")
		}
		wantSANs := []string{"acme.test", "api.foo.acme.test", "foo.acme.test"}
		if !reflect.DeepEqual(gotSANs, wantSANs) {
			t.Fatalf("unexpected issuance sans: got %v want %v", gotSANs, wantSANs)
		}
		if !runtime.Health().CertificateReady {
			t.Fatalf("expected certificate readiness from issued inventory")
		}
	})
}

func TestNetworkRuntimeCertificateReadyFromPreparedInventory(t *testing.T) {
	certPath, keyPath := mustWriteTestCertificateFiles(t, []string{"acme.test", "*.acme.test"})
	runtime, err := NewNetworkRuntime(NetworkRuntimeConfig{
		ManagedSuffix: "test",
		Snapshot: staticDaemonSnapshot(routing.Snapshot{Routes: map[string]routing.Route{
			"api.acme.test": {Hostname: "api.acme.test", ServedHostnames: []string{"acme.test", "api.acme.test"}},
		}}),
		StoredCertificates: map[string]certs.StoredCertificate{
			"acme.test": {
				ProjectRoot: "acme.test",
				SANs:        []string{"acme.test", "*.acme.test"},
				CertPath:    certPath,
				KeyPath:     keyPath,
			},
		},
	})
	if err != nil {
		t.Fatalf("new network runtime: %v", err)
	}
	if !runtime.Health().CertificateReady {
		t.Fatalf("expected prepared inventory to mark certificates ready")
	}
}

func TestNetworkRuntimeRefreshCertificatesLoadsCoverageForNewRoutes(t *testing.T) {
	certPath, keyPath := mustWriteTestCertificateFiles(t, []string{"acme.test", "*.acme.test"})
	snapshot := routing.Snapshot{Routes: map[string]routing.Route{}}
	runtime, err := NewNetworkRuntime(NetworkRuntimeConfig{
		ManagedSuffix:        "test",
		CertificateOutputDir: t.TempDir(),
		Snapshot:             func() routing.Snapshot { return snapshot },
		IssueCertificate: func(outputDir string, sans []string) (certs.IssuedCertificate, error) {
			return certs.IssuedCertificate{ProjectRoot: "acme.test", SANs: append([]string(nil), sans...), CertPath: certPath, KeyPath: keyPath}, nil
		},
	})
	if err != nil {
		t.Fatalf("new network runtime: %v", err)
	}
	if runtime.Health().CertificateReady {
		t.Fatalf("expected empty initial snapshot to start without certificates")
	}

	snapshot = routing.Snapshot{Routes: map[string]routing.Route{
		"api.acme.test": {Hostname: "api.acme.test", ServedHostnames: []string{"acme.test", "api.acme.test"}},
	}}
	if err := runtime.RefreshCertificates(); err != nil {
		t.Fatalf("refresh certificates: %v", err)
	}
	if !runtime.Health().CertificateReady {
		t.Fatalf("expected certificate readiness after refreshing routes")
	}
	if _, err := runtime.HTTPSHandler().TLSConfig().GetCertificate(&tls.ClientHelloInfo{ServerName: "api.acme.test"}); err != nil {
		t.Fatalf("expected refreshed certificate to be loaded: %v", err)
	}
}

func TestNetworkRuntimeStartBindsHTTPHTTPSAndDNS(t *testing.T) {
	runtime, err := NewNetworkRuntime(NetworkRuntimeConfig{
		ManagedSuffix: "test",
		Snapshot:      staticDaemonSnapshot(routing.Snapshot{Routes: map[string]routing.Route{}}),
		Certificates:  map[string]tls.Certificate{"acme.test": mustInlineTestCertificate(t, []string{"acme.test", "*.acme.test"})},
		DNSAddress:    freeLoopbackUDPAddress(t),
		HTTPAddress:   freeLoopbackAddress(t),
		HTTPSAddress:  freeLoopbackAddress(t),
	})
	if err != nil {
		t.Fatalf("new network runtime: %v", err)
	}
	defer func() {
		if err := runtime.Close(); err != nil {
			t.Fatalf("close runtime: %v", err)
		}
	}()

	if err := runtime.Start(); err != nil {
		t.Fatalf("start runtime: %v", err)
	}

	health := runtime.Health()
	if !health.HTTP.Bound {
		t.Fatalf("expected http listener to bind")
	}
	if !health.HTTPS.Bound {
		t.Fatalf("expected https listener to bind")
	}
	if !health.DNS.Bound {
		t.Fatalf("expected dns listener to bind")
	}
	if health.HTTP.BindAddress == "127.0.0.1:80" || health.HTTPS.BindAddress == "127.0.0.1:443" {
		t.Fatalf("expected test override addresses to be used, got http=%q https=%q", health.HTTP.BindAddress, health.HTTPS.BindAddress)
	}
	query := new(mdns.Msg)
	query.SetQuestion(mdns.Fqdn("example.test"), mdns.TypeA)
	resp, _, err := (&mdns.Client{}).Exchange(query, health.DNS.BindAddress)
	if err != nil {
		t.Fatalf("dns query failed: %v", err)
	}
	if got := parseARecord(resp); got != "127.0.0.1" {
		t.Fatalf("expected managed suffix to resolve loopback, got %q", got)
	}

	defaults, err := NewNetworkRuntime(NetworkRuntimeConfig{ManagedSuffix: "test"})
	if err != nil {
		t.Fatalf("new default network runtime: %v", err)
	}
	defaultHealth := defaults.Health()
	if defaultHealth.HTTP.BindAddress != "127.0.0.1:80" {
		t.Fatalf("expected default http bind address, got %q", defaultHealth.HTTP.BindAddress)
	}
	if defaultHealth.HTTPS.BindAddress != "127.0.0.1:443" {
		t.Fatalf("expected default https bind address, got %q", defaultHealth.HTTPS.BindAddress)
	}
}

func TestNetworkRuntimeStartDegradesWhenHTTPAndHTTPSPortsAreUnavailable(t *testing.T) {
	httpBlocker, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve http port: %v", err)
	}
	defer func() { _ = httpBlocker.Close() }()

	httpsBlocker, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve https port: %v", err)
	}
	defer func() { _ = httpsBlocker.Close() }()

	runtime, err := NewNetworkRuntime(NetworkRuntimeConfig{
		ManagedSuffix: "test",
		Snapshot:      staticDaemonSnapshot(routing.Snapshot{Routes: map[string]routing.Route{}}),
		Certificates:  map[string]tls.Certificate{"acme.test": mustInlineTestCertificate(t, []string{"acme.test", "*.acme.test"})},
		DNSAddress:    freeLoopbackUDPAddress(t),
		HTTPAddress:   httpBlocker.Addr().String(),
		HTTPSAddress:  httpsBlocker.Addr().String(),
	})
	if err != nil {
		t.Fatalf("new network runtime: %v", err)
	}
	defer func() {
		if err := runtime.Close(); err != nil {
			t.Fatalf("close runtime: %v", err)
		}
	}()

	if err := runtime.Start(); err != nil {
		t.Fatalf("expected degraded listener startup, got %v", err)
	}

	health := runtime.Health()
	if !health.DNS.Bound {
		t.Fatalf("expected dns listener to stay bound during degraded startup")
	}
	if health.HTTP.Bound || !strings.Contains(health.HTTP.LastError, "address already in use") {
		t.Fatalf("expected explicit http bind failure, got %+v", health.HTTP)
	}
	if health.HTTPS.Bound || !strings.Contains(health.HTTPS.LastError, "address already in use") {
		t.Fatalf("expected explicit https bind failure, got %+v", health.HTTPS)
	}
}

func parseARecord(resp *mdns.Msg) string {
	if resp == nil {
		return ""
	}
	for _, answer := range resp.Answer {
		record, ok := answer.(*mdns.A)
		if !ok || record.A == nil {
			continue
		}
		return record.A.String()
	}
	return ""
}

func staticDaemonSnapshot(s routing.Snapshot) func() routing.Snapshot {
	return func() routing.Snapshot { return s }
}

func freeLoopbackAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve loopback port: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("release reserved loopback port: %v", err)
	}
	return addr
}

func freeLoopbackUDPAddress(t *testing.T) string {
	t.Helper()
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve loopback udp port: %v", err)
	}
	addr := conn.LocalAddr().String()
	if err := conn.Close(); err != nil {
		t.Fatalf("release reserved udp port: %v", err)
	}
	return addr
}

func mustInlineTestCertificate(t *testing.T, dnsNames []string) tls.Certificate {
	t.Helper()
	certPEM, keyPEM := mustTestCertificatePEM(t, dnsNames)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("load inline test certificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("parse inline test certificate: %v", err)
	}
	cert.Leaf = leaf
	return cert
}

func mustWriteTestCertificateFiles(t *testing.T, dnsNames []string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	certPEM, keyPEM := mustTestCertificatePEM(t, dnsNames)
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatalf("write cert pem: %v", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatalf("write key pem: %v", err)
	}
	return certPath, keyPath
}

func mustTestCertificatePEM(t *testing.T, dnsNames []string) ([]byte, []byte) {
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

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	return certPEM, keyPEM
}
