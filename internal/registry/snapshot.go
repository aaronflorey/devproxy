package registry

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/mochaka/devproxy/internal/routing"
)

type Builder struct {
	seq atomic.Uint64
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) Build(routes []routing.Route, conflicts []routing.Conflict, warnings []routing.Warning) routing.Snapshot {
	seq := b.seq.Add(1)
	byHost := make(map[string]routing.Route, len(routes))
	for _, route := range routes {
		byHost[route.Hostname] = route
	}

	return routing.Snapshot{
		Version:   fmt.Sprintf("snapshot-%d", seq),
		CreatedAt: time.Now().UTC(),
		Routes:    byHost,
		Conflicts: append([]routing.Conflict{}, conflicts...),
		Warnings:  append([]routing.Warning{}, warnings...),
	}
}
