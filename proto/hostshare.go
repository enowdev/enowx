package proto

import "time"

// A host shared with a team (docs/cloud.md, Hosts and sessions): its user
// picks which of its projects each team sees. The host then serves the
// team's members on its team listener, through the cloud's relay (and its
// Tailscale address when it has one), and announces itself in the team's
// space; the members' hosts list it.
const (
	MethodHostShare    = "host.share"     // → HostShare (the host's own connections)
	MethodHostShareSet = "host.share.set" // HostShareTeam (no Projects: unshared) → HostShare
	MethodTeamHosts    = "team.hosts"     // → []TeamHost, the hosts the host's teams' members share
)

// EventTeamHostsChanged carries the full []TeamHost after a team's space
// brought a change of them.
const EventTeamHostsChanged = "team.hosts.changed"

// TeamPort is the port the team listener takes, on the host's Tailscale
// address.
const TeamPort = 7178

// EnvTeamListen is an address the team listener takes instead (a test, a
// proxy of the user's).
const EnvTeamListen = "ENOWX_TEAM_LISTEN"

// HostShare is what the host shares, and where members reach it.
type HostShare struct {
	Teams []HostShareTeam `json:"teams"`
	// Addr is where the team listener listens, "" while it doesn't; Err
	// why it doesn't when something is shared.
	Addr string `json:"addr,omitempty"`
	Err  string `json:"err,omitempty"`
	// Relay says the host is connected to its cloud's relay, through which
	// members reach it with no address of its own.
	Relay bool `json:"relay,omitempty"`
}

// HostShareTeam is the projects (their folders) the host shares with Team.
type HostShareTeam struct {
	Team     string   `json:"team"`
	Projects []string `json:"projects"`
	// AllowSessions lets the team's members (not viewers) start sessions
	// of their own in the shared projects: they run as the host's user, so
	// it is off until the user turns it on. Left out of host.share.set, it
	// stays as it is; the host's answer always says it.
	AllowSessions *bool `json:"allowSessions,omitempty"`
}

// TeamHost is a host a member shares with a team, as its space says.
type TeamHost struct {
	Team   string `json:"team"`
	Device string `json:"device"` // the host's device in the cloud
	Name   string `json:"name"`
	Owner  string `json:"owner"` // the member sharing it
	Addr   string `json:"addr"`  // its Tailscale address, "" without one
	// Key is the host's public key, which a member's channel through the
	// cloud's relay opens to (cloud/relay); "" for a host of an older enowX.
	Key      string            `json:"key,omitempty"`
	Projects []TeamHostProject `json:"projects"`
	Updated  time.Time         `json:"updated"`
}

// TeamHostProject is a project a shared host shows the team.
type TeamHostProject struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Ref  string `json:"ref,omitempty"` // its Git remote's, as a note's project
}
