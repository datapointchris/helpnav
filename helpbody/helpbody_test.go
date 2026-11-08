package helpbody

import (
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func names(b Body) []string {
	var out []string
	for _, s := range b.sections {
		out = append(out, s.Name)
	}
	return out
}

// Each body is cut from what the tool prints, colors included.
func TestParseFindsEachToolsHeadingsAndNoProse(t *testing.T) {
	for _, tc := range []struct {
		tool, body string
		want       []string
	}{
		{"clap", "Add dependencies to the project\n\n\x1b[1m\x1b[32mUsage:\x1b[0m \x1b[1m\x1b[36muv add\x1b[0m [OPTIONS]\n\n\x1b[1m\x1b[32mArguments:\x1b[0m\n  [PACKAGES]...  The packages to add\n\n\x1b[1m\x1b[32mCache options:\x1b[0m\n  -n, --no-cache\n          Avoid reading from the cache\n", []string{"Arguments", "Cache options"}},
		{"docker", "Usage:  docker run [OPTIONS] IMAGE\n\nCreate and run a new container\n\nAliases:\n  docker container run, docker run\n\nOptions:\n      --add-host list   Add a custom host\n", []string{"Aliases", "Options"}},
		{"gh", "Work with GitHub pull requests.\n\n\x1b[0;1;39mUSAGE\x1b[0m\n  gh pr <command> [flags]\n\n\x1b[0;1;39mGENERAL COMMANDS\x1b[0m\n  create:        Create a pull request\n", []string{"USAGE", "GENERAL COMMANDS"}},
		{"git", "See 'git help <command>' to read about a specific subcommand\n\nMain Porcelain Commands\n   add                     Add file contents to the index\n", []string{"Main Porcelain Commands"}},
		{"prose", "The grammar is one word: helpnav TOOL.\n  helpnav docker\n\nFor more, read this paragraph.\n  helpnav uv\n", nil},
	} {
		if got := names(Parse(tc.body)); !slices.Equal(got, tc.want) {
			t.Errorf("%s: sections = %q, want %q", tc.tool, got, tc.want)
		}
	}
}

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

func TestALineWiderThanThePaneHangsUnderItsDescription(t *testing.T) {
	for _, tc := range []struct {
		line  string
		width int
		hang  int
	}{
		{"      --allow stringArray      Allow build to access specified resources", 50, 31},
		{"      --no-extra <NO_EXTRA>                      Don't", 33, 23},
		{"    # image.  If 'tar' is not present, 'kubectl cp' will fail.", 30, 4},
	} {
		rows := wrap(tc.line, tc.width)
		if len(rows) < 2 {
			t.Fatalf("%q at %d did not wrap", tc.line, tc.width)
		}
		for _, row := range rows {
			if w := ansi.StringWidth(row); w > tc.width {
				t.Errorf("row %q is %d wide, past %d", row, w, tc.width)
			}
		}
		for _, row := range rows[1:] {
			if indent := len(row) - len(strings.TrimLeft(row, " ")); indent != tc.hang {
				t.Errorf("continuation %q hangs at %d, want %d", row, indent, tc.hang)
			}
		}
	}
}
