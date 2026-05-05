package discovery

import (
	"strconv"
	"strings"

	"github.com/mochaka/devproxy/internal/routing"
)

type Container struct {
	ID     string
	Name   string
	Labels map[string]string
}

type RouteOptions struct {
	EnabledLabel *bool
	LabelDomain  *string
	LabelDomains []string
	LabelRoot    *bool
	LabelPort    *int
	LabelScheme  *string
	LabelPriority *int
	OverridePort int
}

type WarningRecord = routing.Warning

func BuildCandidateBase(container Container) routing.Candidate {
	project := strings.TrimSpace(container.Labels["com.docker.compose.project"])
	service := strings.TrimSpace(container.Labels["com.docker.compose.service"])
	source := routing.SourceComposeLabels

	if project == "" || service == "" {
		project, service = parseContainerName(container.Name)
		source = routing.SourceContainerName
	}

	return routing.Candidate{
		ContainerID:   container.ID,
		ContainerName: strings.TrimPrefix(container.Name, "/"),
		Project:       project,
		Service:       service,
		Source:        source,
		Labels:        container.Labels,
	}
}

func ApplyLabelFields(base RouteOptions, container Container, warnings []WarningRecord) (RouteOptions, []WarningRecord) {
	result := base
	labels := container.Labels

	if v, ok := labels["devproxy.enable"]; ok {
		if parsed, ok := parseBool(v); ok {
			result.EnabledLabel = &parsed
		} else {
			warnings = append(warnings, badLabelWarning(container, "devproxy.enable", v))
		}
	}

	if v := strings.TrimSpace(labels["devproxy.domain"]); v != "" {
		result.LabelDomain = &v
	}

	if v := strings.TrimSpace(labels["devproxy.domains"]); v != "" {
		for _, domain := range strings.Split(v, ",") {
			d := strings.TrimSpace(domain)
			if d != "" {
				result.LabelDomains = append(result.LabelDomains, d)
			}
		}
	}

	if v, ok := labels["devproxy.root"]; ok {
		if parsed, ok := parseBool(v); ok {
			result.LabelRoot = &parsed
		} else {
			warnings = append(warnings, badLabelWarning(container, "devproxy.root", v))
		}
	}

	if v, ok := labels["devproxy.port"]; ok {
		if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && parsed > 0 {
			result.LabelPort = &parsed
		} else {
			warnings = append(warnings, badLabelWarning(container, "devproxy.port", v))
		}
	}

	if v := strings.TrimSpace(labels["devproxy.scheme"]); v != "" {
		if v == "http" || v == "https" {
			result.LabelScheme = &v
		} else {
			warnings = append(warnings, badLabelWarning(container, "devproxy.scheme", v))
		}
	}

	if v, ok := labels["devproxy.priority"]; ok {
		if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			result.LabelPriority = &parsed
		} else {
			warnings = append(warnings, badLabelWarning(container, "devproxy.priority", v))
		}
	}

	return result, warnings
}

func parseContainerName(name string) (string, string) {
	clean := strings.TrimPrefix(name, "/")
	parts := strings.Split(clean, "-")
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return clean, clean
}

func parseBool(v string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1", "yes":
		return true, true
	case "false", "0", "no":
		return false, true
	default:
		return false, false
	}
}

func badLabelWarning(c Container, field, value string) WarningRecord {
	return WarningRecord{Code: "invalid_label", Message: "ignored malformed label value", Container: strings.TrimPrefix(c.Name, "/"), Field: field, Severity: "warning", Source: value}
}
