package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The tool being read is started only by clisurface, which holds the rule on
// what it may be asked. A process started from helpnav's own code would be a
// path around that rule, so no file here names anything in os/exec but
// LookPath, which finds a binary on PATH without starting it.
var execAllowed = map[string]bool{"LookPath": true}

// nav and helpbody hold where you are in the tree and in one help screen. They
// stay testable only while they import no terminal and no UI framework, so
// each of these, and any package under it, is refused there.
var renderingImports = []string{
	"github.com/charmbracelet/bubbletea",
	"github.com/charmbracelet/bubbles",
	"github.com/charmbracelet/lipgloss",
	"github.com/charmbracelet/x/term",
	"golang.org/x/term",
	"github.com/spf13/cobra",
}

var stateOnlyDirs = map[string]bool{"nav": true, "helpbody": true}

func TestToolIsStartedOnlyThroughClisurface(t *testing.T) {
	for path, file := range sourceFiles(t) {
		names := localNames(file)
		ast.Inspect(file, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			switch names[pkg.Name] {
			case "os/exec":
				if !execAllowed[sel.Sel.Name] {
					t.Errorf("%s calls exec.%s; the tool being read is started only by clisurface", path, sel.Sel.Name)
				}
			case "os":
				if sel.Sel.Name == "StartProcess" {
					t.Errorf("%s calls os.StartProcess; the tool being read is started only by clisurface", path)
				}
			}
			return true
		})
	}
}

func TestStatePackagesImportNoRenderer(t *testing.T) {
	for path, file := range sourceFiles(t) {
		if !stateOnlyDirs[filepath.Dir(path)] {
			continue
		}
		for _, imp := range file.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			for _, refused := range renderingImports {
				if p == refused || strings.HasPrefix(p, refused+"/") {
					t.Errorf("%s imports %s; nav and helpbody hold state and render nothing", path, p)
				}
			}
		}
	}
}

// sourceFiles parses every non-test Go file in the module, keyed by its path
// relative to the module root.
func sourceFiles(t *testing.T) map[string]*ast.File {
	t.Helper()
	files := map[string]*ast.File{}
	fset := token.NewFileSet()
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != "." && (strings.HasPrefix(d.Name(), ".") || d.Name() == "testdata" || d.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		files[path] = file
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("found no Go source files to check")
	}
	return files
}

// localNames maps the name each import is used under in file to its path.
func localNames(file *ast.File) map[string]string {
	names := map[string]string{}
	for _, imp := range file.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		name := p[strings.LastIndex(p, "/")+1:]
		if imp.Name != nil {
			name = imp.Name.Name
		}
		names[name] = p
	}
	return names
}
