package devproxy

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mochaka/devproxy/internal/install"
	"github.com/spf13/cobra"
)

func init() {
	registerCommandFactory(newUninstallCommand)
}

func newUninstallCommand() *cobra.Command {
	var withMenubar bool
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall devproxy services and optional local artifacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args
			if os.Geteuid() != 0 {
				return fmt.Errorf("devproxy uninstall requires root privileges; rerun with sudo")
			}
			scope, err := promptCleanupScope(cmd.InOrStdin(), cmd.OutOrStdout())
			if err != nil {
				return err
			}
			uninstaller := install.NewUninstaller(install.UninstallDependencies{})
			if err := uninstaller.Uninstall(cmd.Context(), install.UninstallOptions{
				Suffix:      loadedCfg.DomainSuffix,
				WithMenubar: withMenubar,
				Cleanup:     scope,
			}); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "devproxy uninstall completed")
			return err
		},
	}
	cmd.Flags().BoolVar(&withMenubar, "with-menubar", false, "also unregister optional menu bar LaunchAgent")
	return cmd
}

func promptCleanupScope(in io.Reader, out io.Writer) (install.CleanupScope, error) {
	reader := bufio.NewReader(in)
	task := func(label string) (bool, error) {
		if _, err := fmt.Fprintf(out, "Remove %s? [y/N]: ", label); err != nil {
			return false, err
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}
		response := strings.ToLower(strings.TrimSpace(line))
		return response == "y" || response == "yes", nil
	}

	config, err := task("config")
	if err != nil {
		return install.CleanupScope{}, err
	}
	state, err := task("state")
	if err != nil {
		return install.CleanupScope{}, err
	}
	logs, err := task("logs")
	if err != nil {
		return install.CleanupScope{}, err
	}
	certs, err := task("certificates")
	if err != nil {
		return install.CleanupScope{}, err
	}

	return install.CleanupScope{Config: config, State: state, Logs: logs, Certificates: certs}, nil
}
