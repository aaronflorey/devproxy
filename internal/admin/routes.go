package admin

import "github.com/mochaka/devproxy/internal/routing"

type RouteView struct {
	Hostname      string
	UpstreamScheme string
	UpstreamHost  string
	UpstreamPort  int
	HandlingState string
	Winner        string
	Losers        []string
	Conflict      bool
}

func RoutesFromSnapshot(snapshot routing.Snapshot) []RouteView {
	conflictsByHost := map[string]routing.Conflict{}
	for _, c := range snapshot.Conflicts {
		conflictsByHost[c.Hostname] = c
	}

	out := []RouteView{}
	for host, route := range snapshot.Routes {
		view := RouteView{Hostname: host, UpstreamScheme: route.Upstream.Scheme, UpstreamHost: route.Upstream.Host, UpstreamPort: route.Upstream.Port, HandlingState: "proxy", Winner: route.Winner.ContainerName}
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
