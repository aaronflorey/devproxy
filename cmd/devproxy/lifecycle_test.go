package devproxy

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mochaka/devproxy/internal/dashboard"
)

func TestPromptCleanupScope(t *testing.T) {
	t.Parallel()

	in := bytes.NewBufferString("y\nn\ny\nn\n")
	out := &bytes.Buffer{}
	scope, err := promptCleanupScope(in, out)
	if err != nil {
		t.Fatalf("prompt failed: %v", err)
	}
	if !scope.Config || scope.State || !scope.Logs || scope.Certificates {
		t.Fatalf("unexpected scope: %+v", scope)
	}
}

func TestDashboardCommandDefaultsKeepFixedLocalURLs(t *testing.T) {
	t.Parallel()

	cmd := newDashboardCommand()
	if cmd == nil {
		t.Fatalf("expected dashboard command")
	}

	listen, err := cmd.Flags().GetString("listen")
	if err != nil {
		t.Fatalf("read --listen flag: %v", err)
	}
	if listen != dashboard.DefaultListenAddress {
		t.Fatalf("expected default listen %q, got %q", dashboard.DefaultListenAddress, listen)
	}

	if !strings.Contains("http://"+listen, "127.0.0.1:45831") {
		t.Fatalf("expected default dashboard URL to remain fixed to 127.0.0.1:45831, got http://%s", listen)
	}
	if got, want := "http://127.0.0.1:45831/logs", "http://127.0.0.1:45831/logs"; got != want {
		t.Fatalf("expected fixed logs URL %q, got %q", want, got)
	}
}

func TestReuseRunningDashboardTreatsExistingServerAsSuccess(t *testing.T) {
	t.Parallel()

	oldProbe := dashboardProbe
	oldOpen := openDashboardURL
	t.Cleanup(func() {
		dashboardProbe = oldProbe
		openDashboardURL = oldOpen
	})

	opened := ""
	dashboardProbe = func(context.Context, string) (bool, error) { return true, nil }
	openDashboardURL = func(target string) error {
		opened = target
		return nil
	}

	buf := &bytes.Buffer{}
	reused, err := reuseRunningDashboard(context.Background(), "http://127.0.0.1:45831", errors.New("listen tcp 127.0.0.1:45831: bind: address already in use"), buf, true)
	if err != nil {
		t.Fatalf("reuse running dashboard: %v", err)
	}
	if !reused {
		t.Fatalf("expected already-running dashboard to be treated as success")
	}
	if opened != "http://127.0.0.1:45831" {
		t.Fatalf("expected existing dashboard to open, got %q", opened)
	}
	if !strings.Contains(buf.String(), "dashboard already running on http://127.0.0.1:45831") {
		t.Fatalf("expected already-running message, got %q", buf.String())
	}
}
