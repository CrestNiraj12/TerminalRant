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
		{name: "invalid flag", args: []string{"--bogus", "--pogus"}, mode: cliInvalid, msg: "unexpected argument: --bogus --pogus"},
		{name: "too many args", args: []string{"--version", "extra"}, mode: cliVersion},
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
