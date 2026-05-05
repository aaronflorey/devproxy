package admin

import "github.com/mochaka/devproxy/internal/routing"

type DoctorView struct {
	ConflictCount int
	WarningCount  int
	Warnings      []routing.Warning
	Conflicts     []routing.Conflict
}

func BuildDoctor(snapshot routing.Snapshot) DoctorView {
	return DoctorView{ConflictCount: len(snapshot.Conflicts), WarningCount: len(snapshot.Warnings), Warnings: append([]routing.Warning{}, snapshot.Warnings...), Conflicts: append([]routing.Conflict{}, snapshot.Conflicts...)}
}
