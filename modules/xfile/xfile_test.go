package xfile

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/lexer"
	"github.com/jerr3n/rox/parser"
)

// Output of `rojo sourcemap` (Rojo 7.7) for this layout:
//
//	src/shared/Signal.luau        -> ReplicatedStorage.Shared.Signal
//	src/shared/Signal.meta.json
//	src/shared/Util/init.luau     -> ReplicatedStorage.Shared.Util
//	src/shared/Util/Strings.luau  -> ReplicatedStorage.Shared.Util.Strings
//	src/server/main.server.luau   -> ServerScriptService.Server.main
const sourcemapJSON = `{"name":"demo","className":"DataModel","filePaths":["default.project.json"],"children":[{"name":"ReplicatedStorage","className":"ReplicatedStorage","children":[{"name":"Shared","className":"Folder","children":[{"name":"Signal","className":"ModuleScript","filePaths":["src/shared/Signal.luau","src/shared/Signal.meta.json"]},{"name":"Util","className":"ModuleScript","filePaths":["src/shared/Util/init.luau"],"children":[{"name":"Strings","className":"ModuleScript","filePaths":["src/shared/Util/Strings.luau"]}]}]}]},{"name":"ServerScriptService","className":"ServerScriptService","children":[{"name":"Server","className":"Folder","children":[{"name":"main","className":"Script","filePaths":["src/server/main.server.luau"]}]}]}]}`

func loadTestSourcemap(t *testing.T) *Sourcemap {
	t.Helper()
	sm, err := ParseSourcemap(strings.NewReader(sourcemapJSON), "/proj")
	if err != nil {
		t.Fatal(err)
	}
	return sm
}

func TestFile(t *testing.T) {
	sm := loadTestSourcemap(t)
	tests := map[string]string{
		"/proj/src/shared/Signal.luau":                 "ReplicatedStorage.Shared.Signal",
		"/proj/src/shared/Util/init.luau":              "ReplicatedStorage.Shared.Util",
		"/proj/src/server/../shared/Util/Strings.luau": "ReplicatedStorage.Shared.Util.Strings",
	}
	for path, want := range tests {
		inst := sm.File(path)
		if inst == nil {
			t.Errorf("File(%q) = nil, want %s", path, want)
			continue
		}
		if got := inst.FullName(); got != want {
			t.Errorf("File(%q) = %s, want %s", path, got, want)
		}
	}
	if inst := sm.File("/proj/src/shared/Signal.meta.json"); inst != nil {
		t.Errorf("meta.json should not be indexed, got %s", inst.FullName())
	}
	if n := len(sm.Files()); n != 4 {
		t.Errorf("len(Files()) = %d, want 4", n)
	}
}

func TestResolve(t *testing.T) {
	sm := loadTestSourcemap(t)
	const (
		main = "/proj/src/server/main.server.luau"
		util = "/proj/src/shared/Util/init.luau"
	)
	tests := []struct {
		from string
		code string
		want string // "" means unresolved
	}{
		{main, `require(game:GetService("ReplicatedStorage").Shared.Signal)`, "ReplicatedStorage.Shared.Signal"},
		{main, `require(game.ReplicatedStorage.Shared.Util.Strings)`, "ReplicatedStorage.Shared.Util.Strings"},
		{main, `require(game.ReplicatedStorage["Shared"].Signal)`, "ReplicatedStorage.Shared.Signal"},
		{main, `require(game.ReplicatedStorage.Shared:WaitForChild("Signal"))`, "ReplicatedStorage.Shared.Signal"},
		{main, `require(game.ReplicatedStorage.Shared:WaitForChild("Signal", 5))`, "ReplicatedStorage.Shared.Signal"},
		{main, `require(game.ReplicatedStorage.Shared:FindFirstChild("Util", false))`, "ReplicatedStorage.Shared.Util"},
		{main, `require(script.Parent.Parent.Parent.ReplicatedStorage.Shared.Signal)`, "ReplicatedStorage.Shared.Signal"},
		{main, `require((game.ReplicatedStorage).Shared.Signal)`, "ReplicatedStorage.Shared.Signal"},

		// In init.luau, `script` is the directory's instance.
		{util, `require(script.Strings)`, "ReplicatedStorage.Shared.Util.Strings"},
		{util, `require(script.Parent.Signal)`, "ReplicatedStorage.Shared.Signal"},

		{main, `require(12345)`, ""},
		{main, `require(game.ReplicatedStorage.Missing)`, ""},
		{main, `require(shared.Signal)`, ""},
		{main, `require(game.ReplicatedStorage[name])`, ""},
		{main, `require(game.ReplicatedStorage:FindFirstChild("Signal", true))`, ""},
		{main, `require(game.ReplicatedStorage.Shared:GetService("Signal"))`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			reqs := sm.Requires(parse(t, tt.code), tt.from)
			if len(reqs) != 1 {
				t.Fatalf("got %d requires, want 1", len(reqs))
			}
			got := ""
			if reqs[0].Target != nil {
				got = reqs[0].Target.FullName()
			}
			if got != tt.want {
				t.Errorf("resolved to %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDuplicateChild(t *testing.T) {
	parent := &Instance{Children: []*Instance{{Name: "A"}, {Name: "A"}, {Name: "B"}}}
	if c := parent.Child("A"); c != nil {
		t.Errorf("Child(A) with duplicates = %v, want nil", c)
	}
	if c := parent.Child("B"); c == nil {
		t.Error("Child(B) = nil")
	}
}

func parse(t *testing.T, code string) *ast.Block {
	t.Helper()
	lex, err := lexer.Lexer(code)
	if err != nil {
		t.Fatal(err)
	}
	block, err := parser.Parser(*lex)
	if err != nil {
		t.Fatal(err)
	}
	return block
}

func TestDump(t *testing.T) {
	sm := loadTestSourcemap(t)
	want := `DataModel demo
  ReplicatedStorage ReplicatedStorage
    Folder Shared
      ModuleScript Signal  src/shared/Signal.luau
      ModuleScript Util  src/shared/Util/init.luau
        ModuleScript Strings  src/shared/Util/Strings.luau
  ServerScriptService ServerScriptService
    Folder Server
      Script main  src/server/main.server.luau
`
	if got := Dump(sm); got != want {
		t.Errorf("Dump(sm) =\n%s\nwant:\n%s", got, want)
	}

	main := "/proj/src/server/main.server.luau"
	reqs := sm.Requires(parse(t, `require(game.ReplicatedStorage.Shared.Signal) require(shared.X)`), main)
	want = `Require → ReplicatedStorage.Shared.Signal  /proj/src/shared/Signal.luau
  Field .Signal
    Field .Shared
      Field .ReplicatedStorage
        Ident game
Require ✗ unresolved
  Field .X
    Ident shared
`
	if got := Dump(reqs); got != want {
		t.Errorf("Dump(reqs) =\n%s\nwant:\n%s", got, want)
	}
}

// writeFiles creates each file under a temp directory and returns it.
func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, src := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestResolvePath(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"main.luau":        "",
		"foo.luau":         "",
		"lib.lua":          "",
		"pkg/init.luau":    "",
		"pkg/a.luau":       "",
		"both.luau":        "",
		"both/init.luau":   "",
		"mod.luau":         "",
		"mod/child.luau":   "",
		"empty/readme.txt": "",
	})
	tests := []struct {
		from string
		req  string
		want string // "" means unresolved
	}{
		{"main.luau", "./foo", "foo.luau"},
		{"main.luau", "./lib", "lib.lua"},
		{"main.luau", "./pkg", "pkg/init.luau"},
		{"main.luau", "./pkg/a", "pkg/a.luau"},
		{"pkg/a.luau", "../foo", "foo.luau"},
		{"pkg/a.luau", "./init", "pkg/init.luau"},

		// In init.luau the module is the directory, so ./ means its siblings.
		{"pkg/init.luau", "./foo", "foo.luau"},
		{"pkg/init.luau", "@self/a", "pkg/a.luau"},
		{"mod.luau", "@self/child", "mod/child.luau"},

		{"main.luau", "./both", ""}, // both.luau and both/init.luau
		{"main.luau", "./missing", ""},
		{"main.luau", "./empty", ""},  // a directory without init.luau
		{"main.luau", "foo", ""},      // needs ./
		{"main.luau", "@pkg/foo", ""}, // aliases need .luaurc
	}
	for _, tt := range tests {
		t.Run(tt.from+" "+tt.req, func(t *testing.T) {
			got := ResolvePath(tt.req, filepath.Join(dir, tt.from))
			want := ""
			if tt.want != "" {
				want = filepath.Join(dir, tt.want)
			}
			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

func TestFollow(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"main.luau":     `local foo = require("./foo") local pkg = require("./pkg") require("./missing")`,
		"foo.luau":      `require("./main") return 1`, // a cycle back to main
		"pkg/init.luau": `return require("@self/a")`,
		"pkg/a.luau":    `require("../foo") return {`, // a parse error
		"unused.luau":   `return 1`,
	})
	files := (&Sourcemap{}).Follow(filepath.Join(dir, "main.luau"))

	var got []string
	for _, f := range files {
		rel, _ := filepath.Rel(dir, f.Path)
		got = append(got, rel)
	}
	want := []string{"main.luau", "foo.luau", "pkg/init.luau", "pkg/a.luau"}
	if !slices.Equal(got, want) {
		t.Fatalf("followed %v, want %v", got, want)
	}
	if n := len(files[0].Requires); n != 3 {
		t.Errorf("main.luau has %d requires, want 3", n)
	}
	if files[3].Err == nil {
		t.Error("pkg/a.luau should fail to parse")
	}
}
