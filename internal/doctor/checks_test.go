package doctor

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mochaka/devproxy/internal/admin"
)

func TestDoctorChecksRuntimeAndResolverEvidence(t *testing.T) {
	t.Parallel()

	checker := NewChecker(Dependencies{
		CheckDocker:       func(context.Context) error { return nil },
		CheckLaunchd:      func(context.Context) error { return nil },
		CheckAdminSocket:  func(context.Context) error { return nil },
		CheckProxyHTTP:    func(context.Context) error { return nil },
		CheckProxyHTTPS:   func(context.Context) error { return nil },
		ResolveExampleHost: func(context.Context, string) (string, error) {
			return "127.0.0.1", nil
		},
		ReadResolverState: func(context.Context) (ResolverState, error) {
			return ResolverState{ManagedSuffix: "test", ActiveResolver: true, Nameservers: []string{"127.0.0.1"}}, nil
		},
		ReadNetworkHealth: func(context.Context) (admin.NetworkRuntimeHealth, error) {
			return admin.NetworkRuntimeHealth{
				DNS:              admin.ListenerStatus{Enabled: true, Bound: true, BindAddress: "127.0.0.1:53535"},
				HTTP:             admin.ListenerStatus{Enabled: true, Bound: true, BindAddress: "127.0.0.1:80"},
				HTTPS:            admin.ListenerStatus{Enabled: true, Bound: true, BindAddress: "127.0.0.1:443"},
				CertificateReady: true,
				ManagedSuffix:    "test",
			}, nil
		},
		CheckMKCert:   func(context.Context) error { return nil },
		CheckLocalCA:  func(context.Context) error { return nil },
	})

	report := checker.Run(context.Background(), "acme.test")

	assertCheckOK(t, report, "docker")
	assertCheckOK(t, report, "launchd")
	assertCheckOK(t, report, "admin_socket")
	assertCheckOK(t, report, "resolver_state")
	assertCheckOK(t, report, "http_listener")
	assertCheckOK(t, report, "https_listener")
	assertCheckOK(t, report, "proxy_http")
	assertCheckOK(t, report, "proxy_https")
	assertCheckOK(t, report, "mkcert")
	assertCheckOK(t, report, "local_ca")
	assertCheckOK(t, report, "managed_domain_resolution")
}

func TestDoctorResolverCheckRequiresScutilAlignedActivation(t *testing.T) {
	t.Parallel()

	checker := NewChecker(Dependencies{
		ReadResolverState: func(context.Context) (ResolverState, error) {
			return ResolverState{ManagedSuffix: "test", ActiveResolver: false, Evidence: "scutil --dns missing resolver for test"}, nil
		},
	})

	report := checker.Run(context.Background(), "acme.test")
	resolver := checkByName(t, report, "resolver_state")
	if resolver.OK {
		t.Fatalf("expected resolver_state to fail when scutil-aligned resolver is inactive")
	}
	if !strings.Contains(resolver.Message, "scutil") {
		t.Fatalf("expected resolver failure to reference scutil evidence, got %q", resolver.Message)
	}
}

func TestDoctorIncludesLiveAdminAndProxyReachabilityFailures(t *testing.T) {
	t.Parallel()

	checker := NewChecker(Dependencies{
		CheckAdminSocket: func(context.Context) error { return errors.New("connect: no such file") },
		CheckProxyHTTP:   func(context.Context) error { return errors.New("connection refused") },
		CheckProxyHTTPS:  func(context.Context) error { return errors.New("tls handshake timeout") },
		ReadResolverState: func(context.Context) (ResolverState, error) {
			return ResolverState{ManagedSuffix: "test", ActiveResolver: true}, nil
		},
	})

	report := checker.Run(context.Background(), "acme.test")

	if checkByName(t, report, "admin_socket").OK {
		t.Fatalf("expected admin_socket check to fail")
	}
	if checkByName(t, report, "proxy_http").OK {
		t.Fatalf("expected proxy_http check to fail")
	}
	if checkByName(t, report, "proxy_https").OK {
		t.Fatalf("expected proxy_https check to fail")
	}
}

func assertCheckOK(t *testing.T, report Report, name string) {
	t.Helper()
	check := checkByName(t, report, name)
	if !check.OK {
		t.Fatalf("expected check %q to pass, got failure: %s", name, check.Message)
	}
}

func checkByName(t *testing.T, report Report, name string) CheckResult {
	t.Helper()
	for _, check := range report.Checks {
		if check.Name == name {
			return check
		}
	}
	t.Fatalf("missing check %q", name)
	return CheckResult{}
}
