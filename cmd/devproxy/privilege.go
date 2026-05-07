package devproxy

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func ensureRoot(cmd *cobra.Command) error {
	if os.Geteuid() == 0 {
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve devproxy executable for privileged rerun: %w", err)
	}

	args := append([]string{exe}, os.Args[1:]...)
	reexec := exec.CommandContext(cmd.Context(), "sudo", args...)
	reexec.Stdin = cmd.InOrStdin()
	reexec.Stdout = cmd.OutOrStdout()
	reexec.Stderr = cmd.ErrOrStderr()
	if err := reexec.Run(); err != nil {
		return fmt.Errorf("sudo devproxy command failed: %w", err)
	}
	return errPrivilegedRerunComplete
}

var errPrivilegedRerunComplete = fmt.Errorf("privileged rerun completed")

func handledByPrivilegedRerun(err error) bool {
	return errors.Is(err, errPrivilegedRerunComplete)
}
