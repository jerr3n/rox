package bundling

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/xfile"
	"github.com/jerr3n/rox/scope"
)

func TestResolveExample(t *testing.T) {
	dir := filepath.Join("test", "example", "before")
	files, err := Load(&xfile.Sourcemap{}, filepath.Join(dir, "main.luau"), nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	infos := Resolve(files)
	for _, f := range files {
		info := infos[f.Path]
		if info == nil {
			t.Fatalf("%s has no Info", f.Path)
		}
		// every require calls the real global, not a local named require
		for _, r := range f.Requires {
			fn, ok := r.Call.Fn.(*ast.Ident)
			if !ok {
				t.Errorf("%s: require at byte %d isn't called by name", f.Path, r.Call.Start)
				continue
			}
			if sym := info.Symbols[fn]; sym == nil || sym.Kind != scope.Global {
				t.Errorf("%s: require at byte %d doesn't resolve to the global", f.Path, r.Call.Start)
			}
		}
	}

	var consts *scope.Symbol
	for _, sym := range infos[filepath.Join(dir, "sin.luau")].Symbols {
		if sym.Name == "consts" && sym.Kind == scope.Local {
			consts = sym
		}
	}
	if consts == nil {
		t.Fatal("sin.luau: no local consts")
	}
	if len(consts.Reads) != 2 || len(consts.Writes) != 0 {
		t.Errorf("sin.luau: consts has %d reads and %d writes, want 2 and 0", len(consts.Reads), len(consts.Writes))
	}
	if infos[filepath.Join(dir, "factorial.luau")].Globals["assert"] == nil {
		t.Error("factorial.luau: assert isn't a global")
	}
}

func TestBundleExample(t *testing.T) {
	out, err := Bundle(&xfile.Sourcemap{}, filepath.Join("test", "example", "before", "main.luau"), nil)
	if err != nil {
		t.Fatalf("Bundle: %v", err)
	}
	if strings.Contains(out, "require(") {
		t.Errorf("bundle still has a require:\n%s", out)
	}
}

func TestBundleCyclic(t *testing.T) {
	dir := filepath.Join("test", "cyclic")
	_, err := Bundle(&xfile.Sourcemap{}, filepath.Join(dir, "main.luau"), nil)
	var cycle *CycleError
	if !errors.As(err, &cycle) {
		t.Fatalf("err = %v, want a *CycleError", err)
	}
	even, odd := filepath.Join(dir, "even.luau"), filepath.Join(dir, "odd.luau")
	if want := []string{even, odd, even}; !slices.Equal(cycle.Cycle, want) {
		t.Errorf("cycle = %v, want %v", cycle.Cycle, want)
	}
}

func TestOrderEntryLast(t *testing.T) {
	files, err := Load(&xfile.Sourcemap{}, filepath.Join("test", "example", "before", "main.luau"), nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ordered, err := Order(files)
	if err != nil {
		t.Fatalf("Order: %v", err)
	}
	if ordered[len(ordered)-1] != files[0] {
		t.Errorf("entry %s isn't last", files[0].Path)
	}
	// every file comes after the files it requires
	pos := map[string]int{}
	for i, f := range ordered {
		pos[f.Path] = i
	}
	for _, f := range ordered {
		for _, r := range f.Requires {
			if pos[r.Path] > pos[f.Path] {
				t.Errorf("%s comes before %s, which it requires", f.Path, r.Path)
			}
		}
	}
}
