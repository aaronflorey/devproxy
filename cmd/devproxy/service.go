package devproxy

import (
	"context"
	"fmt"

	"github.com/mochaka/devproxy/internal/install"
	"github.com/spf13/cobra"
)

func init() {
	registerCommandFactory(newStartCommand)
	registerCommandFactory(newStopCommand)
}

func newStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the installed devproxy daemon service",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args
			if err := ensureRoot(cmd); err != nil {
				if handledByPrivilegedRerun(err) {
					return nil
				}
				return err
			}
			cfg := install.DaemonServiceConfig(install.DefaultPaths(), "")
			if err := install.StartService(cfg); err != nil {
				return fmt.Errorf("start daemon service: %w", err)
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "devproxy daemon started")
			return err
		},
	}
}

func newStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the installed devproxy daemon service",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args
			if err := ensureRoot(cmd); err != nil {
				if handledByPrivilegedRerun(err) {
					return nil
				}
				return err
			}
			cfg := install.DaemonServiceConfig(install.DefaultPaths(), "")
			if err := install.StopService(context.Background(), cfg); err != nil {
				return fmt.Errorf("stop daemon service: %w", err)
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "devproxy daemon stopped")
			return err
		},
	}
}
