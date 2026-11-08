package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"github.com/datapointchris/clisurface"
	"github.com/datapointchris/goclikit"
	"github.com/spf13/cobra"

	"github.com/datapointchris/helpnav/tui"
)

var (
	depthFlag int
	onceFlag  bool
	jsonFlag  bool
)

// errNoTerminal refuses a browser nobody could drive, rather than waiting on a
// keyboard nobody holds.
var errNoTerminal = errors.New("helpnav tui needs a terminal; --once prints one frame and --json the command tree")

// The frame --once draws where COLUMNS or LINES is unset.
const (
	onceFallbackWidth  = 120
	onceFallbackHeight = 40
)

var tuiCmd = &cobra.Command{
	Use:     "tui <tool>",
	GroupID: groupBrowse,
	Short:   "Browse one tool's help as a tree, and print the command you stop on",
	Long: `Browse one tool's help as a tree, and print the command you stop on.

Three columns show where you came from, the commands here, and the help of the
one under the cursor. enter prints that command on stdout and exits, and q exits
printing nothing. The screen draws on stderr, so ` + "`$(helpnav tui docker)`" + `
holds only the command. ? lists every key.

--once prints the screen the browser opens on, at COLUMNS by LINES. --json prints
the command tree it opens on, each command with its help, so a program reads what
a person reads. Its help is plain text wrapped wide rather than at the screen's
width. cmd/tui-command-tree.schema.json in helpnav's repository holds the shape
--json prints. With no terminal and neither flag, helpnav tui refuses.`,
	Example: `  helpnav tui docker            a large tree, read whole before you walk it
  helpnav tui aws               too large to read up front, so read one level at a time
  helpnav tui uv                a Rust tool; the reader does not care which language
  CMD=$(helpnav tui docker)     the screen draws on stderr; the command goes to stdout
  COLUMNS=160 LINES=50 helpnav tui uv --once > frame.txt
  helpnav tui uv --json | jq '.root.children[].argv | join(" ")'`,
	Args:          cobra.ExactArgs(1),
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI(cmd.OutOrStdout(), args[0])
	},
}

func init() {
	tuiCmd.Flags().IntVar(&depthFlag, "depth", 0,
		"how many words deep to read (0 reads far enough for a hand-written tool)")
	tuiCmd.Flags().BoolVar(&onceFlag, "once", false, "print the screen the browser opens on at COLUMNS by LINES, then exit")
	tuiCmd.Flags().BoolVar(&jsonFlag, "json", false, "print the command tree the browser opens on as JSON to stdout")
	rootCmd.AddCommand(tuiCmd)
}

// runTUI opens the reader on one tool and prints whatever command it was
// left on. Walking away prints nothing, so a caller substituting the output
// gets an empty string rather than a command nobody chose.
func runTUI(out io.Writer, binary string) error {
	if onceFlag && jsonFlag {
		return goclikit.UsageError(errors.New("--once prints a frame and --json the command tree, so give one of them"))
	}
	// --once and --json read the tool too, and reading means starting it.
	if _, listed := doNotRun(binary); listed {
		return refuseToRun(binary)
	}
	// LookPath runs before the browser opens, so a mistyped name is answered on
	// the line it was typed on rather than after a screen that flashes and
	// closes. A name that resolves to nothing is the caller's mistake and exits
	// 2. A tool that is there and prints no help is still clisurface's to report.
	if _, err := exec.LookPath(binary); err != nil {
		return goclikit.UsageError(fmt.Errorf("helpnav: %q is not a command on PATH, so there is no help to read", binary))
	}
	switch {
	case jsonFlag:
		return printTree(out, binary)
	case onceFlag:
		return printFrame(out, binary)
	}
	// The screen reads keys from stdin and draws on stderr, so those two are the
	// terminal it needs. Stdout is left free to be the pipe carrying the command.
	if !term.IsTerminal(os.Stdin.Fd()) || !term.IsTerminal(os.Stderr.Fd()) {
		return goclikit.UsageError(errNoTerminal)
	}
	argv, err := tui.Run(binary, depthFlag)
	if err != nil {
		return err
	}
	if len(argv) == 0 {
		return nil
	}
	_, err = fmt.Fprintln(out, strings.Join(argv, " "))
	return err
}

// printFrame draws the screen the browser opens on. A frame bound for a file
// or a pipe carries no escapes, because what reads it there is a program or a
// diff rather than a terminal.
func printFrame(out io.Writer, binary string) error {
	frame, err := tui.Once(binary, depthFlag, envSize("COLUMNS", onceFallbackWidth), envSize("LINES", onceFallbackHeight))
	if err != nil {
		return err
	}
	if file, ok := out.(*os.File); !ok || !term.IsTerminal(file.Fd()) {
		frame = ansi.Strip(frame)
	}
	_, err = fmt.Fprintln(out, frame)
	return err
}

// envSize reads COLUMNS or LINES, the size a caller with no terminal asks for.
func envSize(name string, fallback int) int {
	if n, err := strconv.Atoi(os.Getenv(name)); err == nil && n > 0 {
		return n
	}
	return fallback
}

// printTree prints the command tree the browser opens on, as --json.
func printTree(out io.Writer, binary string) error {
	tool, err := tui.Tree(binary, depthFlag)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(out)
	// Help text is full of <placeholders>, and the encoder would otherwise write
	// each angle bracket as an HTML-safe unicode escape.
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(newCommandTree(tool, time.Now().UTC()))
}

// commandTree is `helpnav tui --json`: the tree the browser opens on.
// cmd/tui-command-tree.schema.json is its shape.
type commandTree struct {
	SchemaVersion int          `json:"schema_version"`
	Binary        string       `json:"binary"`
	Framework     string       `json:"framework"`
	ReadAt        time.Time    `json:"read_at"`
	Root          commandShown `json:"root"`
}

const commandTreeVersion = 1

// commandShown is one command with the help it printed. Every field is always
// present, so a reader tells an empty value from a missing one by its value.
type commandShown struct {
	Name     string         `json:"name"`
	Path     []string       `json:"path"`
	Argv     []string       `json:"argv"`
	Short    string         `json:"short"`
	Usage    string         `json:"usage"`
	Flags    []string       `json:"flags"`
	Help     string         `json:"help"`
	Unread   bool           `json:"unread"`
	Children []commandShown `json:"children"`
}

func newCommandTree(tool *clisurface.Tool, readAt time.Time) commandTree {
	root := tool.Root
	if root == nil {
		root = &clisurface.Node{}
	}
	return commandTree{
		SchemaVersion: commandTreeVersion,
		Binary:        tool.Binary,
		Framework:     string(tool.Framework),
		ReadAt:        readAt,
		Root:          newCommandShown(tool.Binary, root),
	}
}

func newCommandShown(binary string, node *clisurface.Node) commandShown {
	argv := append([]string{binary}, node.Path...)
	shown := commandShown{
		// The root's name is the binary, which clisurface leaves blank.
		Name:     argv[len(argv)-1],
		Path:     append([]string{}, node.Path...),
		Argv:     argv,
		Short:    node.Short,
		Usage:    node.Usage,
		Flags:    append([]string{}, node.Flags...),
		Help:     node.Body,
		Unread:   node.Unread,
		Children: make([]commandShown, 0, len(node.Children)),
	}
	for _, child := range node.Children {
		shown.Children = append(shown.Children, newCommandShown(binary, child))
	}
	return shown
}
