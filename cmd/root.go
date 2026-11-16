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
	"github.com/datapointchris/gotermstyle/cobrahelp"
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
	Example: `  helpnav tui docker                     print the docker command you stop on
  helpnav tui helpnav                    this tool, read by itself
  eval "$(helpnav shell widgets zsh)"    land the command at the zsh prompt
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
	cobrahelp.Install(rootCmd)
	refuseUnknownHelpTopics(rootCmd)
	return goclikit.Execute(ctx, rootCmd, config)
}

// refuseUnknownHelpTopics makes `helpnav help WORD` refuse a word that names no
// command, the way `helpnav WORD` refuses it, with the near commands and exit 2.
//
// Cobra's own help command prints "Unknown help topic" and the root's usage for
// such a word, then returns nil, so a mistyped topic exits 0 and offers nothing
// near it. Its usage, description and completion are kept, and only what it does
// with a topic is replaced. A word left over under a command with no subcommands
// is that command's argument rather than a topic, so its help is printed.
func refuseUnknownHelpTopics(root *cobra.Command) {
	root.InitDefaultHelpCmd()
	for _, help := range root.Commands() {
		if help.Name() != "help" {
			continue
		}
		help.Run = nil
		help.RunE = func(cmd *cobra.Command, args []string) error {
			target, rest, err := cmd.Root().Find(args)
			if target == nil {
				target = cmd.Root()
			}
			if len(rest) > 0 && (err != nil || target.HasSubCommands()) {
				return goclikit.UnknownCommand(target, rest[0])
			}
			if err != nil {
				return goclikit.UsageError(err)
			}
			target.InitDefaultHelpFlag()
			target.InitDefaultVersionFlag()
			return target.Help()
		}
	}
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
