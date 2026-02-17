package main

import (
	"testing"
)

func TestParseCLIArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		mode cliMode
		msg  string
	}{
		{name: "run default", args: nil, mode: cliRun},
		{name: "version long", args: []string{"--version"}, mode: cliVersion},
		{name: "version short", args: []string{"-v"}, mode: cliVersion},
		{name: "version single-dash", args: []string{"-version"}, mode: cliVersion},
		{name: "help long", args: []string{"--help"}, mode: cliHelp},
		{name: "help short", args: []string{"-h"}, mode: cliHelp},
		{name: "help word", args: []string{"help"}, mode: cliHelp},
		{name: "invalid flag", args: []string{"--bogus"}, mode: cliInvalid, msg: "unexpected argument: --bogus"},
		{name: "invalid flags", args: []string{"--bogus", "--pogus"}, mode: cliInvalid, msg: "unexpected argument: --bogus --pogus"},
		{name: "valid with invalid after", args: []string{"--version", "extra"}, mode: cliVersion},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mode, msg := parseCLIArgs(tc.args)
			if mode != tc.mode {
				t.Fatalf("mode mismatch: got %v want %v", mode, tc.mode)
			}
			if tc.msg != "" && msg != tc.msg {
				t.Fatalf("msg mismatch: got %q want %q", msg, tc.msg)
			}
		})
	}
}

func TestResolveVersionInfo_TaggedModuleVersion(t *testing.T) {
	v, c, d := resolveVersionInfo(
		"dev",
		"none",
		"unknown",
		"v0.4.0",
		map[string]string{
			"vcs.revision": "0123456789abcdef",
			"vcs.time":     "2026-02-17T00:00:00Z",
		},
	)
	if v != "v0.4.0" {
		t.Fatalf("expected tagged module version, got %q", v)
	}
	if c != "0123456789ab" {
		t.Fatalf("expected shortened revision, got %q", c)
	}
	if d != "2026-02-17T00:00:00Z" {
		t.Fatalf("expected vcs time, got %q", d)
	}
}

func TestResolveVersionInfo_DevelModuleVersionKeepsDev(t *testing.T) {
	v, c, d := resolveVersionInfo(
		"dev",
		"none",
		"unknown",
		"(devel)",
		map[string]string{
			"vcs.revision": "abc123",
			"vcs.time":     "2026-02-17T00:00:00Z",
		},
	)
	if v != "dev" {
		t.Fatalf("expected dev version to remain for devel module, got %q", v)
	}
	if c != "abc123" {
		t.Fatalf("expected revision fallback, got %q", c)
	}
	if d != "2026-02-17T00:00:00Z" {
		t.Fatalf("expected vcs time fallback, got %q", d)
	}
}
