// Package resolve decides which release each environment is eligible for, and
// — just as importantly — explains why it is not eligible for the others.
//
// A resolver that cannot explain itself is not operable. "No eligible release"
// with no reason is the failure mode every system of this kind hits, so every
// constraint here produces a human-readable detail string whether it passes or
// fails, and those strings are covered by tests.
package resolve

import (
	"fmt"

	"github.com/Masterminds/semver/v3"

	"github.com/TineoC/exclave/internal/catalog"
	"github.com/TineoC/exclave/internal/fleet"
)

// Check is one constraint evaluated against one environment.
type Check struct {
	Name   string
	OK     bool
	Detail string
}

// Evaluation is the full result of testing one release against one environment.
type Evaluation struct {
	Version  string
	Eligible bool
	Checks   []Check
}

// Blocker returns the detail of the first failing check, or "" when eligible.
func (e Evaluation) Blocker() string {
	for _, c := range e.Checks {
		if !c.OK {
			return c.Detail
		}
	}
	return ""
}

// Status describes what should happen to an environment.
type Status string

const (
	StatusUpgrade    Status = "upgrade"
	StatusCurrent    Status = "current"
	StatusNoEligible Status = "no eligible release"
	StatusPinned     Status = "pinned"
	StatusBehind     Status = "behind"
)

// Decision is the resolver's answer for one environment.
type Decision struct {
	Environment fleet.Environment
	Target      string // empty when nothing is eligible
	Status      Status
	// Note explains the decision: why a higher release was skipped, or why
	// nothing was eligible. Always safe to render in parentheses.
	Note string
}

// Evaluate tests one release against one environment, recording every check.
func Evaluate(r catalog.Release, e fleet.Environment) Evaluation {
	ev := Evaluation{Version: r.Version}
	add := func(name string, ok bool, format string, args ...any) {
		ev.Checks = append(ev.Checks, Check{Name: name, OK: ok, Detail: fmt.Sprintf(format, args...)})
	}

	// Channel: an environment accepts any release at or below its own rank.
	envRank, envKnown := catalog.ChannelRank[e.Channel]
	relRank, relKnown := catalog.ChannelRank[r.Channel]
	switch {
	case !envKnown:
		add("channel", false, "environment channel %q is unknown", e.Channel)
	case !relKnown:
		add("channel", false, "release channel %q is unknown", r.Channel)
	case relRank > envRank:
		add("channel", false, "channel %s does not accept %s", e.Channel, r.Channel)
	default:
		add("channel", true, "channel %s accepts %s", e.Channel, r.Channel)
	}

	// Schema: monotonic integer floor.
	if r.Requires.Schema > 0 {
		ok := e.Schema >= r.Requires.Schema
		if ok {
			add("schema", true, "schema %d >= %d", e.Schema, r.Requires.Schema)
		} else {
			add("schema", false, "schema %d < %d", e.Schema, r.Requires.Schema)
		}
	}

	// Kubernetes: semver range.
	if r.Requires.Kubernetes != "" {
		kv, err := e.KubernetesSemVer()
		if err != nil {
			add("kubernetes", false, "%v", err)
		} else {
			c, err := semver.NewConstraint(r.Requires.Kubernetes)
			if err != nil {
				add("kubernetes", false, "invalid constraint %q", r.Requires.Kubernetes)
			} else if c.Check(kv) {
				add("kubernetes", true, "kubernetes %s satisfies %s", e.Kubernetes, r.Requires.Kubernetes)
			} else {
				add("kubernetes", false, "kubernetes %s not in %s", e.Kubernetes, r.Requires.Kubernetes)
			}
		}
	}

	// Platform: the upgrade path. Nothing installed means any jump is allowed.
	if r.Requires.Platform != "" {
		cur, err := e.CurrentSemVer()
		switch {
		case err != nil:
			add("platform", false, "%v", err)
		case cur == nil:
			add("platform", true, "no version installed, upgrade path unconstrained")
		default:
			c, err := semver.NewConstraint(r.Requires.Platform)
			if err != nil {
				add("platform", false, "invalid constraint %q", r.Requires.Platform)
			} else if c.Check(cur) {
				add("platform", true, "installed %s satisfies platform %s", e.Current, r.Requires.Platform)
			} else {
				add("platform", false, "requires platform %s, installed %s", r.Requires.Platform, e.Current)
			}
		}
	}

	// Tier: deny list.
	forbidden := false
	for _, t := range r.Forbids.EnvironmentTier {
		if t == e.Tier {
			forbidden = true
			break
		}
	}
	if len(r.Forbids.EnvironmentTier) > 0 {
		if forbidden {
			add("tier", false, "tier %s is forbidden by this release", e.Tier)
		} else {
			add("tier", true, "tier %s is not forbidden", e.Tier)
		}
	}

	// Classification: allow list. An empty list is permissive.
	if len(r.AllowedClassifications) > 0 {
		allowed := false
		for _, c := range r.AllowedClassifications {
			if c == e.Classification {
				allowed = true
				break
			}
		}
		if allowed {
			add("classification", true, "classification %s is permitted", e.Classification)
		} else {
			add("classification", false, "classification %s not permitted by this release", e.Classification)
		}
	}

	ev.Eligible = true
	for _, c := range ev.Checks {
		if !c.OK {
			ev.Eligible = false
			break
		}
	}
	return ev
}

// Plan resolves every environment against the catalog.
func Plan(releases []catalog.Release, envs []fleet.Environment) []Decision {
	out := make([]Decision, 0, len(envs))
	for _, e := range envs {
		out = append(out, decide(releases, e))
	}
	return out
}

func decide(releases []catalog.Release, e fleet.Environment) Decision {
	d := Decision{Environment: e}

	// A pinned environment ignores the channel and takes exactly what it names —
	// but it still has to pass every other constraint, and says so if it cannot.
	if e.Pinned != "" {
		for _, r := range releases {
			if r.Version != e.Pinned {
				continue
			}
			ev := Evaluate(r, e)
			if ev.Eligible {
				d.Target = r.Version
				d.Status = StatusPinned
				if r.Version == e.Current {
					d.Note = "pinned, already installed"
				} else {
					d.Note = fmt.Sprintf("pinned to %s", r.Version)
				}
				return d
			}
			d.Status = StatusNoEligible
			d.Note = fmt.Sprintf("pinned %s blocked: %s", r.Version, ev.Blocker())
			return d
		}
		d.Status = StatusNoEligible
		d.Note = fmt.Sprintf("pinned %s is not in the catalog", e.Pinned)
		return d
	}

	// Releases arrive sorted ascending; walk backwards to find the newest that fits.
	//
	// Two blockers are tracked. When nothing is eligible, the useful explanation is
	// the newest release the environment would actually consider — reporting "does
	// not accept edge" when the real problem is a schema floor sends people chasing
	// the wrong thing. When something *is* eligible, any newer blocked release is
	// worth surfacing, including one held back only by channel.
	var (
		target         *catalog.Release
		anyBlock       blocked
		inChannelBlock blocked
	)
	for i := len(releases) - 1; i >= 0; i-- {
		ev := Evaluate(releases[i], e)
		if ev.Eligible {
			r := releases[i]
			target = &r
			break
		}
		if !anyBlock.set {
			anyBlock = blocked{releases[i].Version, ev.Blocker(), true}
		}
		if !inChannelBlock.set && channelOK(ev) {
			inChannelBlock = blocked{releases[i].Version, ev.Blocker(), true}
		}
	}

	if target == nil {
		d.Status = StatusNoEligible
		switch {
		case inChannelBlock.set:
			d.Note = inChannelBlock.reason
		case anyBlock.set:
			d.Note = anyBlock.reason
		}
		return d
	}

	d.Target = target.Version
	// Only in-channel blockers are worth reporting. An environment on the stable
	// channel is not surprised that an edge release exists, and saying so on every
	// row buries the one message that matters: a release it *would* take, held
	// back by a constraint it could act on.
	if inChannelBlock.set {
		d.Note = fmt.Sprintf("%s blocked: %s", inChannelBlock.version, inChannelBlock.reason)
	}

	switch {
	case e.Current == "":
		d.Status = StatusUpgrade
	case target.Version == e.Current:
		d.Status = StatusCurrent
	default:
		cur, _ := e.CurrentSemVer()
		tv, _ := target.SemVer()
		if cur != nil && tv.LessThan(cur) {
			d.Status = StatusBehind
		} else {
			d.Status = StatusUpgrade
		}
	}
	return d
}

// blocked remembers why one release was rejected.
type blocked struct {
	version string
	reason  string
	set     bool
}

// channelOK reports whether the release cleared the channel gate, meaning the
// environment would have considered it at all.
func channelOK(ev Evaluation) bool {
	for _, c := range ev.Checks {
		if c.Name == "channel" {
			return c.OK
		}
	}
	return true
}
