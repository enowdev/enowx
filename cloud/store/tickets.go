package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"

	"github.com/enowx/enowx/cloud/api"
)

const ticketPrefix = "enxt_"

// IssueTicket gives member accID, on their device fromDevice, a ticket into
// host device hostDevice, which must be a device of a member of team
// teamID.
func (s *DB) IssueTicket(ctx context.Context, teamID, accID, fromDevice, hostDevice string) (api.HostTicket, error) {
	var out api.HostTicket
	err := s.asRole(ctx, teamID, accID, api.RoleViewer, func(tx *sql.Tx, _ string) error {
		members, err := memberDevices(ctx, tx, teamID)
		if err != nil {
			return err
		}
		if !members[hostDevice] {
			return ErrNotFound // no host of the team's
		}
		var b [32]byte
		rand.Read(b[:])
		out.Ticket = ticketPrefix + base64.RawURLEncoding.EncodeToString(b[:])
		out.Expires = now().Add(api.TicketLife)
		_, err = tx.ExecContext(ctx, `INSERT INTO host_tickets (hash, team_id, account_id, from_device, host_device, expires) VALUES ($1, $2, $3, $4, $5, $6)`,
			hashToken(out.Ticket), teamID, accID, fromDevice, hostDevice, stamp(out.Expires))
		return err
	})
	return out, err
}

// ErrTicket is a ticket that is no good: unknown, used, expired, for
// another host, or of one no longer a member.
var ErrTicket = errors.New("store: the ticket is no good here")

// RedeemTicket is who ticket lets into host device hostDevice, once.
func (s *DB) RedeemTicket(ctx context.Context, ticket, hostDevice string) (api.Visitor, error) {
	tx, err := s.BeginTx(ctx, nil)
	if err != nil {
		return api.Visitor{}, err
	}
	defer tx.Rollback()
	var v api.Visitor
	var host, expires string
	var used int
	err = tx.QueryRowContext(ctx, `SELECT team_id, account_id, from_device, host_device, expires, used FROM host_tickets WHERE hash = $1`, hashToken(ticket)).
		Scan(&v.Team, &v.Account.ID, &v.Device, &host, &expires, &used)
	if errors.Is(err, sql.ErrNoRows) {
		return api.Visitor{}, ErrTicket
	}
	if err != nil {
		return api.Visitor{}, err
	}
	if used != 0 || host != hostDevice || time.Now().After(parseStamp(expires)) {
		return api.Visitor{}, ErrTicket
	}
	// Used once: of two redeeming it at once, one finds it taken.
	res, err := tx.ExecContext(ctx, `UPDATE host_tickets SET used = 1 WHERE hash = $1 AND used = 0`, hashToken(ticket))
	if err != nil {
		return api.Visitor{}, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return api.Visitor{}, ErrTicket
	}
	if v.Role, err = role(ctx, tx, v.Team, v.Account.ID); err != nil {
		return api.Visitor{}, ErrTicket // left the team meanwhile
	}
	if err := tx.QueryRowContext(ctx, `SELECT name, github_login FROM accounts WHERE id = $1`, v.Account.ID).Scan(&v.Account.Name, &v.Account.GitHub); err != nil {
		return api.Visitor{}, err
	}
	if err := audit(ctx, tx, v.Team, v.Account.ID, "host.entered", hostDevice); err != nil {
		return api.Visitor{}, err
	}
	// Tickets past their time go as others are redeemed.
	tx.ExecContext(ctx, `DELETE FROM host_tickets WHERE expires < $1`, stamp(now().Add(-time.Hour)))
	return v, tx.Commit()
}

// PeekTicket says which host and team ticket leads to, without using it:
// the relay's check before it pairs a member with a host. It is no good
// used, past its time, or of one no longer in the team.
func (s *DB) PeekTicket(ctx context.Context, ticket string) (teamID, accountID, hostDevice string, err error) {
	var expires string
	var used int
	err = s.QueryRowContext(ctx, `SELECT team_id, account_id, host_device, expires, used FROM host_tickets WHERE hash = $1`, hashToken(ticket)).
		Scan(&teamID, &accountID, &hostDevice, &expires, &used)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", "", ErrTicket
	}
	if err != nil {
		return "", "", "", err
	}
	if used != 0 || time.Now().After(parseStamp(expires)) {
		return "", "", "", ErrTicket
	}
	if _, err := role(ctx, s.DB, teamID, accountID); err != nil {
		return "", "", "", ErrTicket
	}
	return teamID, accountID, hostDevice, nil
}
