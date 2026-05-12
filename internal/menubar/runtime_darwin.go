//go:build darwin

package menubar

import (
	"context"
	"fmt"
	"time"

	"github.com/getlantern/systray"
)

var (
	runSystray  = systray.Run
	quitSystray = systray.Quit
)

type routeSlot struct {
	item    *systray.MenuItem
	host    string
	openURL string
}

type runtimeState struct {
	paused         bool
	startupEnabled bool
}

type routeSlotAssignment struct {
	visible bool
	host    string
	openURL string
}

func computeRouteSlotAssignments(slotCount int, routes []routeMenuItem) []routeSlotAssignment {
	if slotCount < len(routes) {
		slotCount = len(routes)
	}
	assignments := make([]routeSlotAssignment, slotCount)
	for i := 0; i < slotCount; i++ {
		if i >= len(routes) {
			continue
		}
		assignments[i] = routeSlotAssignment{visible: true, host: routes[i].Hostname, openURL: routes[i].OpenURL}
	}
	return assignments
}

func syncRouteSlots(slots []routeSlot, routes []routeMenuItem, create func() *systray.MenuItem, bindClick func(*routeSlot)) []routeSlot {
	assignments := computeRouteSlotAssignments(len(slots), routes)
	for len(slots) < len(assignments) {
		slots = append(slots, routeSlot{item: create()})
		bindClick(&slots[len(slots)-1])
	}
	for i, assignment := range assignments {
		if assignment.visible {
			slots[i].host = assignment.host
			slots[i].openURL = assignment.openURL
			slots[i].item.SetTitle(assignment.host)
			slots[i].item.Show()
			continue
		}
		slots[i].host = ""
		slots[i].openURL = ""
		slots[i].item.Hide()
	}
	return slots
}

func Run(ctx context.Context, client adminClient, op opener) error {
	if client == nil {
		return fmt.Errorf("menubar admin client is required")
	}
	if op == nil {
		op = NewOpener()
	}
	d := newDispatcher(client, op)

	quit := make(chan struct{})

	var statusItem *systray.MenuItem
	var pauseItem *systray.MenuItem
	var dashboardItem *systray.MenuItem
	var logsItem *systray.MenuItem
	var doctorItem *systray.MenuItem
	var refreshItem *systray.MenuItem
	var startupItem *systray.MenuItem
	var routeSlots []routeSlot
	var routeClickCh chan string

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
		systray.AddSeparator()
		routeSlotFactory := func() *systray.MenuItem {
			item := systray.AddMenuItem("", "Open active route")
			item.Hide()
			return item
		}
		routeClickCh = make(chan string)
		routeClickBinder := func(slot *routeSlot) {
			go func(s *routeSlot) {
				for range s.item.ClickedCh {
					routeClickCh <- s.openURL
				}
			}(slot)
		}
		stateInfo, err := refreshMenu(context.Background(), client, statusItem, pauseItem, startupItem, &routeSlots, routeSlotFactory, routeClickBinder)
		if err != nil {
			state := offlineMenuState(err)
			statusItem.SetTitle(state.HealthLine + " — " + state.ErrorLine)
			pauseItem.SetTitle("Pause Routing")
			stateInfo = runtimeState{}
		}
		quitItem := systray.AddMenuItem("Quit", "Quit DevProxy menu bar")
		go func() {
			paused := stateInfo.paused
			startupEnabled := stateInfo.startupEnabled
			for {
				select {
				case routeURL := <-routeClickCh:
					if routeURL != "" {
						_ = d.openRoute(context.Background(), routeURL)
					}
				case <-refreshItem.ClickedCh:
					_ = d.refresh(context.Background())
				case <-pauseItem.ClickedCh:
					_ = d.togglePause(context.Background(), !paused)
				case <-dashboardItem.ClickedCh:
					_ = d.openDashboard(context.Background())
				case <-logsItem.ClickedCh:
					_ = d.openLogs(context.Background())
				case <-doctorItem.ClickedCh:
					_ = d.runDoctor(context.Background())
				case <-startupItem.ClickedCh:
					_ = d.toggleStartup(context.Background(), !startupEnabled)
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
				_, err := refreshMenu(context.Background(), client, statusItem, pauseItem, startupItem, &routeSlots, routeSlotFactory, routeClickBinder)
				if err != nil {
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

	runContextBoundSystray(ctx, onReady, onExit, runSystray, quitSystray)
	<-quit
	return nil
}

func runContextBoundSystray(ctx context.Context, onReady, onExit func(), run func(func(), func()), quit func()) {
	go func() {
		<-ctx.Done()
		quit()
	}()

	// macOS status bar UI must own the main thread; do not move this into a goroutine.
	run(onReady, onExit)
}

func refreshMenu(ctx context.Context, client adminClient, statusItem, pauseItem, startupItem *systray.MenuItem, routeSlots *[]routeSlot, create func() *systray.MenuItem, bindClick func(*routeSlot)) (runtimeState, error) {
	status, err := client.Status(ctx)
	if err != nil {
		return runtimeState{}, err
	}
	routes, err := client.Routes(ctx)
	if err != nil {
		return runtimeState{}, err
	}
	startup, err := client.StartupStatus(ctx)
	if err != nil {
		return runtimeState{}, err
	}
	state := buildMenuState(status, routes, startup.Roles)
	*routeSlots = syncRouteSlots(*routeSlots, state.RouteItems, create, bindClick)
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
	return runtimeState{paused: status.Paused, startupEnabled: startupEnabled}, nil
}
