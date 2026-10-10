package cmd

import (
	"context"
	"errors"
	"os"
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
	// goclikit resolves the command from the root's last parsed arguments where
	// it has any, ahead of os.Args, and rootCmd is one value for the whole test
	// binary. Left alone, each line would resolve against the one before it.
	if err := rootCmd.Flags().Parse(nil); err != nil {
		t.Fatal(err)
	}
	return execute(context.Background(), autoupdate.Config{Suppress: true})
}

// Each line names a word helpnav cannot act on, and is answered with that word
// and exit 2 rather than with a screen or a flag.
func TestAMistypedWordIsRefusedByName(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"no-such-tool"}, `"no-such-tool" is not a command on PATH`},
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
