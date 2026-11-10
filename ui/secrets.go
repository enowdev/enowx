package ui

import (
	"errors"
	"fmt"

	"github.com/egoist/mygo"

	"github.com/enowx/enowx/core/hosts"
)

// savePassword keeps a host's SSH password in the system's credential
// store. Where there is none (Linux without a keyring), it is kept in
// memory until the app quits, and the user is told.
func (a *App) savePassword(h hosts.Host, password string) {
	err := mygo.Secrets.Set(hosts.PasswordKey(h.ID), []byte(password))
	if err == nil {
		a.keychainMu.Lock()
		if a.keychainRead == nil {
			a.keychainRead = map[string]string{}
		}
		a.keychainRead[h.ID] = password
		a.keychainMu.Unlock()
		return
	}
	if a.passwords == nil {
		a.passwords = map[string]string{}
	}
	a.passwords[h.ID] = password
	if errors.Is(err, errors.ErrUnsupported) {
		a.status = "No keychain on this system: the password is kept until enowX quits."
	} else {
		a.status = "Couldn't save the password in the keychain: " + err.Error()
	}
}

// password returns a host's SSH password. It may block while the system
// asks the user to unlock the keychain, so it runs off the main thread.
func (a *App) password(h hosts.Host) (string, error) { return a.readPassword(h) }

// watchPassword is password for a host's watch, which never asks the user to type
// one: the one saved in the keychain, read once per run, so a host signed in to by
// password connects at start like any other. One the keychain doesn't give (none
// saved, or the system's prompt refused) waits until the user opens the host.
func (a *App) watchPassword(h hosts.Host) (string, error) {
	if pw, ok := a.passwordsSnapshot()[h.ID]; ok {
		return pw, nil
	}
	a.keychainMu.Lock()
	missed := a.keychainMissed[h.ID]
	a.keychainMu.Unlock()
	if missed {
		return "", errNeedsYou
	}
	pw, err := a.readPassword(h)
	if err != nil {
		a.keychainMu.Lock()
		if a.keychainMissed == nil {
			a.keychainMissed = map[string]bool{}
		}
		a.keychainMissed[h.ID] = true
		a.keychainMu.Unlock()
		return "", errNeedsYou
	}
	return pw, nil
}

// readPassword returns h's password from memory, else the keychain: read
// once per run while it answers, so its prompt doesn't come back each
// time the host is connected to.
func (a *App) readPassword(h hosts.Host) (string, error) {
	if pw, ok := a.passwordsSnapshot()[h.ID]; ok {
		return pw, nil
	}
	a.keychainMu.Lock()
	defer a.keychainMu.Unlock() // one prompt at a time
	if pw, ok := a.keychainRead[h.ID]; ok {
		return pw, nil
	}
	pw, err := mygo.Secrets.Get(hosts.PasswordKey(h.ID))
	if err != nil {
		return "", fmt.Errorf("no saved password for %s; connect again to type it", h.Name)
	}
	if a.keychainRead == nil {
		a.keychainRead = map[string]string{}
	}
	a.keychainRead[h.ID] = string(pw)
	return string(pw), nil
}

// passwordsSnapshot copies the in-memory passwords on the main thread.
func (a *App) passwordsSnapshot() map[string]string {
	out := make(chan map[string]string, 1)
	a.onMain(func() {
		m := make(map[string]string, len(a.passwords))
		for k, v := range a.passwords {
			m[k] = v
		}
		out <- m
	})
	return <-out
}

func (a *App) forgetPassword(h hosts.Host) {
	delete(a.passwords, h.ID)
	a.keychainMu.Lock()
	delete(a.keychainRead, h.ID)
	a.keychainMu.Unlock()
	if h.PasswordAuth {
		_ = mygo.Secrets.Delete(hosts.PasswordKey(h.ID))
	}
}
