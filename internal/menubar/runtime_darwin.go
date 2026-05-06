//go:build darwin

package menubar

import (
	"context"
	"fmt"
	"time"

	"github.com/getlantern/systray"
)

func Run(ctx context.Context, client adminClient, op opener) error {
	if client == nil {
		return fmt.Errorf("menubar admin client is required")
	}
	if op == nil {
		op = NewOpener()
	}
	d := newDispatcher(client, op)

	ready := make(chan struct{})
	quit := make(chan struct{})

	var statusItem *systray.MenuItem
	var pauseItem *systray.MenuItem
	var dashboardItem *systray.MenuItem
	var logsItem *systray.MenuItem
	var doctorItem *systray.MenuItem
	var refreshItem *systray.MenuItem
	var startupItem *systray.MenuItem

	onReady := func() {
		systray.SetTemplateIcon(trayIcon, trayIcon)
		systray.SetTitle("DevProxy")
		statusItem = systray.AddMenuItem("Loading…", "DevProxy status")
		systray.AddSeparator()
		refreshItem = systray.AddMenuItem("Refresh Routes", "Refresh routes from daemon")
		pauseItem = systray.AddMenuItem("Pause Routing", "Pause or resume routing")
		dashboardItem = systray.AddMenuItem("Open Dashboard", "Open local dashboard")
		logsItem = systray.AddMenuItem("Open Logs", "Open local dashboard logs")
		doctorItem = systray.AddMenuItem("Run Doctor", "Run daemon doctor check")
		startupItem = systray.AddMenuItem("Start at Login (menubar)", "Toggle menubar startup role")
		quitItem := systray.AddMenuItem("Quit", "Quit DevProxy menu bar")
		close(ready)

		go func() {
			for {
				select {
				case <-refreshItem.ClickedCh:
					_ = d.refresh(context.Background())
				case <-pauseItem.ClickedCh:
					_ = d.togglePause(context.Background(), pauseItem.Title() == "Pause Routing")
				case <-dashboardItem.ClickedCh:
					_ = d.openDashboard(context.Background())
				case <-logsItem.ClickedCh:
					_ = d.openLogs(context.Background())
				case <-doctorItem.ClickedCh:
					_ = d.runDoctor(context.Background())
				case <-startupItem.ClickedCh:
					enabled := startupItem.Title() == "Enable Start at Login (menubar)"
					_ = d.toggleStartup(context.Background(), enabled)
				case <-quitItem.ClickedCh:
					systray.Quit()
					return
				case <-ctx.Done():
					systray.Quit()
					return
				}
			}
		}()

		go func() {
			ticker := time.NewTicker(3 * time.Second)
			defer ticker.Stop()
			for {
				if err := refreshMenu(context.Background(), client, statusItem, pauseItem, startupItem); err != nil {
					state := offlineMenuState(err)
					statusItem.SetTitle(state.HealthLine + " — " + state.ErrorLine)
					pauseItem.SetTitle("Pause Routing")
				}
				select {
				case <-ticker.C:
				case <-ctx.Done():
					systray.Quit()
					return
				}
			}
		}()
	}

	onExit := func() {
		close(quit)
	}

	go systray.Run(onReady, onExit)
	<-ready
	select {
	case <-ctx.Done():
		systray.Quit()
		<-quit
		return nil
	case <-quit:
		return nil
	}
}

func refreshMenu(ctx context.Context, client adminClient, statusItem, pauseItem, startupItem *systray.MenuItem) error {
	status, err := client.Status(ctx)
	if err != nil {
		return err
	}
	routes, err := client.Routes(ctx)
	if err != nil {
		return err
	}
	startup, err := client.StartupStatus(ctx)
	if err != nil {
		return err
	}
	state := buildMenuState(status, routes, startup.Roles)
	statusItem.SetTitle(fmt.Sprintf("%s | %s | %s", state.HealthLine, state.PauseLine, state.ActiveRoutesLine))
	if status.Paused {
		pauseItem.SetTitle("Resume Routing")
	} else {
		pauseItem.SetTitle("Pause Routing")
	}
	startupEnabled := false
	for _, role := range startup.Roles {
		if role.Role == startupRoleMenubar {
			startupEnabled = role.Installed
			break
		}
	}
	if startupEnabled {
		startupItem.SetTitle("Disable Start at Login (menubar)")
	} else {
		startupItem.SetTitle("Enable Start at Login (menubar)")
	}
	return nil
}
