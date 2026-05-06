package admin

import "github.com/mochaka/devproxy/internal/routing"

type RouteView struct {
	Hostname        string
	UpstreamScheme  string
	UpstreamHost    string
	UpstreamPort    int
	OpenURL         string
	PreferredScheme string
	FallbackReason  string
	HTTPSReady      bool
	HandlingState   string
	Winner          string
	Losers          []string
	Conflict        bool
}

func RoutesFromSnapshot(snapshot routing.Snapshot) []RouteView {
	return RoutesFromSnapshotWithRuntime(snapshot, true)
}

func RoutesFromSnapshotWithRuntime(snapshot routing.Snapshot, httpsReady bool) []RouteView {
	conflictsByHost := map[string]routing.Conflict{}
	for _, c := range snapshot.Conflicts {
		conflictsByHost[c.Hostname] = c
	}

	out := []RouteView{}
	for host, route := range snapshot.Routes {
		preferredScheme := "http"
		fallbackReason := ""
		if route.Upstream.Scheme == "https" {
			preferredScheme = "https"
			if !httpsReady {
				preferredScheme = "http"
				fallbackReason = "https runtime is not ready"
			}
		} else {
			fallbackReason = "route is configured for HTTP"
		}
		view := RouteView{
			Hostname:        host,
			UpstreamScheme:  route.Upstream.Scheme,
			UpstreamHost:    route.Upstream.Host,
			UpstreamPort:    route.Upstream.Port,
			OpenURL:         preferredScheme + "://" + host,
			PreferredScheme: preferredScheme,
			FallbackReason:  fallbackReason,
			HTTPSReady:      httpsReady,
			HandlingState:   "proxy",
			Winner:          route.Winner.ContainerName,
		}
		if conflict, ok := conflictsByHost[host]; ok {
			view.Conflict = true
			for _, loser := range conflict.Losers {
				view.Losers = append(view.Losers, loser.ContainerName)
			}
		}
		out = append(out, view)
	}
	return out
}
