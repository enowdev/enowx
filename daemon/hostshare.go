package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/enowx/enowx/cloud/api"
	"github.com/enowx/enowx/cloud/relay"
	"github.com/enowx/enowx/core/projectctx"
	"github.com/enowx/enowx/core/tailnet"
	"github.com/enowx/enowx/proto"
)

// A host shared with its teams (docs/cloud.md, Hosts and sessions): the
// projects its user picks for each team, in share.json. While any is
// shared, the team listener serves the teams' members through the cloud's
// relay (and the host's Tailscale address, when it has one), and the host
// announces itself in each team's space
// (hosts/<its device>.json, sealed and synced with it), so the members'
// hosts list it. A visitor sees the shared projects of their team alone.

// listenTeam serves visitors on a listener: ListenTeam, set in init, as
// the handlers it reaches lead back here.
var listenTeam func(*Server, net.Listener) error

func init() { listenTeam = (*Server).ListenTeam }

// hostSharing is the share and the team listener; its mutex guards all.
type hostSharing struct {
	mu    sync.Mutex
	ln    net.Listener // the team listener: the relay's and Tailscale's (teamrelay.go)
	relay *relay.Host  // nil without a cloud signed in to
	key   string       // the public key members' channels open to
	addr  string       // the Tailscale address, "" without one
	err   string
}

type hostShareConfig struct {
	Teams map[string][]string `json:"teams"` // folders, by team
	// Sessions are the teams whose members may start sessions here.
	Sessions map[string]bool `json:"sessions,omitempty"`
}

func (s *Server) hostShareFile() string { return filepath.Join(s.dataDir, "share.json") }

func (s *Server) readShare() hostShareConfig {
	var f hostShareConfig
	if data, err := os.ReadFile(s.hostShareFile()); err == nil {
		_ = json.Unmarshal(data, &f)
	}
	if f.Teams == nil {
		f.Teams = map[string][]string{}
	}
	return f
}

// sharedProjects are the folders the host shares with team.
func (s *Server) sharedProjects(team string) []string { return s.readShare().Teams[team] }

// shareAllowsSessions says team's members may start sessions here.
func (s *Server) shareAllowsSessions(team string) bool {
	f := s.readShare()
	return len(f.Teams[team]) > 0 && f.Sessions[team]
}

// hostShare is what the host shares, and where the listener is.
func (s *Server) hostShare() proto.HostShare {
	out := proto.HostShare{Teams: []proto.HostShareTeam{}}
	f := s.readShare()
	for _, sp := range s.spaces() {
		if list := f.Teams[sp.Team]; len(list) > 0 {
			allow := f.Sessions[sp.Team]
			out.Teams = append(out.Teams, proto.HostShareTeam{Team: sp.Team, Projects: list, AllowSessions: &allow})
		}
	}
	s.share.mu.Lock()
	out.Addr, out.Err = s.share.addr, s.share.err
	if s.share.relay != nil {
		up, why := s.share.relay.Status()
		out.Relay = up
		if !up && why != "" && out.Err == "" {
			out.Err = "the relay: " + why
		}
	}
	s.share.mu.Unlock()
	return out
}

// setHostShare shares projects with a team, or nothing.
func (s *Server) setHostShare(p proto.HostShareTeam) (proto.HostShare, error) {
	sp, ok := s.spaceByTeam(p.Team)
	if !ok {
		return proto.HostShare{}, fmt.Errorf("spaces: team %s isn't on this host", p.Team)
	}
	if sp.Role == api.RoleViewer {
		return proto.HostShare{}, fmt.Errorf("teams: as a viewer of %s you can't share a host with it", sp.Name)
	}
	known := s.projects.List()
	var paths []string
	for _, path := range p.Projects {
		if !slices.ContainsFunc(known, func(x proto.ProjectInfo) bool { return x.Path == path }) {
			return proto.HostShare{}, fmt.Errorf("teams: %s isn't a project of this host", path)
		}
		if !slices.Contains(paths, path) {
			paths = append(paths, path)
		}
	}
	f := s.readShare()
	if len(paths) == 0 {
		delete(f.Teams, p.Team)
		delete(f.Sessions, p.Team) // shared again, it starts off
	} else {
		f.Teams[p.Team] = paths
		if p.AllowSessions != nil {
			if f.Sessions == nil {
				f.Sessions = map[string]bool{}
			}
			if *p.AllowSessions {
				f.Sessions[p.Team] = true
			} else {
				delete(f.Sessions, p.Team)
			}
		}
	}
	data, _ := json.MarshalIndent(f, "", "  ")
	if err := writeTeamFile(s.hostShareFile(), data); err != nil {
		return proto.HostShare{}, err
	}
	s.startSharing()
	s.enforceScopes() // what visitors may no longer see goes now
	return s.hostShare(), nil
}

// startSharing opens the team listener while something is shared, closes
// it while nothing is, and announces the host in its teams' spaces.
func (s *Server) startSharing() {
	shared := false
	for _, list := range s.readShare().Teams {
		shared = shared || len(list) > 0
	}
	s.share.mu.Lock()
	switch {
	case shared && s.share.ln == nil:
		fan := newFanIn()
		var why []string
		h, key, err := s.startRelay()
		if err != nil {
			why = append(why, "the relay: "+err.Error())
		} else if h != nil {
			fan.add(h)
		}
		// Tailscale too, when the host has it: members of an older enowX
		// know no other way.
		addr := os.Getenv(proto.EnvTeamListen)
		if addr == "" {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if ip, err := tailnet.SelfIP(ctx); err == nil && ip != "" {
				addr = net.JoinHostPort(ip, strconv.Itoa(proto.TeamPort))
			}
			cancel()
		}
		var ln net.Listener
		if addr != "" {
			if ln, err = net.Listen("tcp", addr); err != nil {
				why = append(why, "the team listener: "+err.Error())
			} else {
				fan.add(ln)
			}
		}
		if h == nil && ln == nil {
			if len(why) == 0 {
				why = append(why, "sign this host in to the team's cloud: members reach it through the cloud's relay")
			}
			s.share.err = strings.Join(why, "; ")
			break
		}
		s.share.ln, s.share.relay, s.share.key, s.share.err = fan, h, key, strings.Join(why, "; ")
		if ln != nil {
			s.share.addr = ln.Addr().String()
		}
		go listenTeam(s, fan)
	case !shared && s.share.ln != nil:
		s.share.ln.Close()
		s.share.ln, s.share.relay, s.share.key, s.share.addr, s.share.err = nil, nil, "", "", ""
	case !shared:
		s.share.err = ""
	}
	s.share.mu.Unlock()
	s.announceHost()
}

// stopSharing closes the team listener, as the daemon stops.
func (s *Server) stopSharing() {
	s.share.mu.Lock()
	defer s.share.mu.Unlock()
	if s.share.ln != nil {
		s.share.ln.Close()
	}
	s.share.ln, s.share.relay, s.share.key, s.share.addr = nil, nil, "", ""
}

// announceHost writes, or takes back, the host's entry in each of its
// teams' spaces.
func (s *Server) announceHost() {
	st, err := s.cloud.store.Get()
	if err != nil || !st.SignedIn() || st.DeviceID == "" {
		return
	}
	f := s.readShare()
	s.share.mu.Lock()
	addr, key := s.share.addr, s.share.key
	s.share.mu.Unlock()
	for _, sp := range s.spaces() {
		file := filepath.Join(s.teamsDir(), sp.Team, "hosts", st.DeviceID+".json")
		list := f.Teams[sp.Team]
		if len(list) == 0 || addr == "" && key == "" {
			if os.Remove(file) == nil {
				s.teamSync.kick()
			}
			continue
		}
		h := proto.TeamHost{Team: sp.Team, Device: st.DeviceID, Name: s.userdata.Name, Owner: st.Account, Addr: addr, Key: key, Projects: []proto.TeamHostProject{}}
		for _, path := range list {
			h.Projects = append(h.Projects, proto.TeamHostProject{Name: filepath.Base(path), Path: path, Ref: s.remoteRef(path)})
		}
		var old proto.TeamHost
		if data, err := os.ReadFile(file); err == nil && json.Unmarshal(data, &old) == nil {
			h.Updated = old.Updated
			if same, _ := json.Marshal(h); string(same) == string(data) {
				continue
			}
		}
		h.Updated = time.Now().UTC().Truncate(time.Second)
		data, _ := json.Marshal(h)
		if writeTeamFile(file, data) == nil {
			s.teamSync.kick()
		}
	}
}

// teamHosts are the hosts members share with the host's teams, but its own.
func (s *Server) teamHosts() []proto.TeamHost {
	st, _ := s.cloud.store.Get()
	out := []proto.TeamHost{}
	for _, sp := range s.spaces() {
		entries, _ := os.ReadDir(filepath.Join(s.teamsDir(), sp.Team, "hosts"))
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".json") || strings.TrimSuffix(e.Name(), ".json") == st.DeviceID {
				continue
			}
			var h proto.TeamHost
			data, err := os.ReadFile(filepath.Join(s.teamsDir(), sp.Team, "hosts", e.Name()))
			if err != nil || json.Unmarshal(data, &h) != nil || h.Device != strings.TrimSuffix(e.Name(), ".json") {
				continue
			}
			h.Team = sp.Team
			out = append(out, h)
		}
	}
	return out
}

// visitorScope are the folders a visitor sees: their team's shared ones.
func (c *conn) visitorScope() []string {
	v := c.visitor.Load()
	if v == nil {
		return nil
	}
	return c.srv.sharedProjects(v.Team)
}

// inScope says folder dir is within one of the visitor's shared projects,
// by its path ("..", cleaned, can't leave one).
func inScope(scope []string, dir string) bool {
	if dir == "" {
		return false
	}
	dir = filepath.Clean(dir)
	return slices.ContainsFunc(scope, func(p string) bool { return within(dir, filepath.Clean(p)) })
}

// sharedFolder is folder dir resolved, links and all, when it is within
// one of the shared projects of scope, also resolved: a link in a shared
// project leads nowhere out of it.
func sharedFolder(scope []string, dir string) (string, bool) {
	if !inScope(scope, dir) {
		return "", false
	}
	real, err := filepath.EvalSymlinks(filepath.Clean(dir))
	if err != nil {
		return "", false
	}
	for _, p := range scope {
		if root, err := filepath.EvalSymlinks(filepath.Clean(p)); err == nil && within(real, root) {
			return real, true
		}
	}
	return "", false
}

var errNotShared = errors.New("that isn't shared with your team on this host")

// visitorOverrides answer a visitor in place of the host's own handlers:
// the same, within their team's shared projects.
var visitorOverrides = map[string]handler{
	proto.MethodProjectList: bare(func(c *conn) (any, error) {
		scope := c.visitorScope()
		out := []proto.ProjectInfo{}
		for _, p := range c.srv.projects.List() {
			if slices.Contains(scope, p.Path) {
				p.Group = ""
				out = append(out, p)
			}
		}
		return out, nil
	}),
	proto.MethodSessionList: bare(func(c *conn) (any, error) {
		scope := c.visitorScope()
		out := []proto.SessionInfo{}
		for _, sess := range c.srv.sessions.List() {
			if info := sess.Info(); visibleTo(info, scope) {
				out = append(out, forVisitor(info))
			}
		}
		return out, nil
	}),
	proto.MethodProjectContext: with(func(c *conn, p proto.ProjectContextParams) (any, error) {
		dir, ok := sharedFolder(c.visitorScope(), homePath(p.Path))
		if p.Host != "" && p.Host != c.srv.daemonID || !ok {
			return nil, errNotShared
		}
		out, err := c.srv.projectContext(proto.ProjectContextParams{Path: dir})
		if err == nil {
			out.Path = filepath.Clean(homePath(p.Path)) // as they named it
		}
		return out, err
	}),
	proto.MethodProjectRead: with(func(c *conn, p proto.ProjectReadParams) (any, error) {
		dir, ok := sharedFolder(c.visitorScope(), homePath(p.Path))
		if p.Host != "" && p.Host != c.srv.daemonID || !ok {
			return nil, errNotShared
		}
		return projectctx.Read(dir, p.File)
	}),
}

var hostShareHandlers = map[string]handler{
	proto.MethodHostShare: bare(func(c *conn) (any, error) {
		if c.remote {
			return nil, errors.New("only the host's own connections change what it shares")
		}
		return c.srv.hostShare(), nil
	}),
	proto.MethodHostShareSet: with(func(c *conn, p proto.HostShareTeam) (any, error) {
		if c.remote {
			return nil, errors.New("only the host's own connections change what it shares")
		}
		return c.srv.setHostShare(p)
	}),
	proto.MethodTeamHosts: bare(func(c *conn) (any, error) {
		if c.remote {
			return nil, errors.New("only the host's own connections list its teams' hosts")
		}
		return c.srv.teamHosts(), nil
	}),
}
