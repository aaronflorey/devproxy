//go:build darwin

package adminapi

import (
	"fmt"
	"os"
	"syscall"
)

var osStat = os.Stat

func activeGUIUserIDs() (int, int, error) {
	info, err := osStat("/dev/console")
	if err != nil {
		return 0, 0, fmt.Errorf("determine active GUI user from /dev/console: %w", err)
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, fmt.Errorf("determine active GUI user from /dev/console: unsupported stat data")
	}

	uid := int(stat.Uid)
	gid := int(stat.Gid)
	if uid <= 0 || gid <= 0 {
		return 0, 0, fmt.Errorf("no active GUI user session found; unable to set admin socket ownership")
	}

	return uid, gid, nil
}
