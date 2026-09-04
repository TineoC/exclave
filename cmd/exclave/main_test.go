package main

import (
	"reflect"
	"testing"
)

// The boolean-flag bug shipped because the parser was changed and never
// exercised. These pin the two behaviours that broke.
func TestSplitArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		cmd     string
		flags   []string
		posargs []string
	}{
		{
			name:  "value flag pulls its value along",
			args:  []string{"plan", "-catalog", "cat", "-fleet", "flt"},
			cmd:   "plan",
			flags: []string{"-catalog", "cat", "-fleet", "flt"},
		},
		{
			// The regression: a boolean flag must not consume the next argument.
			name:    "boolean flag does not eat the following positional",
			args:    []string{"explain", "site-a", "4.3.0", "--keep-classification"},
			cmd:     "explain",
			flags:   []string{"--keep-classification"},
			posargs: []string{"site-a", "4.3.0"},
		},
		{
			name:    "boolean flag before a positional",
			args:    []string{"explain", "--keep-classification", "site-a"},
			cmd:     "explain",
			flags:   []string{"--keep-classification"},
			posargs: []string{"site-a"},
		},
		{
			name:  "boolean followed by a value flag",
			args:  []string{"redact", "--keep-classification", "-format", "json"},
			cmd:   "redact",
			flags: []string{"--keep-classification", "-format", "json"},
		},
		{
			// Flags after positionals is the reason splitArgs exists: Go's flag
			// package stops at the first non-flag argument.
			name:    "flags after positionals are still collected",
			args:    []string{"explain", "site-a", "4.3.0", "-catalog", "cat"},
			cmd:     "explain",
			flags:   []string{"-catalog", "cat"},
			posargs: []string{"site-a", "4.3.0"},
		},
		{
			name:  "equals form is not given a separate value",
			args:  []string{"plan", "-format=json"},
			cmd:   "plan",
			flags: []string{"-format=json"},
		},
		{
			name:  "a leading dash means no subcommand",
			args:  []string{"--help"},
			cmd:   "",
			flags: []string{"--help"},
		},
		{
			name: "no arguments at all",
			args: nil,
			cmd:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, _ := newFlagSet()
			cmd, flags, positional := splitArgs(fs, tt.args)

			if cmd != tt.cmd {
				t.Errorf("cmd = %q, want %q", cmd, tt.cmd)
			}
			if !reflect.DeepEqual(flags, tt.flags) && !(len(flags) == 0 && len(tt.flags) == 0) {
				t.Errorf("flags = %v, want %v", flags, tt.flags)
			}
			if !reflect.DeepEqual(positional, tt.posargs) && !(len(positional) == 0 && len(tt.posargs) == 0) {
				t.Errorf("positional = %v, want %v", positional, tt.posargs)
			}
		})
	}
}

// -h and --help start with a dash, so they never become the subcommand and the
// switch arms that named them were unreachable. They reach fs.Parse, which
// returns flag.ErrHelp — returning that as an error made `exclave --help` exit 1.
func TestHelpExitsCleanly(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"--help"}, {"help"}, {}} {
		name := "(none)"
		if len(args) > 0 {
			name = args[0]
		}
		t.Run(name, func(t *testing.T) {
			if err := run(args); err != nil {
				t.Errorf("run(%v) = %v, want nil — asking for help is not a failure", args, err)
			}
		})
	}
}

func TestBadInputStillFails(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"unknown flag", []string{"plan", "-bogus", "x"}},
		{"unknown subcommand", []string{"frobnicate"}},
		{"unknown format", []string{"plan", "-format", "xml"}},
		{"verify without a manifest", []string{"verify"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := run(tt.args); err == nil {
				t.Errorf("run(%v) = nil, want an error", tt.args)
			}
		})
	}
}
