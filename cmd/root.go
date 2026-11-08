package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/datapointchris/goclikit"
	"github.com/datapointchris/goselfupdate/autoupdate"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "helpnav",
	Short: "Browse a CLI's help as a tree and leave with the command typed",
	Long: `helpnav reads an installed command-line tool's help and turns it into a tree
you can walk.

` + "`helpnav tui TOOL`" + ` opens the browser on one tool. Move through its commands,
read what each one does beside the list, and stop on the one you want. helpnav
prints that command, so what you went looking for is what you come away with.

The tool being read is not modified and nothing is installed into it. helpnav
asks for --help at each level, plus a framework's own completion callback where
there is one. A tool whose help no reader can parse is read from a recipe
naming a fixed entry point instead, such as ` + "`git help -a`" + ` or
` + "`tmux list-commands`" + `. A bare subcommand is never run, because a noun that
performs a read when invoked with no verb would perform it.

That bounds what helpnav asks for, not what a tool does when asked. Answering
--help means starting the program, so whatever it does before printing happens
too. A tool that opens a window rather than printing is named in
` + "`helpnav do-not-run`" + ` and is never started.`,
	Example: `  helpnav tui docker                     walk docker's commands and print the one you stop on
  helpnav tui helpnav                    this tool, read by itself
  eval "$(helpnav shell widgets zsh)"    land the command you stop on at the zsh prompt
  helpnav do-not-run                     the tools helpnav never starts`,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func Execute() {
	// os.Exit lives here alone so run() can use defer, which os.Exit skips.
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := execute(ctx, autoupdate.Config{Update: updateConfig()}); err != nil {
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

// execute takes the update check's config as an argument, so a test can run
// the shipped path with the check suppressed.
func execute(ctx context.Context, config autoupdate.Config) error {
	return goclikit.Execute(ctx, rootCmd, config)
}

const (
	groupBrowse = "browse"
	groupTool   = "tool"
)

func init() {
	rootCmd.AddGroup(
		&cobra.Group{ID: groupBrowse, Title: "Browsing a tool's help"},
		&cobra.Group{ID: groupTool, Title: "Managing helpnav itself"},
	)
}
