package resolve

import (
	"strings"
	"testing"

	"github.com/TineoC/exclave/internal/fleet"
)

// Defaulting to an empty salt would produce output that looks redacted and is
// reversible by dictionary attack over guessable site names. Refusing is the
// only safe behaviour, so it is pinned by a test.
func TestRedactionRefusesWithoutSalt(t *testing.T) {
	if _, err := SiteID("navy-xyz-il4", ""); err != ErrNoSalt {
		t.Errorf("SiteID with no salt = %v, want ErrNoSalt", err)
	}
	if _, err := Redact([]Decision{{}}, "", false); err != ErrNoSalt {
		t.Errorf("Redact with no salt = %v, want ErrNoSalt", err)
	}
}

func TestSiteIDStability(t *testing.T) {
	a, _ := SiteID("navy-xyz-il4", "salt-one")
	b, _ := SiteID("navy-xyz-il4", "salt-one")
	c, _ := SiteID("navy-xyz-il4", "salt-two")
	d, _ := SiteID("army-abc-il5", "salt-one")

	if a != b {
		t.Error("same name and salt must yield the same id, or a site cannot be tracked over time")
	}
	if a == c {
		t.Error("a different salt must yield a different id, or rotating the salt achieves nothing")
	}
	if a == d {
		t.Error("different names must yield different ids")
	}
	if !strings.HasPrefix(a, "site-") {
		t.Errorf("id %q should be recognisable as an opaque site id", a)
	}

	// Without a separator, salt "ab"+name "cd" and salt "a"+name "bcd" collide.
	x, _ := SiteID("cd", "ab")
	y, _ := SiteID("bcd", "a")
	if x == y {
		t.Error("salt and name must not be concatenable into the same input")
	}
}

// The point of redaction is that identity cannot leak, including through a
// future output format. It must not be carried in the struct at all.
func TestRedactedCarriesNoIdentity(t *testing.T) {
	d := Decision{
		Environment: fleet.Environment{
			Name:              "usaf-def-il6",
			Classification:    "il6",
			Current:           "4.2.1",
			MaintenanceWindow: "quarterly, coordinated",
			Tier:              "production",
			Kubernetes:        "1.29",
		},
		Target: "4.3.0",
		Status: StatusUpgrade,
	}

	rows, err := Redact([]Decision{d}, "salt", false)
	if err != nil {
		t.Fatal(err)
	}
	r := rows[0]

	for field, v := range map[string]string{
		"name":           r.SiteID,
		"classification": r.Classification,
	} {
		if strings.Contains(v, "usaf") || v == "il6" {
			t.Errorf("%s leaked identity: %q", field, v)
		}
	}
	if r.Classification != "" {
		t.Error("classification must be dropped by default: 'three IL6 sites are behind' is itself sensitive")
	}
	if r.Current != "4.2.1" || r.Target != "4.3.0" {
		t.Error("drift must survive redaction, or the roll-up conveys nothing")
	}

	// Opting in is explicit.
	kept, _ := Redact([]Decision{d}, "salt", true)
	if kept[0].Classification != "il6" {
		t.Error("--keep-classification must include it for a cleared receiver")
	}
}
