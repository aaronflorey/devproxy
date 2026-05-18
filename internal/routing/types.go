package routing

import "time"

type SourceKind string

const (
	SourceComposeLabels SourceKind = "compose-labels"
	SourceContainerName SourceKind = "container-name"
)

type Candidate struct {
	ContainerID    string
	ContainerName  string
	Project        string
	Service        string
	Source         SourceKind
	PublishedPorts []PublishedPort
	Labels         map[string]string
	Warnings       []Warning
}

type Route struct {
	Hostname        string
	Domains         []string
	ServedHostnames []string
	Upstream        Upstream
	Winner          Candidate
	Losers          []Candidate
	Priority        int
	HTTPSRedirect   bool
	HTTPSOnly       bool
	Provenance      RouteProvenance
}

type RouteProvenance struct {
	MetadataSource SourceKind
	DomainSource   string
	PortSource     string
	PrioritySource string
}

type PublishedPort struct {
	ContainerPort int
	HostPort      int
	Protocol      string
	HostIP        string
}

type Upstream struct {
	Host   string
	Port   int
	Scheme string
}

type Warning struct {
	Code      string
	Message   string
	Container string
	Field     string
	Severity  string
	Source    string
}

type Conflict struct {
	Hostname    string
	Winner      Candidate
	Losers      []Candidate
	Reason      string
	PriorityTie bool
}

type Snapshot struct {
	Version   string
	CreatedAt time.Time
	Routes    map[string]Route
	Warnings  []Warning
	Conflicts []Conflict
}
