package resolve

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

// ErrNoSalt is returned when redaction is attempted without a salt.
//
// Defaulting to an empty salt would be the worst possible failure: site names
// are guessable ("navy-xyz-il4", "usaf-def-il6"), so an unsalted digest is a
// rainbow table away from being no redaction at all — while still looking
// redacted to whoever reviews it. Refusing to run is the only safe default.
var ErrNoSalt = errors.New(
	"redaction requires a salt: set EXCLAVE_REDACTION_SALT. " +
		"Site names are guessable, so an unsalted digest is reversible by dictionary attack " +
		"while still looking redacted. The salt and the site-to-customer mapping belong outside this repository")

// Redacted is one row of a roll-up that may cross a classification boundary.
//
// Identity is removed; drift is kept. That is the whole trade: a portfolio view
// exists to show which sites are behind, so current, target and status stay.
// Name, hostname, folder, billing reference and maintenance window are identity
// or targeting data and are dropped entirely — they are never carried in this
// struct, so they cannot leak through a future output format by accident.
type Redacted struct {
	SiteID  string `json:"siteId"`
	Current string `json:"current,omitempty"`
	Target  string `json:"target,omitempty"`
	Status  string `json:"status"`
	// Note explains why a newer release was not taken. It can mention an
	// environment's Kubernetes version, which is drift detail rather than
	// identity — but it is still detail, so the operating guide flags it.
	Note string `json:"note,omitempty"`
	// Classification is empty unless the caller explicitly opts in. "Three IL6
	// sites are behind" is itself sensitive, so a receiver has to be cleared for
	// it and has to ask.
	Classification string `json:"classification,omitempty"`
}

// SiteID is a stable, salted identifier for a site.
//
// Stable so a site can be tracked across successive roll-ups without ever being
// named; salted so the mapping cannot be recovered from the output alone.
func SiteID(name, salt string) (string, error) {
	if salt == "" {
		return "", ErrNoSalt
	}
	// The NUL separator stops salt|name ambiguity: without it, salt "ab" + name
	// "cd" and salt "a" + name "bcd" would hash identically.
	sum := sha256.Sum256([]byte(salt + "\x00" + name))
	return "site-" + hex.EncodeToString(sum[:])[:12], nil
}

// Redact converts decisions into rows safe to review for release downward.
//
// It produces the artifact; it does not release it. Moving this output across a
// classification boundary is a human decision made through the normal review
// process, and nothing here should ever be wired into an automated high-to-low
// path.
func Redact(decisions []Decision, salt string, keepClassification bool) ([]Redacted, error) {
	if salt == "" {
		return nil, ErrNoSalt
	}
	out := make([]Redacted, 0, len(decisions))
	for _, d := range decisions {
		id, err := SiteID(d.Environment.Name, salt)
		if err != nil {
			return nil, fmt.Errorf("redacting %s: %w", d.Environment.Name, err)
		}
		r := Redacted{
			SiteID:  id,
			Current: d.Environment.Current,
			Target:  d.Target,
			Status:  string(d.Status),
			Note:    d.Note,
		}
		if keepClassification {
			r.Classification = d.Environment.Classification
		}
		out = append(out, r)
	}
	return out, nil
}
