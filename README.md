# helpnav

Browse a command-line tool's help as a tree, and leave with the command typed.

`--help` answers one screen at a time. Finding a subcommand three levels down
means running it four times and holding the shape in your head. helpnav reads
every level first and hands you the tree. You walk it and stop on a command.
helpnav prints that command.

Reading help is the job of
[clisurface](https://github.com/datapointchris/clisurface). helpnav is the
browser over what it reads: the cursor, the three columns, and the command
handed back on the way out.

## Install

```bash
go install github.com/datapointchris/helpnav@latest  # build from source with a Go toolchain
helpnav update                                       # later: replace the binary with the latest release
```

Each release also publishes a `.tar.gz` archive for Linux and macOS, on amd64
and arm64. One unpacked onto your `PATH` needs no Go toolchain.
`helpnav update` installs from those archives and checks each one against the
release's published checksums.

At a terminal, helpnav asks GitHub for a newer release at most once a day. When
one exists, it prints a single line on stderr naming it. It installs nothing on
its own. The check is skipped unless stdout and stderr are both terminals, so
`$(helpnav docker)` never makes it. Set `HELPNAV_NO_AUTO_UPDATE` to any value
to turn it off.

## Starting from scratch

```bash
go install github.com/datapointchris/helpnav@latest  # or unpack a release archive onto PATH
helpnav version                      # a release number; a copy built in a checkout says dev
helpnav do-not-run                   # the tools it never starts, then the file that extends the list
eval "$(helpnav shell widgets zsh)"  # put this in .zshrc; it defines helpnav-widget and hn
bindkey '^X^H' helpnav-widget        # any key you have free; helpnav binds none itself
hn helpnav                           # walk its own tree; enter leaves the command at the next prompt
```

`helpnav update` refuses a copy whose version is dev, because it has no release
to compare against. A binary from `go install` or from a release archive
reports its release number and updates normally.

`do-not-run` is worth reading before the first browse. Reading a tool means
starting it, and a listed tool is refused before anything starts. The refusal
names the file to edit.

`hn helpnav` is the proof. The screen opens on helpnav's own commands. Stop on
one, press `enter`, and that command waits at the next prompt, ready to edit.

The widget block exists for zsh alone. In any other shell, `helpnav <tool>`
prints the chosen command on stdout for the shell to take from there.

A name that is not on `PATH` exits 2 before the screen opens. A typo is
answered on the line it was typed on, not after a screen that flashes and
closes.

## `helpnav <tool>` — the help tree in three columns, ending in a command

```bash
helpnav docker         # read docker's help, then walk it
CMD=$(helpnav docker)  # the screen draws on stderr; the chosen command goes to stdout
```

The three columns are where you came from, what is here, and the help for the
row under the cursor. The footer carries the command as it assembles.

| Key | |
| --- | --- |
| `j` `k` | move through the commands |
| `l` `h` | into a command, back out |
| `g` `G` | first, last |
| `f` `b`, `d` `u` | scroll the help pane a page, half a page |
| `enter` | take this one and quit |
| `q` `esc` | leave with nothing |

The arrow keys, Home and End do what their letters do.

`enter` always takes the row under the cursor. `l` only ever descends. It
refuses a command with nothing below it. Neither key has to guess which one you
meant.

Leaving with `q` prints nothing. A caller substituting the output gets an empty
string, never a command nobody chose.

Stepping back out with `h` returns the cursor to the row you left, not to the
top of the list.

## `shell widgets zsh` — the chosen command lands on the line you were typing

```bash
eval "$(helpnav shell widgets zsh)"  # defines helpnav-widget and hn
bindkey '^X^H' helpnav-widget        # replace the line you are typing
hn docker                            # or load the next prompt, and keep this one
```

The widget reads the first word already on the line as the tool. Type
`docker`, press the key, walk the tree, and the line becomes
`docker container ls`, ready to run or edit.

Nothing is bound for you. Only you know what the rest of your keymap uses.

`hn` loads the result onto the *next* prompt instead. Use it when the line you
are on is worth keeping.

Run `helpnav shell widgets zsh` without the `eval` to read the block before you
load it.

## `do-not-run` — reading a tool means starting it, so some tools are never read

```bash
helpnav do-not-run  # every listed tool, its reason, and the file you add to
```

helpnav asks a tool for `--help` at each level. Where the framework has a
completion callback, it asks that too. A few tools print help no reader can
parse, so they are read from a fixed entry point instead, such as `git help -a`
or `tmux list-commands`. A bare subcommand is never run, because a noun that
performs a read when invoked with no verb would perform it.

That bounds what helpnav asks for, not what a tool does when asked. Answering
`--help` means starting the program. Everything it does before printing happens
too: a group callback, an update check, a lazily cloned config repo. Some tools
ignore `--help` and open a window instead.

Those tools are on the list. Add to it in
`$XDG_CONFIG_HOME/helpnav/do-not-run.txt`, which is
`~/.config/helpnav/do-not-run.txt` by default. Write one tool per line, with a
reason after the name. `#` starts a comment. The reason is printed when the tool
is refused.

A missing or unreadable file is not an error. The built-in entries still stand.

The list is written by hand. You know a desktop app is not a CLI before it costs
you twenty seconds finding out. A threshold would have to start the tool once to
learn that, which is the cost the list exists to avoid.

## A read costs the tool's startup, not helpnav's

Each command in the tree costs one start of the tool. helpnav reads sibling
commands concurrently, as many at once as the machine has CPUs. A compiled tool
starts in milliseconds, so its whole tree arrives quickly. A Python tool takes
about a fifth of a second to start, so a large one is felt. The screen says
`reading <tool>…` until the tree arrives. `q` works throughout.

A surface too large to read up front is read one level at a time. aws is read
this way: a service's operations are read the first time you step into it.

The walk stops four words past the tool's own name. kubectl, docker and gh nest
three words deep, so the bound cuts nothing from them. `--depth` raises it for a
surface that goes further.

## clisurface does the reading, so helpnav works on tools that know nothing about it

helpnav needs no cooperation from the tool, in any language. clisurface carries
a reader for each shape a help screen takes: cobra's completion callback, the
box panels Typer and Click draw, clap's `Commands:` rows, argparse's
brace-wrapped choices, and hand-rolled headings. A tool no reader reaches is
read from a recipe that names its entry point. git and tmux are read that way.

This repo holds the interaction: the cursor over the tree, the columns that
draw it, and the command handed back on exit. A help screen read wrong is
clisurface's to fix.
