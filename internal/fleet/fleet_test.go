package fleet

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFleet(t *testing.T, descriptors map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range descriptors {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const il2Site = "name: lab\ntier: lab\nclassification: il2\nchannel: stable\nkubernetes: \"1.29\"\n"
const il6Site = "name: enclave\ntier: production\nclassification: il6\nchannel: stable\nkubernetes: \"1.29\"\n"

// A fleet directory is an aggregate, and an aggregate of site descriptors maps
// which sites run which versions and when each is in maintenance. The ceiling is
// what stops the corp low side ingesting one it should never hold.
func TestClassificationCeiling(t *testing.T) {
	dir := writeFleet(t, map[string]string{"lab.yaml": il2Site, "enclave.yaml": il6Site})

	t.Run("over-classified descriptor is an error, not a skip", func(t *testing.T) {
		_, err := Load(dir, "il2")
		if err == nil {
			t.Fatal("expected an il6 descriptor to be refused under an il2 ceiling")
		}
		var over ErrOverClassified
		if !errors.As(err, &over) {
			t.Fatalf("got %T, want ErrOverClassified", err)
		}
		// Silently dropping it would be worse than failing: the operator would
		// believe they had a complete picture.
		if !strings.Contains(err.Error(), "enclave.yaml") || !strings.Contains(err.Error(), "il6") {
			t.Errorf("error must name the file and the level, got: %v", err)
		}
	})

	t.Run("a sufficient ceiling loads everything", func(t *testing.T) {
		envs, err := Load(dir, "il6")
		if err != nil {
			t.Fatal(err)
		}
		if len(envs) != 2 {
			t.Errorf("got %d environments, want 2", len(envs))
		}
	})

	t.Run("no ceiling loads everything", func(t *testing.T) {
		if _, err := Load(dir, ""); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("an unknown ceiling is rejected", func(t *testing.T) {
		if _, err := Load(dir, "il3"); err == nil {
			t.Error("il3 was consolidated into il4 and must not be accepted as a ceiling")
		}
	})

	t.Run("an unrecognised classification is refused, not assumed low", func(t *testing.T) {
		d := writeFleet(t, map[string]string{
			"odd.yaml": "name: odd\ntier: lab\nclassification: secret-squirrel\nchannel: stable\nkubernetes: \"1.29\"\n",
		})
		if _, err := Load(d, "il6"); err == nil {
			t.Error("an unknown classification must not be treated as below the ceiling")
		}
	})
}
