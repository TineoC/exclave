package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeCatalog(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for path, body := range map[string]string{
		"baseline/1.0.0/release.yaml": "product: b\nversion: 1.0.0\n",
		"baseline/1.1.0/release.yaml": "product: b\nversion: 1.1.0\n",
		"evidence/spdx.json":          "{}\n",
	} {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// A manifest whose bytes vary between runs cannot be signed, so determinism is
// the property the whole feature rests on.
func TestManifestIsDeterministic(t *testing.T) {
	dir := writeCatalog(t)
	first, err := BuildManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		again, err := BuildManifest(dir)
		if err != nil {
			t.Fatal(err)
		}
		if again != first {
			t.Fatalf("manifest changed between runs:\n%s\n---\n%s", first, again)
		}
	}
	if !strings.HasPrefix(first, ManifestHeader) {
		t.Errorf("manifest does not start with its header")
	}
	if !strings.Contains(first, "digest  ") {
		t.Errorf("manifest has no digest line")
	}
}

func TestVerifyManifest(t *testing.T) {
	dir := writeCatalog(t)
	manifest, err := BuildManifest(dir)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("unchanged catalog verifies", func(t *testing.T) {
		drift, err := VerifyManifest(dir, manifest)
		if err != nil {
			t.Fatal(err)
		}
		if len(drift) != 0 {
			t.Errorf("expected no drift, got %v", drift)
		}
	})

	// The scenario the feature exists for: someone quietly lowers a constraint
	// or a CVE count in a release file.
	t.Run("a modified release is caught and named", func(t *testing.T) {
		target := filepath.Join(dir, "baseline/1.1.0/release.yaml")
		orig, _ := os.ReadFile(target)
		defer os.WriteFile(target, orig, 0o644)

		if err := os.WriteFile(target, []byte("product: b\nversion: 1.1.0\ntampered: true\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		drift, err := VerifyManifest(dir, manifest)
		if err != nil {
			t.Fatal(err)
		}
		if len(drift) != 1 {
			t.Fatalf("expected 1 drift, got %d: %v", len(drift), drift)
		}
		if drift[0].Path != "baseline/1.1.0/release.yaml" || drift[0].Reason != "modified" {
			t.Errorf("got %+v, want the modified release named", drift[0])
		}
	})

	t.Run("an added file is caught", func(t *testing.T) {
		extra := filepath.Join(dir, "baseline/9.9.9/release.yaml")
		os.MkdirAll(filepath.Dir(extra), 0o755)
		os.WriteFile(extra, []byte("product: b\nversion: 9.9.9\n"), 0o644)
		defer os.RemoveAll(filepath.Dir(extra))

		drift, err := VerifyManifest(dir, manifest)
		if err != nil {
			t.Fatal(err)
		}
		if len(drift) != 1 || drift[0].Reason != "not in manifest" {
			t.Errorf("expected the added file to be caught, got %v", drift)
		}
	})

	t.Run("a removed file is caught", func(t *testing.T) {
		target := filepath.Join(dir, "evidence/spdx.json")
		orig, _ := os.ReadFile(target)
		os.Remove(target)
		defer os.WriteFile(target, orig, 0o644)

		drift, err := VerifyManifest(dir, manifest)
		if err != nil {
			t.Fatal(err)
		}
		if len(drift) != 1 || drift[0].Reason != "missing from catalog" {
			t.Errorf("expected the removed file to be caught, got %v", drift)
		}
	})

	t.Run("a foreign manifest is rejected", func(t *testing.T) {
		if _, err := VerifyManifest(dir, "sha  file\ndigest  abc\n"); err == nil {
			t.Error("expected a manifest without the header to be rejected")
		}
	})
}
