package api

import "time"

// A member's way into a host of their team (docs/cloud.md, Hosts and
// sessions): one of their devices asks the cloud for a ticket for that
// host, and hands it to the host when it connects; the host redeems it
// with its own token and learns who connects, and their role. A ticket
// works once, for that host only, within minutes; no token leaves the
// device that holds it.
const (
	PathHostTickets  = "/v1/teams/{team}/tickets" // POST HostTicketParams → HostTicket
	PathTicketRedeem = "/v1/tickets/redeem"       // POST RedeemParams → Visitor, by the ticket's host
)

// TicketLife is how long a ticket works.
const TicketLife = 2 * time.Minute

// HostTicketParams asks a ticket into host device Host.
type HostTicketParams struct {
	Host string `json:"host"`
}

// HostTicket is a ticket ("enxt_…"), shown once.
type HostTicket struct {
	Ticket  string    `json:"ticket"`
	Expires time.Time `json:"expires"`
	// Relay is where the member connects to reach the host through the
	// cloud's relay (wss://…/v1/relay/connect), "" for a cloud without one.
	Relay string `json:"relay,omitempty"`
}

// RedeemParams redeems a ticket handed to the host.
type RedeemParams struct {
	Ticket string `json:"ticket"`
}

// Visitor is who a redeemed ticket let in: a member of Team with Role,
// from their device Device.
type Visitor struct {
	Account Account `json:"account"`
	Team    string  `json:"team"`
	Role    string  `json:"role"`
	Device  string  `json:"device"`
}

// The relay (docs/cloud.md, Hosts and sessions): a host sharing projects
// keeps a connection open to it (PathRelayHost, its own token), so it needs
// no address members can reach. A member connects with their ticket
// (PathRelayConnect, the ticket in HeaderTicket); the relay tells the host
// (a RelayCall), the host opens a connection for it (PathRelayAccept), and
// the relay passes bytes between the two, which are end to end encrypted
// (cloud/relay): it sees who connects to whom and how much, not what.
// All three are WebSocket connections.
const (
	PathRelayHost    = "/v1/relay/host"        // the host's, its token: RelayCall messages
	PathRelayAccept  = "/v1/relay/accept/{id}" // the host's, its token: a call's bytes
	PathRelayConnect = "/v1/relay/connect"     // the member's, HeaderTicket: the host's bytes
	HeaderTicket     = "X-Enowx-Ticket"
)

// RelayCall tells the host a member waits on the relay: it opens
// PathRelayAccept with ID. From is the address the member came from, for
// the host's limits by address.
type RelayCall struct {
	ID   string `json:"id"`
	From string `json:"from"`
}
