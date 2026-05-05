package daemon

import "time"

type WatcherHealth struct {
	Connected         bool
	LastDisconnect    time.Time
	LastReconnectSync time.Time
}

type Watcher struct {
	reconciler *Reconciler
	health     WatcherHealth
}

func NewWatcher(reconciler *Reconciler) *Watcher {
	return &Watcher{reconciler: reconciler, health: WatcherHealth{Connected: true}}
}

func (w *Watcher) OnDisconnect() {
	w.health.Connected = false
	w.health.LastDisconnect = time.Now().UTC()
}

func (w *Watcher) OnReconnect(containers []ContainerState) {
	_ = w.reconciler.RebuildSnapshot(containers)
	w.health.Connected = true
	w.health.LastReconnectSync = time.Now().UTC()
}

func (w *Watcher) Health() WatcherHealth {
	return w.health
}
