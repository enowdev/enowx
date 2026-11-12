package daemon

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/enowx/enowx/client"
	"github.com/enowx/enowx/cloud/api"
	"github.com/enowx/enowx/cloud/relay"
	"github.com/enowx/enowx/cloud/server"
	"github.com/enowx/enowx/cloud/store/storetest"
	"github.com/enowx/enowx/proto"
)

// A member reaches a host shared with their team through the cloud's relay,
// with no address of the host's: the host signs them in with their ticket
// as over any path, and lets go of them once they leave the team.
func TestVisitorThroughTheRelay(t *testing.T) {
	ctx := context.Background()
	db := storetest.Open(t)
	cloud := httptest.NewServer(server.New(server.Config{URL: "http://cloud.test"}, db, slog.New(slog.NewTextHandler(io.Discard, nil))))
	t.Cleanup(cloud.Close)
	ada := signInHost(t, db, cloud.URL, 1, "ada")
	bob := signInHost(t, db, cloud.URL, 2, "bob")
	team, _ := db.CreateTeam(ctx, ada.acc, "Design")
	inv, _ := db.CreateInvite(ctx, team.ID, ada.acc, api.InviteParams{Role: api.RoleMember})
	db.AcceptInvite(ctx, inv.Token, bob.acc)
	ada.srv.learnTeams(ada.acc, []api.Team{{ID: team.ID, Name: "Design", Role: api.RoleOwner}})
	project := gitRepo(t)
	ada.c.AddProject(project)
	t.Setenv(proto.EnvTeamListen, "127.0.0.1:0") // not this machine's Tailscale
	if _, err := ada.c.SetHostShare(team.ID, []string{project}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		sh, err := ada.c.HostShare()
		if err == nil && sh.Relay {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the host never reached the relay: %+v %v", sh, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	adaHost, _ := ada.srv.cloud.store.Get()
	var announced proto.TeamHost
	data, _ := os.ReadFile(filepath.Join(ada.srv.teamsDir(), team.ID, "hosts", adaHost.DeviceID+".json"))
	if json.Unmarshal(data, &announced) != nil || announced.Key == "" {
		t.Fatalf("the host announced no key for members' channels: %s", data)
	}

	bobDev, _ := bob.srv.cloud.store.Get()
	tk, err := db.IssueTicket(ctx, team.ID, bob.acc, bobDev.DeviceID, adaHost.DeviceID)
	if err != nil {
		t.Fatal(err)
	}
	dctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	nc, err := relay.Dial(dctx, "ws"+strings.TrimPrefix(cloud.URL, "http")+api.PathRelayConnect, tk.Ticket, announced.Key)
	if err != nil {
		t.Fatal(err)
	}
	v, err := client.New(nc, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	if me, err := v.TeamAuth(tk.Ticket); err != nil || me.GitHub != "bob" {
		t.Fatalf("signed in through the relay: %+v %v", me, err)
	}
	if list, err := v.Projects(); err != nil || len(list) != 1 || list[0].Path != project {
		t.Fatalf("the shared project through the relay: %+v %v", list, err)
	}

	if err := db.RemoveMember(ctx, team.ID, ada.acc, bob.acc); err != nil {
		t.Fatal(err)
	}
	ada.srv.recheckVisitors("")
	deadline = time.Now().Add(5 * time.Second)
	for {
		if _, err := v.Projects(); err != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("a member removed from the team stayed connected through the relay")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
