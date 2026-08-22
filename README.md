# helpnav

Browse a command-line tool's help as a tree, and leave with the command typed.

`--help` answers "what does this do" one screen at a time, and finding a
subcommand three levels down means running it four times and holding the shape
in your head. helpnav reads all of it up front and gives you the tree.

## What it does to the tool it reads

Nothing. It runs `--help` at each level, plus a framework's own completion
callback where there is one, and reads what comes back. It never runs a bare
subcommand, because a noun that performs a read when invoked with no verb would
perform it.

It works on tools that know nothing about it, in any language. cobra, Typer,
Click, clap and hand-rolled help screens are all read the same way, by
[clisurface](https://github.com/datapointchris/clisurface).

## Using it

```sh
helpnav docker
```

Three columns: where you came from, what is here, and the help for whatever the
cursor is on. The footer carries the command as it assembles.

| Key | |
| --- | --- |
| `j` `k` | move through the commands |
| `l` `h` | into a command, back out |
| `g` `G` | first, last |
| `enter` | take this one and quit |
| `q` | leave with nothing |

The interface draws on stderr and the chosen command goes to stdout, so it
composes:

```sh
CMD=$(helpnav docker)
```

## Making it a keystroke

This is the point. Bound to a key, helpnav reads the tool you have already
started typing and replaces the line with what you picked.

```sh
eval "$(helpnav shell widgets zsh)"
bindkey '^X^H' helpnav-widget
```

Type `docker`, press the key, walk the tree, and the line becomes
`docker container ls`, ready to run or edit. Nothing is bound for you — only
you know what the rest of your keymap uses.

The same block defines `hn`, which loads the result onto the *next* prompt
instead, for when the line you are on is worth keeping.

## Reading a deep tool

Reading costs whatever the tool costs to start, not what helpnav costs to run,
and commands are read concurrently. A cobra tool answers `--help` in about 4ms
and a Python one in about 200ms, so `docker` at 132 commands takes about three
seconds and `uv` at 60 takes under a tenth.

The walk stops four words past the tool's own name. Nothing measured reaches
that — `gh`, `kubectl` and `icb` are the deepest at three words — so `--depth`
is there for a surface that goes further, not for one anybody has hit.
