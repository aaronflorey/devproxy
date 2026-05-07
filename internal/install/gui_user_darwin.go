//go:build darwin

package install

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"syscall"
)

func ResolveGUIUser() (int, string, error) {
	uid, _, homeDir, err := ResolveGUIUserOwnership()
	if err != nil {
		return 0, "", err
	}
	return uid, homeDir, nil
}

func ResolveGUIUserOwnership() (int, int, string, error) {
	info, err := os.Stat("/dev/console")
	if err != nil {
		return 0, 0, "", fmt.Errorf("determine active desktop user from /dev/console: %w", err)
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, "", fmt.Errorf("determine active desktop user from /dev/console: unsupported stat data")
	}

	uid := int(stat.Uid)
	if uid <= 0 {
		return 0, 0, "", fmt.Errorf("no active GUI user session found; log into macOS desktop and re-run install --with-menubar as sudo")
	}

	u, err := user.LookupId(strconv.Itoa(uid))
	if err != nil {
		return 0, 0, "", fmt.Errorf("lookup GUI user uid %d: %w", uid, err)
	}
	if u.HomeDir == "" {
		return 0, 0, "", fmt.Errorf("GUI user uid %d has no home directory; cannot install menubar LaunchAgent", uid)
	}

	gid, err := strconv.Atoi(u.Gid)
	if err != nil || gid <= 0 {
		return 0, 0, "", fmt.Errorf("GUI user uid %d has invalid primary gid %q; cannot install menubar LaunchAgent", uid, u.Gid)
	}

	return uid, gid, u.HomeDir, nil
}
