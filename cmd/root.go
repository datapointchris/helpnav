package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/datapointchris/clisurface"
	"github.com/datapointchris/goselfupdate/autoupdate"
	"github.com/datapointchris/goselfupdate/cobracmd"
	"github.com/spf13/cobra"
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
read when invoked with no verb would perform it.`,
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
	if err := cobracmd.Execute(ctx, rootCmd, autoConfig); err != nil {
		if !errors.Is(err, cobracmd.ErrReported) {
			fmt.Fprintln(os.Stderr, err)
		}
		// 2 says the command line was wrong rather than the run, which is the
		// only failure a caller should retry with different arguments.
		if errors.Is(err, cobracmd.ErrUsage) {
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

// browse reads one tool's surface and hands it to the reader.
func browse(cmd *cobra.Command, binary string) error {
	tool, err := clisurface.Extract(binary, clisurface.Options{
		WithBody: true,
		MaxDepth: depthFlag,
	})
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if _, err := fmt.Fprintf(out, "%s  (%s)\n", tool.Binary, tool.Framework); err != nil {
		return err
	}
	var walkErr error
	tool.Walk(func(n *clisurface.Node) {
		if walkErr != nil {
			return
		}
		indent := strings.Repeat("  ", len(n.Path))
		_, walkErr = fmt.Fprintf(out, "%s%s  %s\n", indent, n.Name, n.Short)
	})
	return walkErr
}
