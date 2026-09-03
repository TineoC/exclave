// Command exclave resolves which product release each environment in a fleet is
// eligible for, and explains every exclusion.
package main

import (
	"flag"
	"fmt"
	"os"
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

Flags:
  -catalog dir   release catalog (default "catalog")
  -fleet dir     environment descriptors (default "fleet/environments")
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("exclave", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	catalogDir := fs.String("catalog", "catalog", "release catalog directory")
	fleetDir := fs.String("fleet", "fleet/environments", "environment descriptor directory")
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }

	// Allow flags on either side of the positional arguments. Go's flag package
	// stops at the first non-flag argument, so split them apart first.
	// Every flag this command defines takes a value; revisit if a boolean is added.
	var cmd string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") {
			positional = append(positional, a)
			continue
		}
		flags = append(flags, a)
		if !strings.Contains(a, "=") && i+1 < len(args) {
			flags = append(flags, args[i+1])
			i++
		}
	}
	if err := fs.Parse(flags); err != nil {
		return err
	}

	switch cmd {
	case "plan":
		return cmdPlan(*catalogDir, *fleetDir)
	case "explain":
		if len(positional) < 1 {
			return fmt.Errorf("explain needs an environment name")
		}
		version := ""
		if len(positional) > 1 {
			version = positional[1]
		}
		return cmdExplain(*catalogDir, *fleetDir, positional[0], version)
	case "validate":
		return cmdValidate(*catalogDir, *fleetDir)
	case "", "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\n%s", cmd, usage)
	}
}

func load(catalogDir, fleetDir string) ([]catalog.Release, []fleet.Environment, error) {
	releases, err := catalog.Load(catalogDir)
	if err != nil {
		return nil, nil, err
	}
	envs, err := fleet.Load(fleetDir)
	if err != nil {
		return nil, nil, err
	}
	return releases, envs, nil
}

func cmdPlan(catalogDir, fleetDir string) error {
	releases, envs, err := load(catalogDir, fleetDir)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ENVIRONMENT\tCURRENT\tTARGET\tSTATUS")

	for _, d := range resolve.Plan(releases, envs) {
		current := d.Environment.Current
		if current == "" {
			current = "—"
		}
		target := d.Target
		if target == "" {
			target = "—"
		}
		status := string(d.Status)
		if d.Note != "" {
			status = fmt.Sprintf("%s (%s)", status, d.Note)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", d.Environment.Name, current, target, status)
	}
	return w.Flush()
}

func cmdExplain(catalogDir, fleetDir, envName, version string) error {
	releases, envs, err := load(catalogDir, fleetDir)
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

	fmt.Printf("%s — tier %s, classification %s, channel %s, kubernetes %s, schema %d\n",
		env.Name, env.Tier, orDash(env.Classification), env.Channel, env.Kubernetes, env.Schema)
	fmt.Printf("installed: %s", orDash(env.Current))
	if env.Pinned != "" {
		fmt.Printf("   pinned: %s", env.Pinned)
	}
	if env.MaintenanceWindow != "" {
		fmt.Printf("   window: %s", env.MaintenanceWindow)
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

func cmdValidate(catalogDir, fleetDir string) error {
	releases, envs, err := load(catalogDir, fleetDir)
	if err != nil {
		return err
	}
	fmt.Printf("ok: %d releases in %s, %d environments in %s\n",
		len(releases), catalogDir, len(envs), fleetDir)
	return nil
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
