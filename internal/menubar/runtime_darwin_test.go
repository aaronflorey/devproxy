//go:build darwin

package menubar

import (
	"context"
	"testing"
	"time"
)

func TestRuntimeRouteSlotAssignmentsExposeProjectedRoutes(t *testing.T) {
	routes := []routeMenuItem{
		{Hostname: "api.acme.test", OpenURL: "https://api.acme.test"},
		{Hostname: "acme.test", OpenURL: "http://acme.test"},
	}

	assignments := computeRouteSlotAssignments(0, routes)
	if len(assignments) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(assignments))
	}
	if !assignments[0].visible || assignments[0].host != "api.acme.test" || assignments[0].openURL != "https://api.acme.test" {
		t.Fatalf("first assignment mismatch: %+v", assignments[0])
	}
	if !assignments[1].visible || assignments[1].host != "acme.test" || assignments[1].openURL != "http://acme.test" {
		t.Fatalf("second assignment mismatch: %+v", assignments[1])
	}
}

func TestRuntimeRouteSlotAssignmentsHideStaleSlotsOnShrink(t *testing.T) {
	routes := []routeMenuItem{{Hostname: "api.acme.test", OpenURL: "https://api.acme.test"}}

	assignments := computeRouteSlotAssignments(3, routes)
	if len(assignments) != 3 {
		t.Fatalf("expected 3 assignments, got %d", len(assignments))
	}
	if !assignments[0].visible {
		t.Fatalf("expected first slot to remain visible")
	}
	for i := 1; i < len(assignments); i++ {
		if assignments[i].visible {
			t.Fatalf("expected stale slot %d to be hidden", i)
		}
		if assignments[i].openURL != "" {
			t.Fatalf("expected stale slot %d openURL cleared, got %q", i, assignments[i].openURL)
		}
	}
}

func TestRunContextBoundSystrayQuitsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runStarted := make(chan struct{})
	runReleased := make(chan struct{})
	quitCalled := make(chan struct{}, 1)
	onReadyCalled := make(chan struct{}, 1)
	onExitCalled := make(chan struct{}, 1)
	done := make(chan struct{})

	go func() {
		runContextBoundSystray(
			ctx,
			func() { onReadyCalled <- struct{}{} },
			func() { onExitCalled <- struct{}{} },
			func(onReady, onExit func()) {
				close(runStarted)
				onReady()
				<-runReleased
				onExit()
			},
			func() {
				quitCalled <- struct{}{}
				close(runReleased)
			},
		)
		close(done)
	}()

	select {
	case <-runStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("expected systray runner to start")
	}

	select {
	case <-onReadyCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("expected onReady callback")
	}

	cancel()

	select {
	case <-quitCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("expected quit callback after cancellation")
	}

	select {
	case <-onExitCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("expected onExit callback before return")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("expected systray runner wrapper to return")
	}
}
