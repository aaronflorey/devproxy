package daemon

import (
	"fmt"
	"sync"
	"time"

	"github.com/mochaka/devproxy/internal/discovery"
	"github.com/mochaka/devproxy/internal/registry"
	"github.com/mochaka/devproxy/internal/routing"
)

type ContainerState struct {
	ID      string
	Name    string
	Running bool
	Labels  map[string]string
	Ports   []discovery.PublishedPort
}

type ReconcilerOptions struct {
	Suffix         string
	RootServices   []string
	IgnoredServices []string
	IgnoredPorts   []int
}

type Reconciler struct {
	mu       sync.RWMutex
	builder  *registry.Builder
	opts     ReconcilerOptions
	snapshot routing.Snapshot
	lastSync time.Time
	routingPaused bool
}

func NewReconciler(opts ReconcilerOptions) *Reconciler {
	return &Reconciler{builder: registry.NewBuilder(), opts: opts}
}

func (r *Reconciler) RebuildSnapshot(containers []ContainerState) error {
	routes := make([]routing.Route, 0)
	warnings := []routing.Warning{}

	for _, c := range containers {
		candidate := discovery.BuildCandidateBase(discovery.Container{ID: c.ID, Name: c.Name, Labels: c.Labels})
		routeOpts, warningsWithLabels := discovery.ApplyLabelFields(discovery.RouteOptions{}, discovery.Container{ID: c.ID, Name: c.Name, Labels: c.Labels}, warnings)
		warnings = warningsWithLabels
		selected, source, ok := discovery.SelectPublishedPort(c.Ports, routeOpts, 0)
		if !ok {
			continue
		}
		if !discovery.IsEligible(discovery.CandidateInput{Service: candidate.Service, Running: c.Running, PublishedTCPPorts: []int{selected.HostPort}}, discovery.EligibilityConfig{IgnoredServices: r.opts.IgnoredServices, IgnoredPorts: r.opts.IgnoredPorts}) {
			continue
		}
		domains, domainWarnings := routing.GenerateDomains(candidate.Project, candidate.Service, routing.RoutePreferences{}, routing.RouteOptions{Suffix: r.opts.Suffix, RootServices: r.opts.RootServices})
		warnings = append(warnings, domainWarnings...)
		scheme := "http"
		if routeOpts.LabelScheme != nil {
			scheme = *routeOpts.LabelScheme
		}
		for _, domain := range domains {
			routes = append(routes, routing.Route{Hostname: domain, Domains: domains, ServedHostnames: domains, Winner: candidate, Upstream: routing.Upstream{Host: "127.0.0.1", Port: selected.HostPort, Scheme: scheme}, Priority: 0, HTTPSRedirect: false, HTTPSOnly: false, Provenance: routing.RouteProvenance{PortSource: source}})
		}
	}

	active, conflicts := routing.ResolveConflicts(routes)
	snap := r.builder.Build(active, conflicts, warnings)

	r.mu.Lock()
	r.snapshot = snap
	r.lastSync = snap.CreatedAt
	r.mu.Unlock()

	return nil
}

func (r *Reconciler) SetRoutingPaused(paused bool) {
	r.mu.Lock()
	r.routingPaused = paused
	r.mu.Unlock()
}

func (r *Reconciler) IsRoutingPaused() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.routingPaused
}

func (r *Reconciler) Snapshot() routing.Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.snapshot
}

func (r *Reconciler) LastSync() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastSync
}

func (r *Reconciler) HandleEvent(event string, containers []ContainerState) error {
	switch event {
	case "start", "stop", "die", "destroy", "rename", "update":
		return r.RebuildSnapshot(containers)
	default:
		return fmt.Errorf("unsupported event %q", event)
	}
}
