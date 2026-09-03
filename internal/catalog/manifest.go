package catalog

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ManifestHeader identifies the format. Bump it if the layout ever changes; a
// signature over one layout must not silently validate another.
const ManifestHeader = "# exclave catalog manifest v1"

// BuildManifest produces a deterministic digest of every file in the catalog.
//
// The compliance gate is only as trustworthy as the files it reads. Nothing
// stops someone lowering a `requires.platform` or editing `criticalCves: 4` down
// to `0` in a release.yaml — so the catalog gets a manifest, the manifest gets
// signed out of band, and `VerifyManifest` refuses a catalog that has drifted.
//
// Determinism matters more than it looks: a manifest whose bytes vary between
// runs cannot be signed. Paths are relative and sorted, and line endings are
// fixed, so the same catalog always yields identical bytes on any machine.
//
// Signing stays external. `cosign sign-blob` over this output and
// `cosign verify-blob` before use — this package owns the digest, cosign owns
// the crypto, and neither reimplements the other.
func BuildManifest(dir string) (string, error) {
	type entry struct{ path, sum string }
	var entries []entry

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		sum, err := fileSHA256(path)
		if err != nil {
			return err
		}
		// Always forward slashes: a manifest built on Windows must verify on Linux.
		entries = append(entries, entry{filepath.ToSlash(rel), sum})
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("no files found under %s", dir)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })

	var body strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&body, "%s  %s\n", e.sum, e.path)
	}

	// The trailing digest covers every line above it, so a single value can be
	// compared by eye or pasted into a change record.
	roll := sha256.Sum256([]byte(body.String()))

	var out strings.Builder
	out.WriteString(ManifestHeader + "\n")
	out.WriteString(body.String())
	fmt.Fprintf(&out, "digest  %s\n", hex.EncodeToString(roll[:]))
	return out.String(), nil
}

// ManifestDrift is one file that does not match the manifest.
type ManifestDrift struct {
	Path   string
	Reason string // "modified", "missing from catalog", "not in manifest"
}

// VerifyManifest recomputes the catalog's manifest and reports every difference.
//
// It names the files that drifted rather than only reporting that something did.
// "The catalog does not match its signature" sends someone hunting; "4.3.0/
// release.yaml was modified" tells them where to look.
func VerifyManifest(dir, manifest string) ([]ManifestDrift, error) {
	want, wantDigest, err := parseManifest(manifest)
	if err != nil {
		return nil, err
	}

	current, err := BuildManifest(dir)
	if err != nil {
		return nil, err
	}
	got, gotDigest, err := parseManifest(current)
	if err != nil {
		return nil, err
	}

	if wantDigest == gotDigest {
		return nil, nil
	}

	var drift []ManifestDrift
	for path, sum := range want {
		switch cur, ok := got[path]; {
		case !ok:
			drift = append(drift, ManifestDrift{path, "missing from catalog"})
		case cur != sum:
			drift = append(drift, ManifestDrift{path, "modified"})
		}
	}
	for path := range got {
		if _, ok := want[path]; !ok {
			drift = append(drift, ManifestDrift{path, "not in manifest"})
		}
	}
	sort.Slice(drift, func(i, j int) bool { return drift[i].Path < drift[j].Path })

	if len(drift) == 0 {
		// Digests differ but no file does: the manifest itself is malformed.
		return nil, fmt.Errorf("manifest digest %s does not match computed %s, but no file differs — the manifest is corrupt", wantDigest, gotDigest)
	}
	return drift, nil
}

func parseManifest(s string) (map[string]string, string, error) {
	files := map[string]string{}
	digest := ""
	sc := bufio.NewScanner(strings.NewReader(s))
	seenHeader := false

	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			if line == ManifestHeader {
				seenHeader = true
			}
			continue
		}
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 {
			return nil, "", fmt.Errorf("malformed manifest line: %q", line)
		}
		if parts[0] == "digest" {
			digest = parts[1]
			continue
		}
		files[parts[1]] = parts[0]
	}
	if err := sc.Err(); err != nil {
		return nil, "", err
	}
	if !seenHeader {
		return nil, "", fmt.Errorf("not an exclave catalog manifest (missing %q)", ManifestHeader)
	}
	if digest == "" {
		return nil, "", fmt.Errorf("manifest has no digest line")
	}
	return files, digest, nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
