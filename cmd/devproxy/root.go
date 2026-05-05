package devproxy

import (
	"fmt"
	"strings"

	"github.com/mochaka/devproxy/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configPath string
	loadedCfg  config.Config
)

func Execute() error {
	return NewRootCommand().Execute()
}

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "devproxy",
		Short: "Local vanity domains for Docker Compose services",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(configPath)
			if err != nil {
				return err
			}
			loadedCfg = cfg
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&configPath, "config", "", "path to config file")

	registerCommands(cmd,
		newConfigCommand,
		newDaemonCommand,
	)

	return cmd
}

func newConfigCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "print-config",
		Short: "Print the effective config",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "suffix=%s root_services=%s\n", loadedCfg.DomainSuffix, strings.Join(loadedCfg.RootServices, ","))
			return err
		},
	}
}

func loadConfig(path string) (config.Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetEnvPrefix("DEVPROXY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	defaults := config.DefaultConfig()
	v.SetDefault("domain_suffix", defaults.DomainSuffix)
	v.SetDefault("root_services", defaults.RootServices)
	v.SetDefault("ignored_services", defaults.IgnoredServices)
	v.SetDefault("ignored_ports", defaults.IgnoredPorts)
	v.SetDefault("port_preference_order", defaults.PortPreferenceOrder)

	if path != "" {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			return config.Config{}, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg config.Config
	if err := v.Unmarshal(&cfg); err != nil {
		return config.Config{}, fmt.Errorf("decode config: %w", err)
	}

	return cfg, nil
}
