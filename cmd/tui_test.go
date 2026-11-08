package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/datapointchris/goclikit"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// demoHelp answers `demo --help` and each command's, in the headings clap
// prints, so clisurface walks it as it walks uv.
const demoHelp = `#!/bin/sh
case "$*" in
"--help") printf 'A demo tool\n\nUsage: demo [OPTIONS] <COMMAND>\n\nCommands:\n  run     Run a thing\n  remote  Manage remotes\n\nOptions:\n  -h, --help  Print help\n' ;;
"run --help") printf 'Run a thing\n\nUsage: demo run [OPTIONS] <NAME>\n\nOptions:\n  --fast  Run it <fast>\n' ;;
"remote --help") printf 'Manage remotes\n\nUsage: demo remote <COMMAND>\n\nCommands:\n  add  Add a remote\n\nOptions:\n  -h, --help  Print help\n' ;;
"remote add --help") printf 'Add a remote\n\nUsage: demo remote add <NAME>\n' ;;
esac
`

// demoOnPath puts demo alone on PATH, with no do-not-run list but the built-in.
func demoOnPath(t *testing.T) {
	t.Helper()
	bin := t.TempDir()
	t.Setenv("PATH", bin)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := os.WriteFile(filepath.Join(bin, "demo"), []byte(demoHelp), 0o755); err != nil {
		t.Fatal(err)
	}
}

// helpnav tui demo, run where stdin is no terminal and with neither --once nor
// --json, exits 2 rather than waiting on a keyboard nobody holds.
func TestTuiWithNoTerminalAndNeitherFlagRefuses(t *testing.T) {
	demoOnPath(t)
	stdin := os.Stdin
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = devNull
	t.Cleanup(func() { os.Stdin = stdin; _ = devNull.Close() })

	err = runLine(t, "tui", "demo")
	if !errors.Is(err, goclikit.ErrUsage) || !errors.Is(err, errNoTerminal) {
		t.Errorf("answered %v, want the no-terminal usage error", err)
	}
}

// What helpnav tui --depth 1 demo --json writes holds to the schema its --help
// points a reader at, and says the read stopped at demo remote rather than
// passing it off as a leaf. The schema leaves room for an added field, so the
// test closes both objects: a field the code adds without the schema fails here.
func TestTuiJSONAtADepthBoundHoldsToItsSchemaAndSaysWhereItStopped(t *testing.T) {
	demoOnPath(t)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	t.Cleanup(func() { rootCmd.SetOut(nil) })

	if err := runLine(t, "tui", "--depth", "1", "demo", "--json"); err != nil {
		t.Fatal(err)
	}
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatalf("%v in %s", err, out.String())
	}
	if err := closedSchema(t).Validate(instance); err != nil {
		t.Error(err)
	}
	var tree commandTree
	if err := json.Unmarshal(out.Bytes(), &tree); err != nil {
		t.Fatal(err)
	}
	remote := child(&tree.Root, "remote")
	if remote == nil || !tree.Root.ChildrenComplete || remote.ChildrenComplete {
		t.Errorf("want demo's children complete and demo remote's not:\n%s", out.String())
	}
}

func child(command *commandShown, name string) *commandShown {
	if command == nil {
		return nil
	}
	for i := range command.Children {
		if command.Children[i].Name == name {
			return &command.Children[i]
		}
	}
	return nil
}

func closedSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	document, err := os.ReadFile("tui-command-tree.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := jsonschema.UnmarshalJSON(bytes.NewReader(document))
	if err != nil {
		t.Fatal(err)
	}
	root, _ := decoded.(map[string]any)
	root["additionalProperties"] = false
	defs, _ := root["$defs"].(map[string]any)
	command, _ := defs["command"].(map[string]any)
	command["additionalProperties"] = false

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("tui-command-tree.schema.json", decoded); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile("tui-command-tree.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	return schema
}
