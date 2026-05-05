package config

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DomainSuffix != "test" {
		t.Fatalf("expected suffix test, got %q", cfg.DomainSuffix)
	}

	assertStringSliceEqual(t, cfg.RootServices, []string{"app", "web", "nginx", "laravel.test"})
	assertStringSliceEqual(t, cfg.IgnoredServices, []string{"mysql", "mariadb", "postgres", "redis", "memcached", "meilisearch", "selenium"})
	assertIntSliceEqual(t, cfg.IgnoredPorts, []int{3306, 5432, 6379, 9200, 11211})
	assertIntSliceEqual(t, cfg.PortPreferenceOrder, []int{443, 8443, 80, 8080, 8000, 3000, 5173, 8025})

	if cfg.Serving.ManagedSuffix != "test" {
		t.Fatalf("expected managed suffix test, got %q", cfg.Serving.ManagedSuffix)
	}

	if cfg.Serving.RedirectHTTPToHTTPS {
		t.Fatalf("expected redirect disabled by default")
	}
}

func assertStringSliceEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length mismatch got=%d want=%d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d mismatch got=%q want=%q", i, got[i], want[i])
		}
	}
}

func assertIntSliceEqual(t *testing.T, got, want []int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length mismatch got=%d want=%d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d mismatch got=%d want=%d", i, got[i], want[i])
		}
	}
}
