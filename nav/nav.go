// Package nav is a cursor over a command tree: which command you are inside,
// which of its children is selected, and the path you took to get there.
//
// It renders nothing and knows nothing about a terminal. A view asks it what to
// draw and sends it movements; everything about how that looks belongs to the
// view. Keeping the two apart is what lets the tree logic be tested without a
// terminal attached.
package nav

import "github.com/datapointchris/clisurface"

// level is one step of the descent: a command, and which of its children the
// cursor is on. A stack of these is the whole state — the bottom is the tool
// itself and the top is where you are now.
type level struct {
	node *clisurface.Node
	sel  int
}

// Nav is the cursor. The zero value is not usable; call [New].
type Nav struct {
	binary string
	levels []level
}

// New starts a cursor at the tool's root, on its first command.
func New(tool *clisurface.Tool) *Nav {
	return &Nav{
		binary: tool.Binary,
		levels: []level{{node: tool.Root}},
	}
}

// Binary is the tool's own name, which is the first word of every command.
func (n *Nav) Binary() string { return n.binary }

// here is the top of the stack. Every method goes through it rather than
// indexing the slice, so the "top is current" rule is stated once.
func (n *Nav) here() *level { return &n.levels[len(n.levels)-1] }

// Node is the command you are inside. At the start this is the tool itself.
func (n *Nav) Node() *clisurface.Node { return n.here().node }

// Children is what the current command offers, which is what a view lists.
func (n *Nav) Children() []*clisurface.Node { return n.here().node.Children }

// Cursor is which child is selected, by index into [Nav.Children].
func (n *Nav) Cursor() int { return n.here().sel }

// Depth is how many commands deep the cursor has descended, zero at the root.
func (n *Nav) Depth() int { return len(n.levels) - 1 }

// Selected is the child under the cursor, or nil when the current command has
// none. A view showing a preview asks for this first and falls back to [Nav.Node].
func (n *Nav) Selected() *clisurface.Node {
	kids := n.Children()
	if len(kids) == 0 {
		return nil
	}
	return kids[n.here().sel]
}

// Parent is the command one level up and which of its children led here, so a
// view can draw the column to the left. The node is nil at the root.
func (n *Nav) Parent() (*clisurface.Node, int) {
	if len(n.levels) < 2 {
		return nil, 0
	}
	up := n.levels[len(n.levels)-2]
	return up.node, up.sel
}

// Next moves the cursor to the following sibling, stopping at the last.
//
// Deliberately clamped rather than wrapped: a list that jumps from the bottom
// back to the top loses the reader's place, and a command list is read as a list
// rather than scrolled as a ring.
func (n *Nav) Next() {
	here := n.here()
	if here.sel+1 < len(here.node.Children) {
		here.sel++
	}
}

// Prev moves the cursor to the previous sibling, stopping at the first.
func (n *Nav) Prev() {
	here := n.here()
	if here.sel > 0 {
		here.sel--
	}
}

// First and Last jump to the ends of the current list.
func (n *Nav) First() { n.here().sel = 0 }

func (n *Nav) Last() {
	here := n.here()
	if len(here.node.Children) > 0 {
		here.sel = len(here.node.Children) - 1
	}
}

// Enter descends into the selected child and reports whether it moved.
//
// It refuses a child with no children of its own. Descending into a leaf would
// produce a level with nothing to list and no way to tell a view that the column
// it is about to draw is empty for a reason.
func (n *Nav) Enter() bool {
	sel := n.Selected()
	if sel == nil || len(sel.Children) == 0 {
		return false
	}
	n.levels = append(n.levels, level{node: sel})
	return true
}

// Leave returns to the parent command and reports whether it moved. The cursor
// there is where it was left, so going down and back up returns you to the row
// you came from rather than to the top.
func (n *Nav) Leave() bool {
	if len(n.levels) < 2 {
		return false
	}
	n.levels = n.levels[:len(n.levels)-1]
	return true
}

// Command is the argv the cursor is pointing at: the tool, then the selected
// child's path. Inside a command with no children it is that command's own path,
// because there is nothing below it to have selected.
//
// This is what a caller reads back when the view exits, and what makes browsing
// help end with something typed rather than something read.
func (n *Nav) Command() []string {
	target := n.Selected()
	if target == nil {
		target = n.Node()
	}
	argv := make([]string, 0, len(target.Path)+1)
	argv = append(argv, n.binary)
	return append(argv, target.Path...)
}
