package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// The Homebrew formula tests `swrite --version`, and `make verify-release`
// requires the packaged binary's --version to contain the tag. swrite answered
// neither form until v0.5.0: main.version was injected and never used.
func TestVersionFlagAndSubcommandPrintTheSameLine(t *testing.T) {
	// A config file that does not parse: loading it fails the root's pre-run
	// hook, so both forms passing proves neither reads the config.
	bad := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(bad, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SWRITE_MODE", "")

	want := "swrite version v9.9.9\n"
	for _, args := range [][]string{{"--version"}, {"version"}, {"--config", bad, "version"}} {
		root := newRootCmd("v9.9.9")
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs(args)
		if err := root.Execute(); err != nil {
			t.Fatalf("%v: %v (output %q)", args, err, out.String())
		}
		if got := out.String(); got != want {
			t.Errorf("%v printed %q, want %q", args, got, want)
		}
	}
}

// The malformed config does fail a command that loads it; without this the
// test above could pass because nothing ever read the file.
func TestMalformedConfigFailsACommandThatLoadsIt(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(bad, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SWRITE_MODE", "")

	root := newRootCmd("v9.9.9")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", bad, "cache", "clear"})
	if err := root.Execute(); err == nil {
		t.Fatalf("cache clear with a malformed config succeeded (output %q)", out.String())
	}
}
