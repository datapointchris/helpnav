package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/datapointchris/goclikit"
	"github.com/datapointchris/goselfupdate/autoupdate"
)

// runLine drives the command line through execute, the path main takes, with
// the update check suppressed. goclikit resolves the command from os.Args, so
// the line goes there rather than through SetArgs.
func runLine(t *testing.T, args ...string) error {
	t.Helper()
	original := os.Args
	os.Args = append([]string{"helpnav"}, args...)
	t.Cleanup(func() { os.Args = original })
	return execute(context.Background(), autoupdate.Config{Suppress: true})
}

// Each line names a word helpnav cannot act on, or asks for a screen with no
// terminal to draw it on, and is answered in words with exit 2 rather than with
// a screen or a flag.
func TestAMistypedWordIsRefusedByName(t *testing.T) {
	bin := t.TempDir()
	t.Setenv("PATH", bin)
	if err := os.WriteFile(filepath.Join(bin, "tool"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A terminal running the suite would otherwise be the browser's.
	stdin := os.Stdin
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = devNull
	t.Cleanup(func() { os.Stdin = stdin; _ = devNull.Close() })
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"tui", "no-such-tool"}, `"no-such-tool" is not a command on PATH`},
		{[]string{"tui", "tool"}, "helpnav tui needs a terminal; --once prints one frame and --json the command tree"},
		{[]string{"shell", "bogus"}, `unknown command "bogus" for "helpnav shell"`},
		{[]string{"shell", "bogus", "--json"}, `unknown command "bogus" for "helpnav shell"`},
		{[]string{"shell", "bogus", "--help"}, `unknown command "bogus" for "helpnav shell"`},
	} {
		err := runLine(t, c.args...)
		if !errors.Is(err, goclikit.ErrUsage) {
			t.Errorf("%v: error is not a usage error: %v", c.args, err)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%v answered %q, want it to carry %q", c.args, err, c.want)
		}
	}
}
