package routing

import "sort"

func ResolveConflicts(routes []Route) ([]Route, []Conflict) {
	grouped := map[string][]Route{}
	for _, route := range routes {
		grouped[route.Hostname] = append(grouped[route.Hostname], route)
	}

	active := make([]Route, 0, len(grouped))
	conflicts := []Conflict{}

	for hostname, claims := range grouped {
		sort.SliceStable(claims, func(i, j int) bool {
			if claims[i].Priority != claims[j].Priority {
				return claims[i].Priority > claims[j].Priority
			}
			return claims[i].Winner.ContainerName < claims[j].Winner.ContainerName
		})

		winner := claims[0]
		active = append(active, winner)

		if len(claims) > 1 {
			losers := make([]Candidate, 0, len(claims)-1)
			for _, claim := range claims[1:] {
				losers = append(losers, claim.Winner)
			}
			conflicts = append(conflicts, Conflict{
				Hostname:    hostname,
				Winner:      winner.Winner,
				Losers:      losers,
				Reason:      "highest priority, then stable container-name tie-break",
				PriorityTie: claims[0].Priority == claims[1].Priority,
			})
		}
	}

	return active, conflicts
}
