package doctor

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mochaka/devproxy/internal/admin"
)

func TestCheckLaunchdFailsWhenStateNotRunning(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("launchctl script test is unix-only")
	}

	binDir := t.TempDir()
	launchctlPath := filepath.Join(binDir, "launchctl")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"print\" ]; then\n" +
		"  echo \"state = exited\"\n" +
		"  echo \"last exit code = 1\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 0\n"
	if err := os.WriteFile(launchctlPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake launchctl: %v", err)
	}

	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+":"+originalPath)

	err := checkLaunchd(context.Background())
	if err == nil {
		t.Fatalf("expected checkLaunchd to fail when state is not running")
	}
	if !strings.Contains(err.Error(), "not running") || !strings.Contains(err.Error(), "state = exited") {
		t.Fatalf("expected launchd state hint in error, got %v", err)
	}
}

func TestDoctorChecksRuntimeAndResolverEvidence(t *testing.T) {
	t.Parallel()

	checker := NewChecker(Dependencies{
		CheckDocker:      func(context.Context) error { return nil },
		CheckLaunchd:     func(context.Context) error { return nil },
		CheckAdminSocket: func(context.Context) error { return nil },
		CheckProxyHTTP:   func(context.Context, string) error { return nil },
		CheckProxyHTTPS:  func(context.Context, string) error { return nil },
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
		CheckMKCert:  func(context.Context) error { return nil },
		CheckLocalCA: func(context.Context) error { return nil },
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
		CheckProxyHTTP:   func(context.Context, string) error { return errors.New("connection refused") },
		CheckProxyHTTPS:  func(context.Context, string) error { return errors.New("tls handshake timeout") },
		ReadResolverState: func(context.Context) (ResolverState, error) {
			return ResolverState{ManagedSuffix: "test", ActiveResolver: true}, nil
		},
		ReadNetworkHealth: func(context.Context) (admin.NetworkRuntimeHealth, error) {
			return admin.NetworkRuntimeHealth{}, nil
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

func TestDoctorBlocksManagedProxyChecksWhenRuntimeStatusUnavailable(t *testing.T) {
	t.Parallel()

	httpCalled := false
	httpsCalled := false
	checker := NewChecker(Dependencies{
		ReadResolverState: func(context.Context) (ResolverState, error) {
			return ResolverState{ManagedSuffix: "test", ActiveResolver: true}, nil
		},
		ReadNetworkHealth: func(context.Context) (admin.NetworkRuntimeHealth, error) {
			return admin.NetworkRuntimeHealth{}, errors.New("admin socket unreachable")
		},
		CheckProxyHTTP: func(context.Context, string) error {
			httpCalled = true
			return nil
		},
		CheckProxyHTTPS: func(context.Context, string) error {
			httpsCalled = true
			return nil
		},
	})

	report := checker.Run(context.Background(), "example.test")
	for _, name := range []string{"http_listener", "https_listener", "proxy_http", "proxy_https"} {
		check := checkByName(t, report, name)
		if check.OK {
			t.Fatalf("expected %s to fail when daemon status unavailable", name)
		}
		if !strings.Contains(check.Message, "cannot verify managed proxy reachability without daemon status") && strings.HasPrefix(name, "proxy_") {
			t.Fatalf("unexpected proxy message for %s: %q", name, check.Message)
		}
	}
	if httpCalled || httpsCalled {
		t.Fatalf("expected managed proxy probes not to run without runtime status")
	}
}

func TestDoctorPassesManagedHostToProxyChecks(t *testing.T) {
	t.Parallel()

	var httpHost, httpsHost string
	checker := NewChecker(Dependencies{
		ReadResolverState: func(context.Context) (ResolverState, error) {
			return ResolverState{ManagedSuffix: "test", ActiveResolver: true}, nil
		},
		ReadNetworkHealth: func(context.Context) (admin.NetworkRuntimeHealth, error) {
			return admin.NetworkRuntimeHealth{HTTP: admin.ListenerStatus{Bound: true}, HTTPS: admin.ListenerStatus{Bound: true}}, nil
		},
		CheckProxyHTTP: func(_ context.Context, host string) error {
			httpHost = host
			return nil
		},
		CheckProxyHTTPS: func(_ context.Context, host string) error {
			httpsHost = host
			return nil
		},
	})

	checker.Run(context.Background(), "example.test")
	if httpHost != "example.test" || httpsHost != "example.test" {
		t.Fatalf("expected managed host to be forwarded, got http=%q https=%q", httpHost, httpsHost)
	}
}

func TestScutilHasManagedResolverMatchesRealSpacing(t *testing.T) {
	t.Parallel()

	scutil := `resolver #8
  domain   : test
  nameserver[0] : 127.0.0.1
  port     : 53535
`

	if !scutilHasManagedResolver(scutil, "test") {
		t.Fatalf("expected resolver parser to match scutil spacing variant")
	}
}

func TestResolveExampleHostUsesDSCacheUtilOutput(t *testing.T) {
	originalExecCommand := execCommand
	execCommand = fakeExecCommand
	t.Cleanup(func() { execCommand = originalExecCommand })

	addr, err := resolveExampleHost(context.Background(), "example.test")
	if err != nil {
		t.Fatalf("resolveExampleHost returned error: %v", err)
	}
	if addr != "127.0.0.1" {
		t.Fatalf("expected 127.0.0.1, got %q", addr)
	}
}

func TestParseDSCacheUtilAddressNoAddress(t *testing.T) {
	t.Parallel()
	if got := parseDSCacheUtilAddress("name: example.test\n"); got != "" {
		t.Fatalf("expected empty address, got %q", got)
	}
}

func fakeExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	sep := -1
	for i, a := range args {
		if a == "--" {
			sep = i
			break
		}
	}
	if sep == -1 || sep+1 >= len(args) {
		os.Exit(2)
	}

	cmd := args[sep+1]
	if cmd == "dscacheutil" {
		_, _ = os.Stdout.WriteString("name: example.test\nip_address: 127.0.0.1\n")
		os.Exit(0)
	}
	os.Exit(3)
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
