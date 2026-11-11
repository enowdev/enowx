package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/enowx/enowx/cloud/api"
	"github.com/enowx/enowx/cloud/store"
)

// A team's items: what changed after a sequence number, and writing one
// at its next version (docs/cloud.md, API).

func (s *Server) items(w http.ResponseWriter, r *http.Request) {
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	list, err := s.db.Items(r.Context(), r.PathValue("team"), asker(r).Account.ID, after)
	s.answer(w, list, err)
}

func (s *Server) putItem(w http.ResponseWriter, r *http.Request) {
	var p api.PutItemParams
	// Base64 makes the sealed content a third bigger.
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, api.MaxItemSize/3*4+4096)).Decode(&p); err != nil {
		s.fail(w, http.StatusBadRequest, "can't read the item: "+err.Error())
		return
	}
	me := asker(r)
	it, err := s.db.PutItem(r.Context(), r.PathValue("team"), me.Account.ID, me.Device.ID, r.PathValue("id"), p)
	var conflict *store.ConflictError
	if errors.As(err, &conflict) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(conflict.Item)
		return
	}
	s.answer(w, it, err)
}

func (s *Server) recordAudit(w http.ResponseWriter, r *http.Request) {
	var p api.AuditParams
	if !s.decode(w, r, &p) {
		return
	}
	if !s.limits.allow("audit", asker(r).Account.ID, 120) {
		s.fail(w, http.StatusTooManyRequests, "too many records at once: wait a minute")
		return
	}
	s.answer(w, nil, s.db.RecordAudit(r.Context(), r.PathValue("team"), asker(r).Account.ID, p))
}

func (s *Server) issueTicket(w http.ResponseWriter, r *http.Request) {
	var p api.HostTicketParams
	if !s.decode(w, r, &p) {
		return
	}
	me := asker(r)
	if !s.limits.allow("ticket", me.Account.ID, 30) {
		s.fail(w, http.StatusTooManyRequests, "too many tickets at once: wait a minute")
		return
	}
	t, err := s.db.IssueTicket(r.Context(), r.PathValue("team"), me.Account.ID, me.Device.ID, p.Host)
	t.Relay = s.relayURL()
	s.answer(w, t, err)
}

func (s *Server) redeemTicket(w http.ResponseWriter, r *http.Request) {
	var p api.RedeemParams
	if !s.decode(w, r, &p) {
		return
	}
	v, err := s.db.RedeemTicket(r.Context(), p.Ticket, asker(r).Device.ID)
	if errors.Is(err, store.ErrTicket) {
		s.fail(w, http.StatusForbidden, "the ticket is no good here: used, expired, or for another host")
		return
	}
	s.answer(w, v, err)
}

func (s *Server) audit(w http.ResponseWriter, r *http.Request) {
	before, _ := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64)
	list, err := s.db.Audit(r.Context(), r.PathValue("team"), asker(r).Account.ID, before)
	s.answer(w, list, err)
}

func (s *Server) setRecovery(w http.ResponseWriter, r *http.Request) {
	var p api.RecoveryParams
	if !s.decode(w, r, &p) {
		return
	}
	if !s.recovery.take(asker(r).Device.ID, p.Proof) {
		s.fail(w, http.StatusForbidden, "prove this device's key first: open the challenge "+api.PathRecoveryChallenge+" gives (update enowX if it asks for none)")
		return
	}
	d, err := s.db.SetRecovery(r.Context(), asker(r).Account.ID, p.PublicKey)
	s.answer(w, d, err)
}

func (s *Server) recoveryKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := s.db.RecoveryKeys(r.Context(), r.PathValue("team"), asker(r).Account.ID)
	s.answer(w, keys, err)
}

func (s *Server) recover(w http.ResponseWriter, r *http.Request) {
	var p api.RecoverParams
	if !s.decode(w, r, &p) {
		return
	}
	me := asker(r)
	s.answer(w, nil, s.db.Recover(r.Context(), r.PathValue("team"), me.Account.ID, me.Device.ID, p))
}
