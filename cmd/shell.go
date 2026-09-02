package cmd

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/datapointchris/goclikit"
	"github.com/spf13/cobra"
)

//go:embed widgets.zsh
var zshWidgets string

// widgetShells is the one list. ValidArgs and the error that names the
// alternatives both read it, so the two cannot disagree about what exists.
var widgetShells = []string{"zsh"}

var shellCmd = &cobra.Command{
	Use:     "shell",
	GroupID: groupTool,
	Short:   "The blocks your shell loads from helpnav",
}

var widgetsCmd = &cobra.Command{
	Use:   "widgets <shell>",
	Short: "Print the line-editor block that lands a browsed command on the prompt",
	Long: `Print the line-editor block that turns helpnav into a keystroke.

Bound to a key, the widget reads the tool you have already started typing,
opens the browser on it, and replaces the line with whatever you stopped on.
The flow is: type ` + "`docker`" + `, press the key, walk the tree, and land on
` + "`docker container ls`" + ` ready to run or edit.

Nothing is bound for you. Load the block where your shell is configured and
choose the key yourself, because only you know what the rest of your keymap
already uses.`,
	Example: `  helpnav shell widgets zsh                    read the block
  eval "$(helpnav shell widgets zsh)"          load it for this shell`,
	ValidArgs: widgetShells,
	Args:      cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if args[0] != "zsh" {
			return goclikit.UsageError(fmt.Errorf(
				"no widget block for %q; helpnav has one for %s",
				args[0], strings.Join(widgetShells, ", ")))
		}
		_, err := fmt.Fprint(cmd.OutOrStdout(), zshWidgets)
		return err
	},
}

func init() {
	shellCmd.AddCommand(widgetsCmd)
	rootCmd.AddCommand(shellCmd)
}
