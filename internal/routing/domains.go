package routing

import (
	"fmt"
	"slices"
	"strings"
)

type RouteOptions struct {
	Suffix       string
	RootServices []string
}

var publicSuffixes = []string{".com", ".net", ".org", ".io", ".dev", ".co"}

func GenerateDomains(project, service string, prefs RoutePreferences, opts RouteOptions) ([]string, []Warning) {
	warnings := []Warning{}
	if opts.Suffix == "" {
		opts.Suffix = "test"
	}

	if prefs.Domain != "" {
		if isPublicDomain(prefs.Domain) {
			warnings = append(warnings, Warning{Code: "public_suffix_rejected", Message: "explicit domain rejected because suffix appears public", Field: "domain", Severity: "error"})
			return nil, warnings
		}
		if !strings.HasSuffix(prefs.Domain, "."+opts.Suffix) {
			warnings = append(warnings, Warning{Code: "unmanaged_suffix", Message: "explicit domain uses unmanaged suffix", Field: "domain", Severity: "warning"})
		}
		return []string{prefs.Domain}, warnings
	}

	domains := []string{fmt.Sprintf("%s.%s.%s", service, project, opts.Suffix)}
	shouldRoot := slices.Contains(opts.RootServices, service)
	if prefs.Root != nil {
		shouldRoot = *prefs.Root
	}
	if shouldRoot {
		domains = []string{fmt.Sprintf("%s.%s", project, opts.Suffix)}
	}

	for _, extra := range prefs.Domains {
		if isPublicDomain(extra) {
			warnings = append(warnings, Warning{Code: "public_suffix_rejected", Message: "extra explicit domain rejected because suffix appears public", Field: "domains", Severity: "error", Source: extra})
			continue
		}
		if !strings.HasSuffix(extra, "."+opts.Suffix) {
			warnings = append(warnings, Warning{Code: "unmanaged_suffix", Message: "explicit domain uses unmanaged suffix", Field: "domains", Severity: "warning", Source: extra})
		}
		domains = append(domains, extra)
	}

	return domains, warnings
}

func isPublicDomain(domain string) bool {
	for _, suffix := range publicSuffixes {
		if strings.HasSuffix(strings.ToLower(domain), suffix) {
			return true
		}
	}
	parts := strings.Split(domain, ".")
	if len(parts) > 1 {
		tld := parts[len(parts)-1]
		if len(tld) == 2 {
			return true
		}
	}
	return false
}
