package daemon

import (
	"context"
	"slices"
	"sync"
	"time"
)

var supportedDockerEvents = []string{"start", "stop", "die", "destroy", "rename", "update"}

type DockerEvent struct {
	Action string
}

type EventStream struct {
	Events <-chan DockerEvent
	Errors <-chan error
	close  func() error
}

func (s *EventStream) Close() error {
	if s == nil || s.close == nil {
		return nil
	}
	return s.close()
}

type DockerEventSource func(context.Context) (*EventStream, error)

type WatcherHealth struct {
	Connected         bool
	LastDisconnect    time.Time
	LastReconnectSync time.Time
}

type Watcher struct {
	mu         sync.RWMutex
	reconciler *Reconciler
	health     WatcherHealth
}

func NewWatcher(reconciler *Reconciler) *Watcher {
	return &Watcher{reconciler: reconciler, health: WatcherHealth{Connected: true}}
}

func (w *Watcher) OnDisconnect() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.health.Connected = false
	w.health.LastDisconnect = time.Now().UTC()
}

func (w *Watcher) OnReconnect(containers []ContainerState) error {
	if err := w.reconciler.RebuildSnapshot(containers); err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.health.Connected = true
	w.health.LastReconnectSync = time.Now().UTC()
	return nil
}

func (w *Watcher) HandleEvent(event string, containers []ContainerState) error {
	return w.reconciler.HandleEvent(event, containers)
}

func (w *Watcher) Health() WatcherHealth {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.health
}

func isSupportedDockerEvent(action string) bool {
	return slices.Contains(supportedDockerEvents, action)
}
