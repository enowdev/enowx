package daemon

import (
	"strings"
	"testing"

	"github.com/enowx/enowx/proto"
)

// A report still waiting for GOA when its task is settled goes all the same, said to be
// settled, as GOA may have settled it on a guess; a line only saying to look goes no more.
func TestGOAReportOfASettledTaskStillGoes(t *testing.T) {
	srv := startServer(t)
	task, err := srv.goa.board.Add(proto.GOATask{Title: "settled", Session: "a-session", Status: proto.GOATaskWorking})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := srv.goa.board.Update(task.ID, func(x *proto.GOATask) { x.Status = proto.GOATaskDone }); err != nil {
		t.Fatal(err)
	}
	srv.goa.mu.Lock()
	srv.goa.tell = append(srv.goa.tell,
		goaLine{Text: "it reports done", Task: task.ID, Session: "a-session", Turn: goaLineReport, Report: true, Closed: true},
		goaLine{Text: "it isn't moving", Task: task.ID, Session: "a-session"},
		goaLine{Text: "no task"})
	srv.goa.mu.Unlock()
	got := srv.goaLinesToSay()
	if len(got) != 2 || got[0].Text != "it reports done"+goaSettledNote || got[1].Text != "no task" {
		t.Fatalf("to say: %+v", got)
	}
	if !strings.Contains(got[0].Text, "settled already") {
		t.Fatalf("the report doesn't say its task is settled: %q", got[0].Text)
	}
}

// A report joined to its turn's end, or a turn's end to it, stays a report.
func TestGOAReportJoinedStaysAReport(t *testing.T) {
	srv := startServer(t)
	srv.goa.mu.Lock()
	srv.goa.tell = append(srv.goa.tell, goaLine{Text: "[enowX GOA] Task t1 finished a turn: said", Task: "t1", Session: "s1", Turn: goaLineTurn})
	srv.goa.mu.Unlock()
	srv.goaTellReport(proto.SessionInfo{ID: "s1"}, goaLine{Text: "[enowX GOA] Task t1 reports done: ok", Task: "t1", Session: "s1"})
	srv.goa.mu.Lock()
	defer srv.goa.mu.Unlock()
	if len(srv.goa.tell) != 1 || !srv.goa.tell[0].Report {
		t.Fatalf("waiting: %+v", srv.goa.tell)
	}
}
