package discovery

import "slices"

type CandidateInput struct {
	Service           string
	Running           bool
	EnabledLabel      *bool
	DisabledLabel     *bool
	PublishedTCPPorts []int
}

type EligibilityConfig struct {
	IgnoredServices []string
	IgnoredPorts    []int
}

func IsEligible(in CandidateInput, cfg EligibilityConfig) bool {
	if !in.Running {
		return false
	}

	if len(in.PublishedTCPPorts) == 0 {
		return false
	}

	if in.DisabledLabel != nil && *in.DisabledLabel {
		return false
	}

	if in.EnabledLabel != nil && *in.EnabledLabel {
		return true
	}

	if slices.Contains(cfg.IgnoredServices, in.Service) {
		return false
	}

	allIgnored := true
	for _, p := range in.PublishedTCPPorts {
		if !slices.Contains(cfg.IgnoredPorts, p) {
			allIgnored = false
			break
		}
	}

	return !allIgnored
}
