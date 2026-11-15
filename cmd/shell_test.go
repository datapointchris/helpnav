package cmd

import (
	"bytes"
	"errors"
	"testing"

	"github.com/datapointchris/goclikit"
)

// Every shell in widgetBlocks prints its own block, and a shell outside it is
// refused as a usage error.
func TestShellWidgetsPrintsTheBlockForEachShellAndRefusesOthers(t *testing.T) {
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	t.Cleanup(func() { rootCmd.SetOut(nil) })

	for shell, block := range widgetBlocks {
		out.Reset()
		if err := runLine(t, "shell", "widgets", shell); err != nil {
			t.Fatalf("%s: %v", shell, err)
		}
		if out.String() != block {
			t.Errorf("%s printed a block other than its own", shell)
		}
	}

	if err := runLine(t, "shell", "widgets", "no-such-shell"); !errors.Is(err, goclikit.ErrUsage) {
		t.Errorf("an unknown shell is not a usage error: %v", err)
	}
}
