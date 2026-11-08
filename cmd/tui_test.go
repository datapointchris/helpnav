package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/datapointchris/clisurface"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// helpnav tui --json holds to the schema its --help points a reader at, for a
// tree with a leaf, a nested command and one named but not read.
func TestTheCommandTreeHoldsToItsSchema(t *testing.T) {
	document, err := os.ReadFile("tui-command-tree.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := jsonschema.UnmarshalJSON(bytes.NewReader(document))
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("tui-command-tree.schema.json", decoded); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile("tui-command-tree.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	tool := &clisurface.Tool{
		Binary:    "demo",
		Framework: clisurface.FrameworkCobra,
		Root: &clisurface.Node{
			Usage: "demo [command]",
			Flags: []string{"--help"},
			Body:  "Usage: demo [command]\n",
			Children: []*clisurface.Node{
				{Path: []string{"run"}, Name: "run", Short: "run it", Body: "Usage: demo run\n"},
				{Path: []string{"remote"}, Name: "remote", Children: []*clisurface.Node{
					{Path: []string{"remote", "add"}, Name: "add"},
				}},
				{Path: []string{"cloud"}, Name: "cloud", Unread: true},
			},
		},
	}
	raw, err := json.Marshal(newCommandTree(tool, time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(instance); err != nil {
		t.Error(err)
	}
}
