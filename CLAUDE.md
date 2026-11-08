# helpnav — Claude Code instructions

A Go CLI that reads another tool's help and lets you walk it. The surface it
reads comes from `clisurface`; this repo is the interaction model over it.

## Constraints that must not regress

- **The tool being read is never modified and never invoked bare.** Only
  `--help`, at each level, plus a framework's own completion callback. A tool
  no reader reaches is asked only what its `clisurface` recipe names, a fixed
  entry point such as `git help -a` or `tmux list-commands`. This is
  `clisurface`'s invariant and helpnav must not add a path around it — no
  running a subcommand to see what it does, no writing into the tool's config to
  learn its shape. A noun that performs a read when invoked with no verb would
  perform it.

- **`nav` and `helpbody` render nothing; the view holds no tree or fold logic.**
  The cursor over the command tree, and the window over one help screen with its
  folded sections, are plain packages with no terminal and no framework, which
  is the only reason they can be tested at all. A bubbletea model mixes state
  with drawing, so anything about *where you are* that leaks into the view stops
  being testable the moment it does.

- **Space and `[` `]` act on the help beside the list, not on the cursor's
  row.** Space folds the section at the top of the help, and `[` `]` move the
  help a section at a time, while the cursor stays on its command. In a TUI
  whose rows open in place, the same keys fold the row and jump between
  groups. Here the help is the only text that opens, because it always shows
  whole. `J` `K` scroll it on the same terms. A TUI that copies this layout
  copies these meanings with it; one whose rows open does not.

- **Nothing here knows about any particular portfolio.** helpnav reads whatever
  is on PATH. It holds no registry, no list of tools, and no opinion about which
  ones matter. The test is whether it makes sense to someone who has never seen
  this machine.

- **Browsing ends with a command, not with a screen.** The point is to leave
  with the thing you went looking for. Any change that makes the exit path
  lossy — no argv on the way out, no way for a shell to read it — removes the
  reason the tool exists over just running `--help`.

## Rules that live elsewhere

Layout, gofumpt, golangci-lint v2, doc comments and the module rules are
house-wide standards rather than this repo's, and are not restated here. What a
help screen owes its reader governs this tool's own `--help` as much as it
governs the screens it renders.

## Never write the breaking-change trailer in a commit message

Those two words — either number, colon or not, subject or body — cut a major
release here, and the module path carries no `/vN` suffix, so once a major
exists `go install …@latest` resolves the highest v1 instead and every installed
copy is stranded. The analyzer matches them unanchored against the raw message,
so no config prevents it and it majors even a `fix:` commit. **The ban covers a
commit that merely discusses the trailer**; name it some other way and never
quote it. Deliberate majors use `chore(release-major)`.
