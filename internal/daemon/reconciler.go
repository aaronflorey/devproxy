package daemon

import (
	"fmt"
	"sync"
	"time"

	"github.com/mochaka/devproxy/internal/config"
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
	Suffix          string
	RootServices    []string
	IgnoredServices []string
	IgnoredPorts    []int
	Overrides       map[string]config.ProjectConfig
}

type Reconciler struct {
	mu            sync.RWMutex
	builder       *registry.Builder
	opts          ReconcilerOptions
	snapshot      routing.Snapshot
	lastSync      time.Time
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
		configPrefs := r.servicePreferences(candidate.Project, candidate.Service)
		labelPrefs := routePreferencesFromLabels(routeOpts)
		effectivePrefs, overrideWarnings := routing.MergeOverrides(configPrefs, labelPrefs)
		warnings = append(warnings, overrideWarnings...)

		configPort := 0
		if effectivePrefs.Port != nil {
			configPort = *effectivePrefs.Port
		}

		selected, source, ok := discovery.SelectPublishedPort(c.Ports, routeOpts, configPort)
		if !ok {
			continue
		}
		if !discovery.IsEligible(discovery.CandidateInput{Service: candidate.Service, Running: c.Running, PublishedTCPPorts: []int{selected.HostPort}}, discovery.EligibilityConfig{IgnoredServices: r.opts.IgnoredServices, IgnoredPorts: r.opts.IgnoredPorts}) {
			continue
		}
		domains, domainWarnings := routing.GenerateDomains(candidate.Project, candidate.Service, effectivePrefs, routing.RouteOptions{Suffix: r.opts.Suffix, RootServices: r.opts.RootServices})
		warnings = append(warnings, domainWarnings...)
		scheme := "http"
		if effectivePrefs.Scheme != "" {
			scheme = effectivePrefs.Scheme
		}
		priority := 0
		if effectivePrefs.Priority != nil {
			priority = *effectivePrefs.Priority
		}
		for _, domain := range domains {
			routes = append(routes, routing.Route{Hostname: domain, Domains: domains, ServedHostnames: domains, Winner: candidate, Upstream: routing.Upstream{Host: "127.0.0.1", Port: selected.HostPort, Scheme: scheme}, Priority: priority, HTTPSRedirect: false, HTTPSOnly: false, Provenance: routing.RouteProvenance{PortSource: source}})
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

func (r *Reconciler) servicePreferences(project, service string) routing.RoutePreferences {
	projectConfig, ok := r.opts.Overrides[project]
	if !ok {
		return routing.RoutePreferences{}
	}
	override, ok := projectConfig.Services[service]
	if !ok {
		return routing.RoutePreferences{}
	}

	prefs := routing.RoutePreferences{
		Enable:   override.Enable,
		Domain:   override.Domain,
		Root:     override.Root,
		Port:     override.Port,
		Scheme:   override.Scheme,
		Priority: override.Priority,
	}
	if len(override.Domains) > 0 {
		prefs.Domains = append([]string(nil), override.Domains...)
	}
	return prefs
}

func routePreferencesFromLabels(opts discovery.RouteOptions) routing.RoutePreferences {
	prefs := routing.RoutePreferences{
		Enable:   opts.EnabledLabel,
		Root:     opts.LabelRoot,
		Port:     opts.LabelPort,
		Priority: opts.LabelPriority,
	}
	if opts.LabelDomain != nil {
		prefs.Domain = *opts.LabelDomain
	}
	if len(opts.LabelDomains) > 0 {
		prefs.Domains = append([]string(nil), opts.LabelDomains...)
	}
	if opts.LabelScheme != nil {
		prefs.Scheme = *opts.LabelScheme
	}
	return prefs
}
