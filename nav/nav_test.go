package nav

import (
	"strings"
	"testing"

	"github.com/datapointchris/clisurface"
)

// demo is a tool with one group that nests, one group that does not, and a
// bare leaf at the root, which is every shape a cursor has to handle.
//
//	demo
//	├── projects
//	│   ├── list
//	│   └── items
//	│       ├── add
//	│       └── show
//	├── books
//	│   └── list
//	└── version
func demo() *clisurface.Tool {
	leaf := func(path ...string) *clisurface.Node {
		return &clisurface.Node{Path: path, Name: path[len(path)-1]}
	}
	group := func(name string, kids ...*clisurface.Node) *clisurface.Node {
		return &clisurface.Node{Path: []string{name}, Name: name, Children: kids}
	}
	items := &clisurface.Node{
		Path:     []string{"projects", "items"},
		Name:     "items",
		Children: []*clisurface.Node{leaf("projects", "items", "add"), leaf("projects", "items", "show")},
	}
	return &clisurface.Tool{
		Binary:    "demo",
		Framework: clisurface.FrameworkCobra,
		Root: &clisurface.Node{
			Name: "demo",
			Children: []*clisurface.Node{
				{Path: []string{"projects"}, Name: "projects", Children: []*clisurface.Node{leaf("projects", "list"), items}},
				group("books", leaf("books", "list")),
				leaf("version"),
			},
		},
	}
}

func cmd(n *Nav) string { return strings.Join(n.Command(), " ") }

func TestStartsAtTheRootOnTheFirstCommand(t *testing.T) {
	n := New(demo())
	if n.Depth() != 0 {
		t.Errorf("depth = %d, want 0", n.Depth())
	}
	if got := n.Selected().Name; got != "projects" {
		t.Errorf("selected = %q, want projects", got)
	}
	if got := cmd(n); got != "demo projects" {
		t.Errorf("command = %q, want \"demo projects\"", got)
	}
	if node, _ := n.Parent(); node != nil {
		t.Error("root reported a parent")
	}
}

func TestMovementClampsRatherThanWraps(t *testing.T) {
	n := New(demo())
	n.Prev()
	if got := n.Cursor(); got != 0 {
		t.Errorf("cursor = %d after Prev at the top, want 0", got)
	}
	for range 10 {
		n.Next()
	}
	if got := n.Selected().Name; got != "version" {
		t.Errorf("selected = %q after running off the bottom, want version", got)
	}
	n.First()
	if got := n.Selected().Name; got != "projects" {
		t.Errorf("First selected %q, want projects", got)
	}
	n.Last()
	if got := n.Selected().Name; got != "version" {
		t.Errorf("Last selected %q, want version", got)
	}
}

func TestEnterDescendsIntoAGroupAndRefusesALeaf(t *testing.T) {
	n := New(demo())
	if !n.Enter() {
		t.Fatal("Enter refused projects, which has children")
	}
	if n.Depth() != 1 || n.Node().Name != "projects" {
		t.Fatalf("inside %q at depth %d, want projects at 1", n.Node().Name, n.Depth())
	}
	if got := cmd(n); got != "demo projects list" {
		t.Errorf("command = %q, want \"demo projects list\"", got)
	}

	// `list` is a leaf, so there is nothing to descend into.
	if n.Enter() {
		t.Error("Enter descended into a leaf")
	}
	if n.Depth() != 1 {
		t.Errorf("depth = %d after refusing, want 1", n.Depth())
	}

	n.Next() // items, which nests
	if !n.Enter() {
		t.Fatal("Enter refused items, which has children")
	}
	if got := cmd(n); got != "demo projects items add" {
		t.Errorf("command = %q, want \"demo projects items add\"", got)
	}
}

func TestLeaveRestoresTheRowYouCameFrom(t *testing.T) {
	n := New(demo())
	n.Next() // books
	if !n.Enter() {
		t.Fatal("Enter refused books")
	}
	if !n.Leave() {
		t.Fatal("Leave refused at depth 1")
	}
	if got := n.Selected().Name; got != "books" {
		t.Errorf("selected = %q after coming back up, want books", got)
	}
	if n.Leave() {
		t.Error("Leave moved above the root")
	}
}

func TestParentIsTheColumnToTheLeft(t *testing.T) {
	n := New(demo())
	n.Next() // books
	n.Enter()
	node, sel := n.Parent()
	if node == nil || node.Name != "demo" {
		t.Fatalf("parent = %v, want the root", node)
	}
	if sel != 1 {
		t.Errorf("parent cursor = %d, want 1 (books)", sel)
	}
}

// Inside a command with no children there is nothing selected, so the command
// is the one you are in rather than one below it.
func TestCommandInsideAChildlessNodeIsThatNode(t *testing.T) {
	tool := &clisurface.Tool{
		Binary: "demo",
		Root:   &clisurface.Node{Name: "demo"},
	}
	n := New(tool)
	if n.Selected() != nil {
		t.Error("a tool with no commands reported a selection")
	}
	if got := cmd(n); got != "demo" {
		t.Errorf("command = %q, want \"demo\"", got)
	}
	if n.Enter() || n.Leave() {
		t.Error("a tool with no commands moved")
	}
}

// A surface too large to read whole names a command without reading it, so
// descending has to fetch its children first. Entering before they arrive would
// land on a level with nothing to draw.
func TestAnUnreadCommandIsPendingUntilFilled(t *testing.T) {
	tool := &clisurface.Tool{
		Binary: "aws",
		Root: &clisurface.Node{Name: "aws", Children: []*clisurface.Node{
			{Name: "s3", Path: []string{"s3"}, Unread: true},
		}},
	}
	n := New(tool)

	if !n.Pending() {
		t.Fatal("s3 was named but not read; it must report as pending")
	}
	if n.Enter() {
		t.Error("entered a command with no children read")
	}

	filled := []*clisurface.Node{
		{Name: "ls", Path: []string{"s3", "ls"}},
		{Name: "cp", Path: []string{"s3", "cp"}},
	}
	if !n.Fill(filled) {
		t.Fatal("Fill reported nothing was waiting")
	}
	if n.Pending() {
		t.Error("still pending after being filled")
	}
	if !n.Enter() {
		t.Fatal("could not enter after filling")
	}
	if got := len(n.Children()); got != 2 {
		t.Errorf("inside s3 there are %d commands, want 2", got)
	}
	if got := strings.Join(n.Command(), " "); got != "aws s3 ls" {
		t.Errorf("command = %q, want aws s3 ls", got)
	}
}

// Filling a command that was already read would replace what a walk found with
// whatever a second read returned.
func TestFillingAnAlreadyReadCommandDoesNothing(t *testing.T) {
	tool := &clisurface.Tool{
		Binary: "demo",
		Root: &clisurface.Node{Name: "demo", Children: []*clisurface.Node{
			{Name: "one", Path: []string{"one"}, Children: []*clisurface.Node{
				{Name: "kept", Path: []string{"one", "kept"}},
			}},
		}},
	}
	n := New(tool)

	if n.Pending() {
		t.Error("a command that was read reported as pending")
	}
	if n.Fill([]*clisurface.Node{{Name: "replaced"}}) {
		t.Error("Fill claimed a read command was waiting")
	}
	n.Enter()
	if got := n.Children()[0].Name; got != "kept" {
		t.Errorf("child = %q, want kept — a second read overwrote the first", got)
	}
}
