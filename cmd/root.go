package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/datapointchris/goclikit"
	"github.com/datapointchris/goselfupdate/autoupdate"
	"github.com/spf13/cobra"

	"github.com/datapointchris/helpnav/tui"
)

var depthFlag int

var rootCmd = &cobra.Command{
	Use:   "helpnav [tool]",
	Short: "Browse a CLI's help as a tree and leave with the command typed",
	Long: `helpnav reads an installed command-line tool's help and turns it into a tree
you can walk.

The grammar is one word: helpnav TOOL. Everything after that happens in the
browser — move through the commands, read what each one does beside the list,
and stop on the one you want. helpnav prints that command, so what you went
looking for is what you come away with.

The tool being read is not modified and nothing is installed into it. Only
--help is run, at each level, plus a framework's own completion callback where
there is one. A bare subcommand is never run, because a noun that performs a
read when invoked with no verb would perform it.

That bounds what helpnav asks for, not what a tool does when asked. Answering
--help means starting the program, so whatever it does before printing happens
too. A tool that opens a window rather than printing is named in
` + "`helpnav do-not-run`" + ` and is never started.`,
	Example: `  helpnav docker      a large tree, read one level at a time
  helpnav uv          a Rust tool; the reader does not care which language
  helpnav helpnav     this tool, read by itself`,
	Args:          cobra.MaximumNArgs(1),
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return browse(cmd, args[0])
	},
}

func Execute() {
	// os.Exit lives here alone so run() can use defer, which os.Exit skips.
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	autoConfig := autoupdate.Config{Update: updateConfig()}
	if err := goclikit.Execute(ctx, rootCmd, autoConfig); err != nil {
		if !errors.Is(err, goclikit.ErrReported) {
			fmt.Fprintln(os.Stderr, err)
		}
		// 2 says the command line was wrong rather than the run, which is the
		// only failure a caller should retry with different arguments.
		if errors.Is(err, goclikit.ErrUsage) {
			return 2
		}
		return 1
	}
	return 0
}

const groupTool = "tool"

func init() {
	rootCmd.AddGroup(&cobra.Group{ID: groupTool, Title: "Managing helpnav itself"})

	rootCmd.Flags().IntVar(&depthFlag, "depth", 0,
		"how many words deep to read (0 reads far enough for a hand-written tool)")
}

// browse opens the reader on one tool and prints whatever command it was left
// on. Walking away prints nothing, so a caller substituting the output gets an
// empty string rather than a command nobody chose.
func browse(cmd *cobra.Command, binary string) error {
	if _, listed := doNotRun(binary); listed {
		return refuseToRun(binary)
	}
	argv, err := tui.Run(binary, depthFlag)
	if err != nil {
		return err
	}
	if len(argv) == 0 {
		return nil
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), strings.Join(argv, " "))
	return err
}
