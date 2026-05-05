package admin

import "github.com/mochaka/devproxy/internal/routing"

type DoctorView struct {
	ConflictCount int
	WarningCount  int
	Warnings      []routing.Warning
	Conflicts     []routing.Conflict
	Network       NetworkDoctorStatus
}

type NetworkDoctorStatus struct {
	DNSHealthy       bool
	HTTPBound        bool
	HTTPSBound       bool
	Paused           bool
	CertificateReady bool
	ManagedSuffix    string
}

func BuildDoctor(snapshot routing.Snapshot, network NetworkDoctorStatus) DoctorView {
	return DoctorView{ConflictCount: len(snapshot.Conflicts), WarningCount: len(snapshot.Warnings), Warnings: append([]routing.Warning{}, snapshot.Warnings...), Conflicts: append([]routing.Conflict{}, snapshot.Conflicts...), Network: network}
}
