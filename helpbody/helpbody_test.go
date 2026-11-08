package helpbody

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func section(name, line string, n int) string {
	return name + ":\n" + strings.Repeat(line+"\n", n)
}

func TestSpaceClosesWhatItOpenedWhileReadingInsideASection(t *testing.T) {
	folds := map[string]bool{}
	body := Parse("intro\n\n" + section("Options", "  --flag", 10) + "\n" + section("Global options", "  --quiet", 10))
	r := NewReading(body, folds, 40, 5)

	r.Scroll(5)
	if s, ok := r.Sticky(); !ok || s.Name != "Options" {
		t.Fatalf("reading inside Options, the band names %q", s.Name)
	}

	r.Toggle()
	if first := r.Window()[0]; first.Mark != Folded || first.Hidden != 10 {
		t.Fatalf("after space the top row is %+v, want Options folded over 10 lines", first)
	}
	if s, _ := r.Target(); s.Name != "Options" {
		t.Fatalf("after folding, space acts on %q, want Options", s.Name)
	}

	next := NewReading(Parse(section("Options", "  --other", 3)), folds, 40, 5)
	if first := next.Window()[0]; first.Mark != Folded {
		t.Errorf("the next command's Options is %+v, want it folded as well", first)
	}

	r.Toggle()
	if first := r.Window()[0]; first.Mark != Open {
		t.Errorf("a second space left the top row %+v, want Options open again", first)
	}
}

// uv run's help ends in Global options, which a reader pages to with f and
// folds with space.
func TestSpaceReopensTheLastSectionFoldedFromItsEnd(t *testing.T) {
	body := Parse("Run a command\n\n" + section("Cache options", "  --no-cache", 10) + "\n" + section("Global options", "  --quiet", 36))
	r := NewReading(body, map[string]bool{}, 40, 20)
	r.Scroll(1000)

	r.Toggle()
	if s, _ := r.Target(); s.Name != "Global options" {
		t.Fatalf("after folding at the end, space acts on %q, want Global options", s.Name)
	}
	r.Toggle()
	if first := r.Window()[0]; first.Mark != Open {
		t.Errorf("a second space left the top row %+v, want Global options open again", first)
	}
}

func TestALongDescriptionReadsDownOneColumnAtThePanesWidth(t *testing.T) {
	clap := func(word string) string { return strings.Repeat(" ", 65) + word }
	for _, tc := range []struct {
		scene string
		lines []string
		width int
		hang  int
		whole string
	}{
		{
			"docker's flag column beside a 60-cell pane",
			[]string{"      --allow stringArray      Allow build to access specified resources"},
			60, 31, "resources",
		},
		{
			"uv run read at 56 cells for a 100-column terminal, one word a line at 65",
			[]string{"      --no-binary" + strings.Repeat(" ", 48) + "Don't", clap("install"), clap("pre-built"), clap("wheels"), clap("[env:"), clap("UV_NO_BINARY=]")},
			56, 10, "UV_NO_BINARY=]",
		},
		{
			"kubectl's prose with two spaces after a period",
			[]string{"    # image.  If 'tar' is not present, 'kubectl cp' will fail."},
			30, 4, "present,",
		},
	} {
		rows, used := fit(tc.lines, tc.width)
		if used != len(tc.lines) {
			t.Errorf("%s: used %d lines, want %d", tc.scene, used, len(tc.lines))
		}
		found := false
		for i, row := range rows {
			if w := ansi.StringWidth(row); w > tc.width {
				t.Errorf("%s: row %q is %d wide, past %d", tc.scene, row, w, tc.width)
			}
			if i > 0 {
				if indent := len(row) - len(strings.TrimLeft(row, " ")); indent != tc.hang {
					t.Errorf("%s: row %q hangs at %d, want %d", tc.scene, row, indent, tc.hang)
				}
			}
			found = found || strings.Contains(row, tc.whole)
		}
		if !found {
			t.Errorf("%s: no row holds %q whole: %q", tc.scene, tc.whole, rows)
		}
	}
}
