//go:build !darwin

package menubar

import (
	"context"
	"fmt"
)

func Run(context.Context, adminClient, opener) error {
	return fmt.Errorf("menubar is only supported on macOS")
}
