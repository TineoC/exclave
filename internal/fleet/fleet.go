// Package fleet loads the environment descriptors. Environments are data, not
// pipelines: one file per place the product runs, describing what it is and
// what it currently has installed.
package fleet

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"gopkg.in/yaml.v3"
)

// Environment is one place the product runs. You do not have network access to
// it; this file is everything the resolver knows about it.
type Environment struct {
	Name string `yaml:"name"`
	// Tier is matched against a release's forbids.environmentTier.
	Tier string `yaml:"tier"`
	// Classification is matched against a release's allowedClassifications.
	Classification string `yaml:"classification"`
	// Channel is the most adventurous channel this environment accepts.
	Channel string `yaml:"channel"`
	// Kubernetes is the cluster version, matched against requires.kubernetes.
	Kubernetes string `yaml:"kubernetes"`
	// Schema is the environment's data schema level, matched against requires.schema.
	Schema int `yaml:"schema"`
	// Current is the product version installed right now. Empty means nothing
	// is installed yet, which makes every upgrade-path constraint pass.
	Current string `yaml:"current"`
	// Pinned overrides channel tracking with an exact version. Empty means track
	// the channel.
	Pinned string `yaml:"pinned"`
	// MaintenanceWindow is reported, never scheduled. This is a resolver, not cron.
	MaintenanceWindow string `yaml:"maintenanceWindow"`

	// RequiresCapabilities are the capabilities this environment demands, matched
	// by exact value against a release's `provides`. This is where accreditation
	// obligations live — a STIG profile, a FIPS mode, a FedRAMP baseline — so the
	// resolver does not need a new check per compliance concern.
	RequiresCapabilities map[string]any `yaml:"requiresCapabilities"`

	// MaxCriticalCVEs is the count this environment tolerates. Nil means the
	// environment does not gate on CVE counts; zero means a release must have
	// been scanned and found clean, which is stricter than silence.
	MaxCriticalCVEs *int `yaml:"maxCriticalCves"`

	// Path is where this environment was loaded from. Not part of the file.
	Path string `yaml:"-"`
}

// CurrentSemVer parses the installed version. Returns nil when nothing is installed.
func (e Environment) CurrentSemVer() (*semver.Version, error) {
	if strings.TrimSpace(e.Current) == "" {
		return nil, nil
	}
	v, err := semver.NewVersion(e.Current)
	if err != nil {
		return nil, fmt.Errorf("environment %s: invalid current version %q: %w", e.Name, e.Current, err)
	}
	return v, nil
}

// KubernetesSemVer parses the cluster version, tolerating a "1.29" style major.minor.
func (e Environment) KubernetesSemVer() (*semver.Version, error) {
	raw := strings.TrimPrefix(e.Kubernetes, "v")
	if strings.Count(raw, ".") == 1 {
		raw += ".0"
	}
	v, err := semver.NewVersion(raw)
	if err != nil {
		return nil, fmt.Errorf("environment %s: invalid kubernetes version %q: %w", e.Name, e.Kubernetes, err)
	}
	return v, nil
}

// Validate reports structural problems that would make an environment unusable.
func (e Environment) Validate() error {
	if e.Name == "" {
		return fmt.Errorf("%s: name is required", e.Path)
	}
	if e.Tier == "" {
		return fmt.Errorf("%s: tier is required", e.Name)
	}
	if e.Channel == "" {
		return fmt.Errorf("%s: channel is required (stable, candidate or edge)", e.Name)
	}
	if e.Kubernetes == "" {
		return fmt.Errorf("%s: kubernetes version is required", e.Name)
	}
	if _, err := e.KubernetesSemVer(); err != nil {
		return err
	}
	if _, err := e.CurrentSemVer(); err != nil {
		return err
	}
	if e.Pinned != "" {
		if _, err := semver.NewVersion(e.Pinned); err != nil {
			return fmt.Errorf("%s: invalid pinned version %q: %w", e.Name, e.Pinned, err)
		}
	}
	return nil
}

// Load reads every YAML file under dir as an environment, sorted by name.
func Load(dir string) ([]Environment, error) {
	var envs []Environment

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if ext := filepath.Ext(path); ext != ".yaml" && ext != ".yml" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var e Environment
		if err := yaml.Unmarshal(b, &e); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		e.Path = path
		if err := e.Validate(); err != nil {
			return err
		}
		envs = append(envs, e)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(envs) == 0 {
		return nil, fmt.Errorf("no environment files found under %s", dir)
	}

	sort.Slice(envs, func(i, j int) bool { return envs[i].Name < envs[j].Name })
	return envs, nil
}
