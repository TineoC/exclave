// Package catalog loads the release catalog: the set of product versions that
// exist and the constraints each one carries about where it may be installed.
package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Masterminds/semver/v3"
	"gopkg.in/yaml.v3"
)

// Component is one service inside a product release, pinned by digest.
type Component struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Image   string `yaml:"image"`
}

// Requires holds the constraints an environment must satisfy to receive a release.
type Requires struct {
	// Kubernetes is a semver range, e.g. ">=1.28 <1.32".
	Kubernetes string `yaml:"kubernetes"`
	// Schema is a monotonic integer floor. The environment's schema must be >= this.
	Schema int `yaml:"schema"`
	// Platform is the upgrade path: the minimum currently-installed version an
	// environment may jump to this release from. Empty means any version may.
	Platform string `yaml:"platform"`
}

// Forbids holds deny lists evaluated against environment attributes.
type Forbids struct {
	EnvironmentTier []string `yaml:"environmentTier"`
}

// Provenance points at the evidence that travels with the artifact.
type Provenance struct {
	SBOM        string `yaml:"sbom"`
	Attestation string `yaml:"attestation"`
}

// Release is one self-describing product version.
type Release struct {
	Product                string      `yaml:"product"`
	Version                string      `yaml:"version"`
	Channel                string      `yaml:"channel"`
	Components             []Component `yaml:"components"`
	Requires               Requires    `yaml:"requires"`
	Forbids                Forbids     `yaml:"forbids"`
	AllowedClassifications []string    `yaml:"allowedClassifications"`
	Provenance             Provenance  `yaml:"provenance"`

	// Path is where this release was loaded from. Not part of the file.
	Path string `yaml:"-"`
}

// SemVer parses the release version.
func (r Release) SemVer() (*semver.Version, error) {
	v, err := semver.NewVersion(r.Version)
	if err != nil {
		return nil, fmt.Errorf("release %s: invalid version %q: %w", r.Path, r.Version, err)
	}
	return v, nil
}

// Validate reports structural problems that would make a release unusable.
func (r Release) Validate() error {
	if r.Product == "" {
		return fmt.Errorf("%s: product is required", r.Path)
	}
	if _, err := r.SemVer(); err != nil {
		return err
	}
	if r.Channel == "" {
		return fmt.Errorf("%s: channel is required (stable, candidate or edge)", r.Path)
	}
	if _, ok := ChannelRank[r.Channel]; !ok {
		return fmt.Errorf("%s: unknown channel %q", r.Path, r.Channel)
	}
	if r.Requires.Kubernetes != "" {
		if _, err := semver.NewConstraint(r.Requires.Kubernetes); err != nil {
			return fmt.Errorf("%s: invalid kubernetes constraint %q: %w", r.Path, r.Requires.Kubernetes, err)
		}
	}
	if r.Requires.Platform != "" {
		if _, err := semver.NewConstraint(r.Requires.Platform); err != nil {
			return fmt.Errorf("%s: invalid platform constraint %q: %w", r.Path, r.Requires.Platform, err)
		}
	}
	if len(r.Components) == 0 {
		return fmt.Errorf("%s: at least one component is required", r.Path)
	}
	for _, c := range r.Components {
		if c.Image == "" {
			return fmt.Errorf("%s: component %s has no image", r.Path, c.Name)
		}
	}
	return nil
}

// ChannelRank orders channels from most to least conservative. An environment
// accepts any release whose channel rank is at or below its own.
var ChannelRank = map[string]int{
	"stable":    0,
	"candidate": 1,
	"edge":      2,
}

// Load reads every release.yaml under dir, sorted ascending by version.
func Load(dir string) ([]Release, error) {
	var releases []Release

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "release.yaml" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var r Release
		if err := yaml.Unmarshal(b, &r); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		r.Path = path
		if err := r.Validate(); err != nil {
			return err
		}
		releases = append(releases, r)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(releases) == 0 {
		return nil, fmt.Errorf("no release.yaml found under %s", dir)
	}

	sort.Slice(releases, func(i, j int) bool {
		a, _ := releases[i].SemVer()
		b, _ := releases[j].SemVer()
		return a.LessThan(b)
	})
	return releases, nil
}
