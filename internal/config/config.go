package config

type Config struct {
	DomainSuffix        string                   `mapstructure:"domain_suffix"`
	RootServices        []string                 `mapstructure:"root_services"`
	IgnoredServices     []string                 `mapstructure:"ignored_services"`
	IgnoredPorts        []int                    `mapstructure:"ignored_ports"`
	PortPreferenceOrder []int                    `mapstructure:"port_preference_order"`
	Serving             ServingConfig            `mapstructure:"serving"`
	Overrides           map[string]ProjectConfig `mapstructure:"overrides"`
}

type ServingConfig struct {
	ManagedSuffix       string `mapstructure:"managed_suffix"`
	RedirectHTTPToHTTPS bool   `mapstructure:"redirect_http_to_https"`
}

type ProjectConfig struct {
	Services map[string]ServiceOverride `mapstructure:"services"`
}

type ServiceOverride struct {
	Enable   *bool    `mapstructure:"enable"`
	Domain   string   `mapstructure:"domain"`
	Domains  []string `mapstructure:"domains"`
	Root     *bool    `mapstructure:"root"`
	Port     *int     `mapstructure:"port"`
	Scheme   string   `mapstructure:"scheme"`
	Priority *int     `mapstructure:"priority"`
}

func DefaultConfig() Config {
	return Config{
		DomainSuffix:        "test",
		RootServices:        []string{"app", "web", "nginx", "laravel.test"},
		IgnoredServices:     []string{"mysql", "mariadb", "postgres", "redis", "memcached", "meilisearch", "selenium"},
		IgnoredPorts:        []int{3306, 5432, 6379, 9200, 11211},
		PortPreferenceOrder: []int{443, 8443, 80, 8080, 8000, 3000, 5173, 8025},
		Serving:             ServingConfig{ManagedSuffix: "test", RedirectHTTPToHTTPS: false},
		Overrides:           map[string]ProjectConfig{},
	}
}
