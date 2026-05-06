//go:build !darwin

package adminapi

import "fmt"

func activeGUIUserIDs() (int, int, error) {
	return 0, 0, fmt.Errorf("active GUI user resolution is only supported on darwin")
}
