package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var doNotRunCmd = &cobra.Command{
	Use:     "do-not-run",
	GroupID: groupTool,
	Short:   "List the tools helpnav refuses to read, and where to change that",
	Long: `List the tools helpnav refuses to read.

Reading a tool means starting it, and a few programs answer --help by opening a
window instead of printing. Those are named here so nobody has to find out by
waiting twenty seconds for one.

The list is written by hand. Add to it in the file named at the end of the
output: one tool per line, a reason after the name, and ` + "`#`" + ` for a comment.`,
	Args:          cobra.NoArgs,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		out := cmd.OutOrStdout()
		for _, row := range doNotRunList() {
			if _, err := fmt.Fprintln(out, row); err != nil {
				return err
			}
		}
		_, err := fmt.Fprintf(out, "\nadd to this list in %s\n", doNotRunPath())
		return err
	},
}

func init() { rootCmd.AddCommand(doNotRunCmd) }

// Reading a tool means starting it, and a few programs answer `--help` by
// launching instead of printing. Measured across the 176 CLIs this machine
// declares: bitwarden took 1291ms and claude-desktop 20036ms, both showing no
// commands for it, against a median of 6ms and a slowest legitimate flat read
// of 380ms. Nothing sits in that gap.
//
// The list is written by hand rather than learned. A threshold would have to
// run the tool once to find out, which is the cost it exists to avoid, and a
// person already knows a desktop app is not a CLI before it wastes twenty
// seconds.
//
// A desktop entry is not the test. Twelve of the 176 have one, and ten of them
// — htop, btop, mpv, kitty, ghostty, aerc, sioyek, rofi, vivaldi, zen-browser —
// print help correctly in under 100ms.
var builtinDoNotRun = map[string]string{
	"bitwarden":      "the desktop app; it ignores --help and opens a window",
	"claude-desktop": "an Electron app; it ignores --help and opens a window",
}

// doNotRunPath is where a person adds to the list without waiting for a
// release. One name per line, `#` starts a comment, and a reason after the name
// is shown when the tool is refused.
//
// Plain text rather than YAML or TOML: the file is a list of names, and the
// comment syntax already carries the explanation a structured format would be
// chosen for. Neither would earn the parser it costs.
func doNotRunPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "helpnav", "do-not-run.txt")
}

// doNotRun reports why a tool must not be read, and whether it is listed at all.
//
// A missing or unreadable file is not an error. The built-in list still stands,
// and refusing to browse anything because a config file is absent would be a
// worse failure than the one this prevents.
func doNotRun(binary string) (string, bool) {
	if reason, listed := readDoNotRunFile()[binary]; listed {
		return reason, true
	}
	reason, listed := builtinDoNotRun[binary]
	return reason, listed
}

func readDoNotRunFile() map[string]string {
	listed := map[string]string{}
	path := doNotRunPath()
	if path == "" {
		return listed
	}
	file, err := os.Open(path)
	if err != nil {
		return listed
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		name, reason := parseDoNotRunLine(scanner.Text())
		if name != "" {
			listed[name] = reason
		}
	}
	return listed
}

// parseDoNotRunLine splits one line into the tool and the reason beside it.
// Returns an empty name for a blank line or a comment.
func parseDoNotRunLine(line string) (name, reason string) {
	if cut := strings.IndexByte(line, '#'); cut >= 0 {
		line = line[:cut]
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return "", ""
	}
	name, reason, _ = strings.Cut(line, " ")
	return name, strings.TrimSpace(reason)
}

// refuseToRun is the error a listed tool produces, naming the file so the
// person who disagrees knows where to say so.
func refuseToRun(binary string) error {
	reason, _ := doNotRun(binary)
	if reason == "" {
		reason = "listed as a tool that must not be run"
	}
	return fmt.Errorf("helpnav: refusing to read %q: %s\nreading it means running it; edit %s to change that",
		binary, reason, doNotRunPath())
}

// doNotRunList is every listed tool with its reason, for `helpnav do-not-run`.
func doNotRunList() []string {
	merged := map[string]string{}
	for name, reason := range builtinDoNotRun {
		merged[name] = reason
	}
	for name, reason := range readDoNotRunFile() {
		merged[name] = reason
	}

	names := make([]string, 0, len(merged))
	for name := range merged {
		names = append(names, name)
	}
	sort.Strings(names)

	rows := make([]string, 0, len(names))
	for _, name := range names {
		rows = append(rows, fmt.Sprintf("%-18s %s", name, merged[name]))
	}
	return rows
}
