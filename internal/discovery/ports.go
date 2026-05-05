package discovery

import "slices"

type PublishedPort struct {
	HostIP   string
	HostPort int
	Protocol string
}

var preferredPorts = []int{443, 8443, 80, 8080, 8000, 3000, 5173, 8025}

func SelectPublishedPort(ports []PublishedPort, opts RouteOptions, configOverridePort int) (PublishedPort, string, bool) {
	tcp := make([]PublishedPort, 0, len(ports))
	for _, p := range ports {
		if p.Protocol == "tcp" {
			tcp = append(tcp, p)
		}
	}
	if len(tcp) == 0 {
		return PublishedPort{}, "", false
	}

	if opts.LabelPort != nil {
		if port, ok := lookupPort(tcp, *opts.LabelPort); ok {
			return port, "label", true
		}
	}

	if configOverridePort > 0 {
		if port, ok := lookupPort(tcp, configOverridePort); ok {
			return port, "override", true
		}
	}

	for _, preferred := range preferredPorts {
		if port, ok := lookupPort(tcp, preferred); ok {
			return port, "preference", true
		}
	}

	return tcp[0], "first-published", true
}

func lookupPort(ports []PublishedPort, value int) (PublishedPort, bool) {
	idx := slices.IndexFunc(ports, func(p PublishedPort) bool {
		return p.HostPort == value
	})
	if idx == -1 {
		return PublishedPort{}, false
	}
	return ports[idx], true
}
