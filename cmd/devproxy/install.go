package devproxy

import (
	"fmt"

	"github.com/mochaka/devproxy/internal/install"
	"github.com/spf13/cobra"
)

func init() {
	registerCommandFactory(newInstallCommand)
}

func newInstallCommand() *cobra.Command {
	var withMenubar bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install devproxy lifecycle integration on macOS",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args
			if err := ensureRoot(cmd); err != nil {
				if handledByPrivilegedRerun(err) {
					return nil
				}
				return err
			}
			cfg := loadedCfg
			if cfg.DomainSuffix == "" {
				return fmt.Errorf("config domain_suffix is required")
			}

			installer := install.NewInstaller(install.Dependencies{})
			return installer.Install(cmd.Context(), install.Options{
				Suffix:      cfg.DomainSuffix,
				WithMenubar: withMenubar,
				Progress: func(step string) {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "==> %s\n", step)
				},
			})
		},
	}

	cmd.Flags().BoolVar(&withMenubar, "with-menubar", false, "also install optional menu bar LaunchAgent")
	return cmd
}
