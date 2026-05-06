//go:build !darwin

package install

import "fmt"

func ResolveGUIUser() (int, string, error) {
	return 0, "", fmt.Errorf("menubar installation is only supported on macOS")
}
