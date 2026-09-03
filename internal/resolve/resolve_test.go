package resolve

import (
	"strings"
	"testing"

	"github.com/TineoC/exclave/internal/catalog"
	"github.com/TineoC/exclave/internal/fleet"
)

func rel(version, channel string, mut ...func(*catalog.Release)) catalog.Release {
	r := catalog.Release{
		Product:    "acme-platform",
		Version:    version,
		Channel:    channel,
		Components: []catalog.Component{{Name: "auth-svc", Version: "1.0.0", Image: "acme/auth@sha256:aa"}},
	}
	for _, m := range mut {
		m(&r)
	}
	return r
}

func env(name string, mut ...func(*fleet.Environment)) fleet.Environment {
	e := fleet.Environment{
		Name:           name,
		Tier:           "production",
		Classification: "il5",
		Channel:        "stable",
		Kubernetes:     "1.29",
		Schema:         9,
		Current:        "3.1.0",
	}
	for _, m := range mut {
		m(&e)
	}
	return e
}

// The detail strings are the product. If they regress, the resolver stops being
// operable even while it still picks the right versions — so assert on them.
func TestEvaluateChecks(t *testing.T) {
	tests := []struct {
		name       string
		release    catalog.Release
		env        fleet.Environment
		wantEligib bool
		wantDetail string // substring expected in the first failing check
	}{
		{
			name:       "all constraints satisfied",
			release:    rel("3.2.0", "stable"),
			env:        env("ok"),
			wantEligib: true,
		},
		{
			name:       "schema floor not met",
			release:    rel("3.2.0", "stable", func(r *catalog.Release) { r.Requires.Schema = 12 }),
			env:        env("old", func(e *fleet.Environment) { e.Schema = 6 }),
			wantEligib: false,
			wantDetail: "schema 6 < 12",
		},
		{
			name:       "kubernetes outside range",
			release:    rel("3.2.0", "stable", func(r *catalog.Release) { r.Requires.Kubernetes = ">=1.28 <1.32" }),
			env:        env("newk8s", func(e *fleet.Environment) { e.Kubernetes = "1.33" }),
			wantEligib: false,
			wantDetail: "kubernetes 1.33 not in >=1.28 <1.32",
		},
		{
			name:       "kubernetes inside range",
			release:    rel("3.2.0", "stable", func(r *catalog.Release) { r.Requires.Kubernetes = ">=1.28 <1.32" }),
			env:        env("goodk8s", func(e *fleet.Environment) { e.Kubernetes = "1.29" }),
			wantEligib: true,
		},
		{
			name:       "upgrade path not satisfied",
			release:    rel("3.2.0", "stable", func(r *catalog.Release) { r.Requires.Platform = ">=3.1" }),
			env:        env("behind", func(e *fleet.Environment) { e.Current = "3.0.1" }),
			wantEligib: false,
			wantDetail: "requires platform >=3.1, installed 3.0.1",
		},
		{
			name:       "upgrade path unconstrained on a fresh install",
			release:    rel("3.2.0", "stable", func(r *catalog.Release) { r.Requires.Platform = ">=3.1" }),
			env:        env("fresh", func(e *fleet.Environment) { e.Current = "" }),
			wantEligib: true,
		},
		{
			name:       "channel ladder rejects a newer channel",
			release:    rel("3.3.0", "edge"),
			env:        env("conservative", func(e *fleet.Environment) { e.Channel = "stable" }),
			wantEligib: false,
			wantDetail: "channel stable does not accept edge",
		},
		{
			name:       "channel ladder accepts a more conservative channel",
			release:    rel("3.2.0", "stable"),
			env:        env("adventurous", func(e *fleet.Environment) { e.Channel = "edge" }),
			wantEligib: true,
		},
		{
			name: "forbidden tier",
			release: rel("3.2.0", "stable", func(r *catalog.Release) {
				r.Forbids.EnvironmentTier = []string{"production"}
			}),
			env:        env("prod"),
			wantEligib: false,
			wantDetail: "tier production is forbidden",
		},
		{
			name: "classification not permitted",
			release: rel("3.2.0", "stable", func(r *catalog.Release) {
				r.AllowedClassifications = []string{"il5", "il6"}
			}),
			env:        env("low", func(e *fleet.Environment) { e.Classification = "il2" }),
			wantEligib: false,
			wantDetail: "classification il2 not permitted",
		},
		{
			name: "classification permitted",
			release: rel("3.2.0", "stable", func(r *catalog.Release) {
				r.AllowedClassifications = []string{"il5", "il6"}
			}),
			env:        env("high", func(e *fleet.Environment) { e.Classification = "il6" }),
			wantEligib: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := Evaluate(tt.release, tt.env)
			if ev.Eligible != tt.wantEligib {
				t.Fatalf("eligible = %v, want %v (blocker: %s)", ev.Eligible, tt.wantEligib, ev.Blocker())
			}
			if tt.wantDetail == "" {
				return
			}
			if got := ev.Blocker(); !strings.Contains(got, tt.wantDetail) {
				t.Errorf("blocker = %q, want it to contain %q", got, tt.wantDetail)
			}
		})
	}
}

func TestPlanDecisions(t *testing.T) {
	// Every release is restricted to il5/il6, which is what lets one environment
	// below have nothing at all available to it.
	cleared := func(r *catalog.Release) { r.AllowedClassifications = []string{"il5", "il6"} }

	releases := []catalog.Release{
		rel("3.0.0", "stable", cleared),
		rel("3.1.4", "stable", cleared),
		rel("3.2.0", "stable", cleared, func(r *catalog.Release) {
			r.Requires.Platform = ">=3.1"
			r.Requires.Schema = 7
		}),
		rel("3.3.0", "edge", cleared),
	}

	tests := []struct {
		name       string
		env        fleet.Environment
		wantTarget string
		wantStatus Status
		wantNote   string
	}{
		{
			name:       "takes the newest eligible release",
			env:        env("a-prod", func(e *fleet.Environment) { e.Current = "3.1.4" }),
			wantTarget: "3.2.0",
			wantStatus: StatusUpgrade,
			// No note: the only newer release is on a channel this environment
			// deliberately does not track, which is not news.
			wantNote: "",
		},
		{
			name:       "upgrade path blocks the newest, falls back and says why",
			env:        env("b-prod", func(e *fleet.Environment) { e.Current = "3.0.1" }),
			wantTarget: "3.1.4",
			wantStatus: StatusUpgrade,
			wantNote:   "3.2.0 blocked: requires platform >=3.1, installed 3.0.1",
		},
		{
			name: "already on the newest eligible release",
			env: env("lab", func(e *fleet.Environment) {
				e.Current = "3.2.0"
				e.Schema = 9
			}),
			wantTarget: "3.2.0",
			wantStatus: StatusCurrent,
		},
		{
			name: "nothing eligible reports the blocking constraint",
			env: env("c-prod", func(e *fleet.Environment) {
				e.Current = "2.9.0"
				e.Channel = "stable"
				e.Classification = "il2"
				e.Schema = 6
			}),
			wantTarget: "",
			wantStatus: StatusNoEligible,
			// The in-channel blocker, not the edge-channel one: reporting a channel
			// mismatch here would send someone chasing entirely the wrong problem.
			wantNote: "schema 6 < 7",
		},
		{
			name: "pinned version wins over channel tracking",
			env: env("pinned", func(e *fleet.Environment) {
				e.Current = "3.0.0"
				e.Pinned = "3.1.4"
			}),
			wantTarget: "3.1.4",
			wantStatus: StatusPinned,
			wantNote:   "pinned to 3.1.4",
		},
		{
			name: "pinned but ineligible explains itself",
			env: env("pinned-bad", func(e *fleet.Environment) {
				e.Current = "3.0.0"
				e.Pinned = "3.2.0"
			}),
			wantTarget: "",
			wantStatus: StatusNoEligible,
			wantNote:   "pinned 3.2.0 blocked: requires platform >=3.1",
		},
		{
			name: "pinned to something that does not exist",
			env: env("pinned-missing", func(e *fleet.Environment) {
				e.Pinned = "9.9.9"
			}),
			wantTarget: "",
			wantStatus: StatusNoEligible,
			wantNote:   "pinned 9.9.9 is not in the catalog",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Plan(releases, []fleet.Environment{tt.env})
			if len(got) != 1 {
				t.Fatalf("expected 1 decision, got %d", len(got))
			}
			d := got[0]
			if d.Target != tt.wantTarget {
				t.Errorf("target = %q, want %q", d.Target, tt.wantTarget)
			}
			if d.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", d.Status, tt.wantStatus)
			}
			if tt.wantNote != "" && !strings.Contains(d.Note, tt.wantNote) {
				t.Errorf("note = %q, want it to contain %q", d.Note, tt.wantNote)
			}
		})
	}
}

// A blocked environment must name a constraint, never just fail silently.
func TestNoEligibleAlwaysExplains(t *testing.T) {
	releases := []catalog.Release{
		rel("3.2.0", "stable", func(r *catalog.Release) { r.Requires.Schema = 12 }),
	}
	e := env("stuck", func(e *fleet.Environment) { e.Schema = 6 })

	d := Plan(releases, []fleet.Environment{e})[0]
	if d.Status != StatusNoEligible {
		t.Fatalf("status = %q, want %q", d.Status, StatusNoEligible)
	}
	if d.Note == "" {
		t.Fatal("a blocked environment produced no explanation, which is the failure mode this tool exists to prevent")
	}
	if !strings.Contains(d.Note, "schema 6 < 12") {
		t.Errorf("note = %q, want it to name the schema constraint", d.Note)
	}
}
