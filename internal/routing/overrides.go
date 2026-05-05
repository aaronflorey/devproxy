package routing

type RoutePreferences struct {
	Enable   *bool
	Domain   string
	Domains  []string
	Root     *bool
	Port     *int
	Scheme   string
	Priority *int
}

func MergeOverrides(config, labels RoutePreferences) (RoutePreferences, []Warning) {
	merged := config
	if labels.Enable != nil {
		merged.Enable = labels.Enable
	}
	if labels.Domain != "" {
		merged.Domain = labels.Domain
	}
	if len(labels.Domains) > 0 {
		merged.Domains = append([]string{}, labels.Domains...)
	}
	if labels.Root != nil {
		merged.Root = labels.Root
	}
	if labels.Port != nil {
		merged.Port = labels.Port
	}
	if labels.Scheme != "" {
		merged.Scheme = labels.Scheme
	}
	if labels.Priority != nil {
		merged.Priority = labels.Priority
	}
	return merged, nil
}
