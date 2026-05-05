package doctor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"

	"github.com/mochaka/devproxy/internal/admin"
	"github.com/mochaka/devproxy/internal/adminapi"
)

type ResolverState struct {
	ManagedSuffix  string
	ActiveResolver bool
	Nameservers    []string
	Evidence       string
}

type CheckResult struct {
	Name    string
	OK      bool
	Message string
}

type Report struct {
	Checks []CheckResult
}

type Dependencies struct {
	CheckDocker        func(context.Context) error
	CheckLaunchd       func(context.Context) error
	CheckAdminSocket   func(context.Context) error
	CheckProxyHTTP     func(context.Context) error
	CheckProxyHTTPS    func(context.Context) error
	CheckMKCert        func(context.Context) error
	CheckLocalCA       func(context.Context) error
	ReadResolverState  func(context.Context) (ResolverState, error)
	ResolveExampleHost func(context.Context, string) (string, error)
	ReadNetworkHealth  func(context.Context) (admin.NetworkRuntimeHealth, error)
}

type Checker struct{ deps Dependencies }

func NewChecker(deps Dependencies) *Checker {
	if deps.CheckDocker == nil {
		deps.CheckDocker = checkDocker
	}
	if deps.CheckLaunchd == nil {
		deps.CheckLaunchd = checkLaunchd
	}
	if deps.CheckAdminSocket == nil {
		deps.CheckAdminSocket = checkAdminSocket
	}
	if deps.CheckProxyHTTP == nil {
		deps.CheckProxyHTTP = checkProxyHTTP
	}
	if deps.CheckProxyHTTPS == nil {
		deps.CheckProxyHTTPS = checkProxyHTTPS
	}
	if deps.CheckMKCert == nil {
		deps.CheckMKCert = checkMKCert
	}
	if deps.CheckLocalCA == nil {
		deps.CheckLocalCA = checkLocalCA
	}
	if deps.ReadResolverState == nil {
		deps.ReadResolverState = readResolverState
	}
	if deps.ResolveExampleHost == nil {
		deps.ResolveExampleHost = resolveExampleHost
	}
	if deps.ReadNetworkHealth == nil {
		deps.ReadNetworkHealth = readNetworkHealth
	}
	return &Checker{deps: deps}
}

func (c *Checker) Run(ctx context.Context, exampleHost string) Report {
	checks := []CheckResult{
		probe("docker", c.deps.CheckDocker, ctx),
		probe("launchd", c.deps.CheckLaunchd, ctx),
		probe("admin_socket", c.deps.CheckAdminSocket, ctx),
	}

	resolverState, resolverErr := c.deps.ReadResolverState(ctx)
	if resolverErr != nil {
		checks = append(checks, CheckResult{Name: "resolver_state", OK: false, Message: resolverErr.Error()})
	} else if !resolverState.ActiveResolver {
		message := resolverState.Evidence
		if message == "" {
			message = "scutil --dns shows no active managed resolver"
		}
		checks = append(checks, CheckResult{Name: "resolver_state", OK: false, Message: message})
	} else {
		checks = append(checks, CheckResult{Name: "resolver_state", OK: true, Message: resolverMessage(resolverState)})
	}

	health, healthErr := c.deps.ReadNetworkHealth(ctx)
	if healthErr != nil {
		checks = append(checks,
			CheckResult{Name: "http_listener", OK: false, Message: healthErr.Error()},
			CheckResult{Name: "https_listener", OK: false, Message: healthErr.Error()},
		)
	} else {
		checks = append(checks,
			listenerCheck("http_listener", health.HTTP),
			listenerCheck("https_listener", health.HTTPS),
		)
	}

	checks = append(checks,
		probe("proxy_http", c.deps.CheckProxyHTTP, ctx),
		probe("proxy_https", c.deps.CheckProxyHTTPS, ctx),
		probe("mkcert", c.deps.CheckMKCert, ctx),
		probe("local_ca", c.deps.CheckLocalCA, ctx),
	)

	if strings.TrimSpace(exampleHost) != "" {
		_, err := c.deps.ResolveExampleHost(ctx, exampleHost)
		if err != nil {
			checks = append(checks, CheckResult{Name: "managed_domain_resolution", OK: false, Message: err.Error()})
		} else {
			checks = append(checks, CheckResult{Name: "managed_domain_resolution", OK: true, Message: "example managed domain resolves"})
		}
	}

	return Report{Checks: checks}
}

func probe(name string, fn func(context.Context) error, ctx context.Context) CheckResult {
	if err := fn(ctx); err != nil {
		return CheckResult{Name: name, OK: false, Message: err.Error()}
	}
	return CheckResult{Name: name, OK: true, Message: "ok"}
}

func listenerCheck(name string, listener admin.ListenerStatus) CheckResult {
	if !listener.Bound {
		msg := "not bound"
		if listener.LastError != "" {
			msg = listener.LastError
		}
		return CheckResult{Name: name, OK: false, Message: msg}
	}
	return CheckResult{Name: name, OK: true, Message: listener.BindAddress}
}

func resolverMessage(state ResolverState) string {
	ns := strings.Join(state.Nameservers, ",")
	if ns == "" {
		ns = "none"
	}
	return fmt.Sprintf("scutil resolver active for .%s via %s", state.ManagedSuffix, ns)
}

func checkDocker(context.Context) error {
	cmd := exec.Command("docker", "info")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker info failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func checkLaunchd(context.Context) error {
	cmd := exec.Command("launchctl", "print", "system/com.devproxy.daemon")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl print system/com.devproxy.daemon failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func checkAdminSocket(ctx context.Context) error {
	client := adminapi.NewClient("/tmp/devproxy/admin.sock")
	_, err := client.Status(ctx)
	if err != nil {
		return fmt.Errorf("admin socket unreachable: %w", err)
	}
	return nil
}

func checkProxyHTTP(context.Context) error {
	resp, err := http.Get("http://127.0.0.1")
	if err != nil {
		return fmt.Errorf("http proxy unreachable: %w", err)
	}
	_ = resp.Body.Close()
	return nil
}

func checkProxyHTTPS(context.Context) error {
	resp, err := http.Get("https://127.0.0.1")
	if err != nil {
		return fmt.Errorf("https proxy unreachable: %w", err)
	}
	_ = resp.Body.Close()
	return nil
}

func checkMKCert(context.Context) error {
	if _, err := exec.LookPath("mkcert"); err != nil {
		return fmt.Errorf("mkcert not found: %w", err)
	}
	return nil
}

func checkLocalCA(context.Context) error {
	cmd := exec.Command("mkcert", "-CAROOT")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mkcert local CA unavailable: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if strings.TrimSpace(string(out)) == "" {
		return fmt.Errorf("mkcert local CA root is empty")
	}
	return nil
}

func readResolverState(context.Context) (ResolverState, error) {
	cmd := exec.Command("scutil", "--dns")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ResolverState{}, fmt.Errorf("scutil --dns failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	text := string(out)
	active := strings.Contains(text, "domain : test") || strings.Contains(text, "domain : .test")
	return ResolverState{ManagedSuffix: "test", ActiveResolver: active, Nameservers: []string{"127.0.0.1"}, Evidence: "scutil --dns inspected"}, nil
}

func resolveExampleHost(_ context.Context, host string) (string, error) {
	addrs, err := net.LookupHost(host)
	if err != nil {
		return "", fmt.Errorf("lookup %s failed: %w", host, err)
	}
	if len(addrs) == 0 {
		return "", fmt.Errorf("lookup %s returned no addresses", host)
	}
	return addrs[0], nil
}

func readNetworkHealth(ctx context.Context) (admin.NetworkRuntimeHealth, error) {
	client := adminapi.NewClient("/tmp/devproxy/admin.sock")
	status, err := client.Status(ctx)
	if err != nil {
		return admin.NetworkRuntimeHealth{}, fmt.Errorf("read runtime status: %w", err)
	}
	return admin.NetworkRuntimeHealth{
		DNS:              admin.ListenerStatus{Enabled: true, Bound: status.DNS.Healthy, BindAddress: "127.0.0.1:53535"},
		HTTP:             status.HTTP,
		HTTPS:            status.HTTPS,
		Paused:           status.Paused,
		CertificateReady: status.CertificateReady,
		ManagedSuffix:    status.DNS.ManagedSuffix,
	}, nil
}
