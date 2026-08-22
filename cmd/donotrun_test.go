package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestALineSplitsIntoToolAndReason(t *testing.T) {
	cases := []struct{ line, name, reason string }{
		{"obsidian   an Electron note app", "obsidian", "an Electron note app"},
		{"  bitwarden  opens a window  ", "bitwarden", "opens a window"},
		{"figma", "figma", ""},
		{"slack # not a CLI", "slack", ""},
		{"# a whole-line comment", "", ""},
		{"   ", "", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		name, reason := parseDoNotRunLine(c.line)
		if name != c.name || reason != c.reason {
			t.Errorf("%q -> (%q, %q), want (%q, %q)", c.line, name, reason, c.name, c.reason)
		}
	}
}

func TestTheListLivesUnderXDGConfigWithAnExtension(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg")

	if got, want := doNotRunPath(), "/xdg/helpnav/do-not-run.txt"; got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
}

// The built-in list stands on a machine with no config file, because a desktop
// app is not a CLI whether or not anyone has written that down yet.
func TestTheBuiltinListStandsWithoutAConfigFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if _, listed := doNotRun("claude-desktop"); !listed {
		t.Error("claude-desktop was not listed")
	}
	if _, listed := doNotRun("docker"); listed {
		t.Error("docker was listed; an ordinary CLI must stay readable")
	}
}

func TestTheFileAddsToTheBuiltinList(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(filepath.Join(dir, "helpnav"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# mine\nobsidian   an Electron note app\n\nfigma\n"
	if err := os.WriteFile(filepath.Join(dir, "helpnav", "do-not-run.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	if reason, listed := doNotRun("obsidian"); !listed || reason != "an Electron note app" {
		t.Errorf("obsidian -> (%q, %v), want the file's reason", reason, listed)
	}
	if _, listed := doNotRun("figma"); !listed {
		t.Error("a name with no reason was not listed")
	}
	if _, listed := doNotRun("claude-desktop"); !listed {
		t.Error("the file replaced the built-in list instead of adding to it")
	}
}

// A machine with no config file, or an unreadable one, must still browse. The
// failure this guards against is worse than the one it prevents.
func TestAnUnreadableFileIsNotAnError(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "does-not-exist"))

	if _, listed := doNotRun("docker"); listed {
		t.Error("docker was refused because the config file was missing")
	}
	if len(doNotRunList()) != len(builtinDoNotRun) {
		t.Error("the built-in list did not survive a missing file")
	}
}
