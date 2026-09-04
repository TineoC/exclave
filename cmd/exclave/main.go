// Command exclave resolves which product release each environment in a fleet is
// eligible for, and explains every exclusion.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/TineoC/exclave/internal/catalog"
	"github.com/TineoC/exclave/internal/fleet"
	"github.com/TineoC/exclave/internal/resolve"
)

const usage = `exclave — decide which release belongs in which environment.

Usage:
  exclave plan                        newest eligible release per environment
  exclave explain <environment>       every release tested against one environment
  exclave explain <environment> <ver> one release tested, constraint by constraint
  exclave validate                    structural check of the catalog and fleet
  exclave manifest                    deterministic digest of the catalog, for signing
  exclave verify                      check the catalog against a signed manifest
  exclave redact                      roll-up with site identities removed

Flags:
  -catalog dir             release catalog (default "catalog")
  -fleet dir               environment descriptors (default "fleet/environments")
  -format text|json        output format (default "text")
  -max-classification lvl  refuse descriptors above this level (il2, il4, il5, il6)
  -manifest file           manifest to verify against
  -keep-classification     include classification in redacted output (cleared receivers only)

Redaction requires EXCLAVE_REDACTION_SALT. Signing a manifest is external:
  exclave manifest > catalog.manifest && cosign sign-blob catalog.manifest
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// options holds every flag the command accepts.
type options struct {
	catalogDir   *string
	fleetDir     *string
	format       *string
	maxClass     *string
	manifestFile *string
	keepClass    *bool
}

// newFlagSet builds the command's flags. Tests use this rather than declaring a
// lookalike, so splitArgs is always exercised against the real flag set — a copy
// would drift the first time someone adds a flag, which is exactly how the
// boolean-flag bug got in.
func newFlagSet() (*flag.FlagSet, *options) {
	fs := flag.NewFlagSet("exclave", flag.ContinueOnError)
	o := &options{
		catalogDir:   fs.String("catalog", "catalog", "release catalog directory"),
		fleetDir:     fs.String("fleet", "fleet/environments", "environment descriptor directory"),
		format:       fs.String("format", "text", "output format: text or json"),
		maxClass:     fs.String("max-classification", "", "refuse descriptors above this level"),
		manifestFile: fs.String("manifest", "", "manifest file to verify against"),
		keepClass:    fs.Bool("keep-classification", false, "include classification in redacted output"),
	}
	return fs, o
}

func run(args []string) error {
	fs, opt := newFlagSet()
	catalogDir, fleetDir := opt.catalogDir, opt.fleetDir
	format, maxClass := opt.format, opt.maxClass
	manifestFile, keepClass := opt.manifestFile, opt.keepClass
	// The flag package writes its own errors and usage; we want --help on stdout
	// with exit 0, and a single error line otherwise, so we own all output.
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}

	cmd, flags, positional := splitArgs(fs, args)

	if err := fs.Parse(flags); err != nil {
		// -h and --help start with a dash, so they never become the subcommand.
		// They reach fs.Parse, which returns ErrHelp — propagating that as an
		// error made `exclave --help` exit 1 with "error: flag: help requested".
		if errors.Is(err, flag.ErrHelp) {
			fmt.Print(usage)
			return nil
		}
		fmt.Fprint(os.Stderr, usage, "\n")
		return err
	}
	if *format != "text" && *format != "json" {
		return fmt.Errorf("unknown format %q (want text or json)", *format)
	}

	switch cmd {
	case "plan":
		return cmdPlan(*catalogDir, *fleetDir, *maxClass, *format)
	case "explain":
		if len(positional) < 1 {
			return fmt.Errorf("explain needs an environment name")
		}
		version := ""
		if len(positional) > 1 {
			version = positional[1]
		}
		return cmdExplain(*catalogDir, *fleetDir, *maxClass, positional[0], version)
	case "validate":
		return cmdValidate(*catalogDir, *fleetDir, *maxClass)
	case "manifest":
		return cmdManifest(*catalogDir)
	case "verify":
		return cmdVerify(*catalogDir, *manifestFile)
	case "redact":
		return cmdRedact(*catalogDir, *fleetDir, *maxClass, *format, *keepClass)
	case "", "help":
		fmt.Print(usage)
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\n%s", cmd, usage)
	}
}

// splitArgs separates the subcommand, the flags and the positional arguments.
//
// Go's flag package stops parsing at the first non-flag argument, so flags after
// a positional would be silently ignored. Splitting them first lets
// `exclave explain army-abc-il5 4.3.0 -catalog dir` work the way anyone expects.
//
// The subtlety is boolean flags: consuming the next argument as their value eats
// a positional. `fs.Lookup` is the authority on which flags are boolean, rather
// than a hand-maintained list that drifts the moment someone adds a flag.
func splitArgs(fs *flag.FlagSet, args []string) (cmd string, flags, positional []string) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}

	isBool := func(a string) bool {
		f := fs.Lookup(strings.TrimLeft(a, "-"))
		if f == nil {
			return false
		}
		b, ok := f.Value.(interface{ IsBoolFlag() bool })
		return ok && b.IsBoolFlag()
	}

	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") {
			positional = append(positional, a)
			continue
		}
		flags = append(flags, a)
		if !strings.Contains(a, "=") && !isBool(a) && i+1 < len(args) {
			flags = append(flags, args[i+1])
			i++
		}
	}
	return cmd, flags, positional
}

func load(catalogDir, fleetDir, maxClass string) ([]catalog.Release, []fleet.Environment, error) {
	releases, err := catalog.Load(catalogDir)
	if err != nil {
		return nil, nil, err
	}
	envs, err := fleet.Load(fleetDir, maxClass)
	if err != nil {
		return nil, nil, err
	}
	return releases, envs, nil
}

type planRow struct {
	Environment    string `json:"environment"`
	Classification string `json:"classification,omitempty"`
	Current        string `json:"current,omitempty"`
	Target         string `json:"target,omitempty"`
	Status         string `json:"status"`
	Note           string `json:"note,omitempty"`
}

func cmdPlan(catalogDir, fleetDir, maxClass, format string) error {
	releases, envs, err := load(catalogDir, fleetDir, maxClass)
	if err != nil {
		return err
	}
	decisions := resolve.Plan(releases, envs)

	if format == "json" {
		rows := make([]planRow, 0, len(decisions))
		for _, d := range decisions {
			rows = append(rows, planRow{
				Environment:    d.Environment.Name,
				Classification: d.Environment.Classification,
				Current:        d.Environment.Current,
				Target:         d.Target,
				Status:         string(d.Status),
				Note:           d.Note,
			})
		}
		return writeJSON(rows)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ENVIRONMENT\tCURRENT\tTARGET\tSTATUS")
	for _, d := range decisions {
		status := string(d.Status)
		if d.Note != "" {
			status = fmt.Sprintf("%s (%s)", status, d.Note)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			d.Environment.Name, orDash(d.Environment.Current), orDash(d.Target), status)
	}
	return w.Flush()
}

func cmdRedact(catalogDir, fleetDir, maxClass, format string, keepClass bool) error {
	salt := os.Getenv("EXCLAVE_REDACTION_SALT")
	releases, envs, err := load(catalogDir, fleetDir, maxClass)
	if err != nil {
		return err
	}
	rows, err := resolve.Redact(resolve.Plan(releases, envs), salt, keepClass)
	if err != nil {
		return err
	}

	if format == "json" {
		return writeJSON(rows)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	header := "SITE\tCURRENT\tTARGET\tSTATUS"
	if keepClass {
		header = "SITE\tLEVEL\tCURRENT\tTARGET\tSTATUS"
	}
	fmt.Fprintln(w, header)
	for _, r := range rows {
		status := r.Status
		if r.Note != "" {
			status = fmt.Sprintf("%s (%s)", status, r.Note)
		}
		if keepClass {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.SiteID, r.Classification, orDash(r.Current), orDash(r.Target), status)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.SiteID, orDash(r.Current), orDash(r.Target), status)
		}
	}
	return w.Flush()
}

func cmdManifest(catalogDir string) error {
	m, err := catalog.BuildManifest(catalogDir)
	if err != nil {
		return err
	}
	fmt.Print(m)
	return nil
}

func cmdVerify(catalogDir, manifestFile string) error {
	if manifestFile == "" {
		return fmt.Errorf("verify needs -manifest <file>")
	}
	b, err := os.ReadFile(manifestFile)
	if err != nil {
		return err
	}
	drift, err := catalog.VerifyManifest(catalogDir, string(b))
	if err != nil {
		return err
	}
	if len(drift) == 0 {
		fmt.Printf("ok: %s matches %s\n", catalogDir, manifestFile)
		return nil
	}
	// Name the files. "Does not match its signature" sends someone hunting.
	fmt.Fprintf(os.Stderr, "catalog does not match %s:\n", manifestFile)
	for _, d := range drift {
		fmt.Fprintf(os.Stderr, "  %-24s %s\n", d.Reason, d.Path)
	}
	return fmt.Errorf("%d file(s) drifted from the signed manifest", len(drift))
}

func cmdExplain(catalogDir, fleetDir, maxClass, envName, version string) error {
	releases, envs, err := load(catalogDir, fleetDir, maxClass)
	if err != nil {
		return err
	}

	var env *fleet.Environment
	for i := range envs {
		if envs[i].Name == envName {
			env = &envs[i]
			break
		}
	}
	if env == nil {
		return fmt.Errorf("no environment named %q", envName)
	}

	fmt.Printf("%s — tier %s, classification %s, channel %s, kubernetes %s",
		env.Name, env.Tier, orDash(env.Classification), env.Channel, env.Kubernetes)
	if env.Schema > 0 {
		fmt.Printf(", schema %d", env.Schema)
	}
	if env.MaxCriticalCVEs != nil {
		fmt.Printf(", max %d critical CVEs", *env.MaxCriticalCVEs)
	}
	fmt.Println()

	fmt.Printf("installed: %s", orDash(env.Current))
	if env.Pinned != "" {
		fmt.Printf("   pinned: %s", env.Pinned)
	}
	if env.MaintenanceWindow != "" {
		fmt.Printf("   window: %s", env.MaintenanceWindow)
	}
	if len(env.RequiresCapabilities) > 0 {
		keys := make([]string, 0, len(env.RequiresCapabilities))
		for k := range env.RequiresCapabilities {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Printf("\nrequires:  %s", strings.Join(keys, ", "))
	}
	fmt.Print("\n\n")

	matched := false
	for i := len(releases) - 1; i >= 0; i-- {
		r := releases[i]
		if version != "" && r.Version != version {
			continue
		}
		matched = true

		ev := resolve.Evaluate(r, *env)
		verdict := "ELIGIBLE"
		if !ev.Eligible {
			verdict = "BLOCKED"
		}
		fmt.Printf("%s  %s  (channel %s)\n", verdict, r.Version, r.Channel)
		for _, c := range ev.Checks {
			mark := "ok  "
			if !c.OK {
				mark = "FAIL"
			}
			fmt.Printf("    %s  %-14s %s\n", mark, c.Name, c.Detail)
		}
		fmt.Println()
	}
	if !matched {
		return fmt.Errorf("no release %q in the catalog", version)
	}
	return nil
}

func cmdValidate(catalogDir, fleetDir, maxClass string) error {
	releases, envs, err := load(catalogDir, fleetDir, maxClass)
	if err != nil {
		return err
	}
	fmt.Printf("ok: %d releases in %s, %d environments in %s",
		len(releases), catalogDir, len(envs), fleetDir)
	if maxClass != "" {
		fmt.Printf(" (all at or below %s)", maxClass)
	}
	fmt.Println()
	return nil
}

func writeJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
