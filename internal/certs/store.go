package certs

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mochaka/devproxy/internal/routing"
)

type StoredCertificate struct {
	ProjectRoot string
	SANs        []string
	CertPath    string
	KeyPath     string
}

type InventoryDecision struct {
	ProjectRoot   string
	Hostnames     []string
	RequiredSANs  []string
	ReuseExisting bool
	Reason        string
}

func BuildCertificateInventory(snapshot routing.Snapshot, suffix string, existing map[string]StoredCertificate) []InventoryDecision {
	projectHosts := map[string]map[string]struct{}{}
	for _, route := range snapshot.Routes {
		hosts := route.ServedHostnames
		if len(hosts) == 0 && route.Hostname != "" {
			hosts = []string{route.Hostname}
		}
		for _, host := range hosts {
			normalized := normalizeHostname(host)
			if normalized == "" || !isManagedHostname(normalized, suffix) {
				continue
			}
			root, ok := projectRootForHostname(normalized, suffix)
			if !ok {
				continue
			}
			if _, ok := projectHosts[root]; !ok {
				projectHosts[root] = map[string]struct{}{}
			}
			projectHosts[root][normalized] = struct{}{}
		}
	}

	roots := make([]string, 0, len(projectHosts))
	for root := range projectHosts {
		roots = append(roots, root)
	}
	sort.Strings(roots)

	out := make([]InventoryDecision, 0, len(roots))
	for _, root := range roots {
		hosts := keys(projectHosts[root])
		sans := requiredSANs(root, hosts)
		decision := InventoryDecision{ProjectRoot: root, Hostnames: hosts, RequiredSANs: sans, ReuseExisting: false}

		stored, ok := existing[root]
		if !ok {
			decision.Reason = "new project root requires issuance"
			out = append(out, decision)
			continue
		}

		if coversAll(stored.SANs, hosts) {
			decision.ReuseExisting = true
			decision.Reason = "existing certificate covers active served hostnames"
		} else {
			decision.Reason = "served hostname shape changed beyond current coverage"
		}
		out = append(out, decision)
	}

	return out
}

func requiredSANs(projectRoot string, hosts []string) []string {
	if wildcardEligible(projectRoot, hosts) {
		return []string{projectRoot, fmt.Sprintf("*.%s", projectRoot)}
	}
	seen := map[string]struct{}{projectRoot: {}}
	out := []string{projectRoot}
	for _, h := range hosts {
		if _, ok := seen[h]; ok {
			continue
		}
		seen[h] = struct{}{}
		out = append(out, h)
	}
	sort.Strings(out)
	return out
}

func wildcardEligible(projectRoot string, hosts []string) bool {
	rootLabels := strings.Split(projectRoot, ".")
	for _, host := range hosts {
		if host == projectRoot {
			continue
		}
		labels := strings.Split(host, ".")
		if len(labels) != len(rootLabels)+1 {
			return false
		}
		if !strings.HasSuffix(host, "."+projectRoot) {
			return false
		}
	}
	return true
}

func coversAll(sans []string, hosts []string) bool {
	for _, host := range hosts {
		if !isCovered(host, sans) {
			return false
		}
	}
	return true
}

func isCovered(host string, sans []string) bool {
	for _, san := range sans {
		san = normalizeHostname(san)
		if san == host {
			return true
		}
		if strings.HasPrefix(san, "*.") {
			root := strings.TrimPrefix(san, "*.")
			if wildcardMatches(host, root) {
				return true
			}
		}
	}
	return false
}

func wildcardMatches(host, root string) bool {
	if !strings.HasSuffix(host, "."+root) {
		return false
	}
	hostLabels := strings.Split(host, ".")
	rootLabels := strings.Split(root, ".")
	return len(hostLabels) == len(rootLabels)+1
}

func projectRootForHostname(hostname, suffix string) (string, bool) {
	parts := strings.Split(hostname, ".")
	if len(parts) < 2 {
		return "", false
	}
	sfx := normalizeSuffix(suffix)
	if parts[len(parts)-1] != sfx {
		return "", false
	}
	project := parts[len(parts)-2]
	if project == "" {
		return "", false
	}
	return project + "." + sfx, true
}

func isManagedHostname(hostname, suffix string) bool {
	sfx := normalizeSuffix(suffix)
	return hostname == sfx || strings.HasSuffix(hostname, "."+sfx)
}

func normalizeSuffix(suffix string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(suffix)), ".")
}

func normalizeHostname(host string) string {
	return strings.Trim(strings.ToLower(strings.TrimSpace(host)), ".")
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
