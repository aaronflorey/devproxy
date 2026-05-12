package routing

import "testing"

func TestDomainGeneration(t *testing.T) {
	opts := RouteOptions{Suffix: "test", RootServices: []string{"app", "web", "nginx", "laravel.test"}}

	domains, warnings := GenerateDomains("acme", "api", RoutePreferences{}, opts)
	if len(warnings) != 0 || len(domains) != 1 || domains[0] != "api.acme.test" {
		t.Fatalf("expected default domain, got domains=%v warnings=%v", domains, warnings)
	}

	domains, _ = GenerateDomains("acme", "app", RoutePreferences{}, opts)
	if len(domains) == 0 || domains[0] != "acme.test" {
		t.Fatalf("expected root domain for app, got %v", domains)
	}

	domains, warnings = GenerateDomains("acme", "api", RoutePreferences{Domain: "api.acme.local"}, opts)
	if len(domains) == 0 || domains[0] != "api.acme.local" || len(warnings) != 1 {
		t.Fatalf("expected unmanaged suffix warning for .local")
	}

	domains, warnings = GenerateDomains("acme", "api", RoutePreferences{Domain: "api.acme.com"}, opts)
	if len(domains) != 0 || len(warnings) == 0 {
		t.Fatalf("expected public suffix rejection")
	}

	rootFalse := false
	domains, warnings = GenerateDomains("acme", "app", RoutePreferences{Root: &rootFalse}, opts)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings for explicit root false, got %v", warnings)
	}
	if len(domains) != 1 || domains[0] != "app.acme.test" {
		t.Fatalf("expected explicit root false to suppress root domain, got %v", domains)
	}

	rootTrue := true
	domains, warnings = GenerateDomains("acme", "api", RoutePreferences{Root: &rootTrue}, opts)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings for explicit root true, got %v", warnings)
	}
	if len(domains) != 1 || domains[0] != "acme.test" {
		t.Fatalf("expected explicit root true to force root domain, got %v", domains)
	}
}
