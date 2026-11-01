package simplify

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/inlining"
	"github.com/jerr3n/rox/lexer"
	"github.com/jerr3n/rox/parser"
	"github.com/jerr3n/rox/printer"
)

func parse(t *testing.T, code string) *ast.Block {
	t.Helper()
	lex, err := lexer.Lexer(code, nil)
	if err != nil {
		t.Fatal(err)
	}
	block, err := parser.Parser(*lex, nil)
	if err != nil {
		t.Fatal(err)
	}
	return block
}

func render(b *ast.Block) string {
	return printer.Printer(b, printer.Config{Indent: "\t"}, nil)
}

// Both sides go through parse+render so formatting differences don't matter.
func TestRemoveDead(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "used local function is kept",
			in:   "local function f() end\nf()",
			want: "local function f() end\nf()",
		},
		{
			name: "unused local function is removed",
			in:   "local function f() end\nprint(1)",
			want: "print(1)",
		},
		{
			name: "function only used by a dead function",
			in:   "local function f() end\nlocal function g() f() end\nprint(1)",
			want: "print(1)",
		},
		{
			name: "unused recursive function",
			in:   "local function f(n) return f(n - 1) end\nprint(1)",
			want: "print(1)",
		},
		{
			name: "pure values are removed",
			in:   "local a = 1\nlocal b = function() end\nlocal c = {1, x = 2, [\"k\"] = -1 / 0}\nlocal d = not a and b\nlocal e = ...\nprint(1)",
			want: "print(1)",
		},
		{
			name: "call value is kept as a statement",
			in:   "local x = g(1)\nlocal y, z = h(), o:m()",
			want: "g(1)\nh()\no:m()",
		},
		{
			name: "impure value keeps the local",
			in:   "local x = a + b\nlocal y = t.x\nlocal z = g\nlocal w = {[k] = 1}",
			want: "local x = a + b\nlocal y = t.x\nlocal z = g\nlocal w = {[k] = 1}",
		},
		{
			name: "write-only local is removed with its writes",
			in:   "local x\nx = 1\nif c then x = function() end end\nprint(1)",
			want: "if c then end\nprint(1)",
		},
		{
			name: "write-only local with a call write",
			in:   "local x = 1\nx = g()",
			want: "g()",
		},
		{
			name: "write-only local with an impure write is kept",
			in:   "local x\nx = a.b",
			want: "local x\nx = a.b",
		},
		{
			name: "shadowed write-only local",
			in:   "local y = 5\ndo\n\tlocal y\n\ty = 3\nend\nprint(y)",
			want: "local y = 5\ndo\nend\nprint(y)",
		},
		{
			name: "partly dead local is kept",
			in:   "local a, b = 1, 2\nprint(a)",
			want: "local a, b = 1, 2\nprint(a)",
		},
		{
			name: "partly dead assignment drops pure pairs",
			in:   "local a = 1\nlocal b = 2\na, b = 3, 4\nprint(a)\nb = 5",
			want: "local a = 1\na = 3\nprint(a)",
		},
		{
			name: "partly dead assignment with an impure dead value",
			in:   "local a = 1\nlocal b = 2\na, b = 3, t.x\nprint(a)",
			want: "local a = 1\nlocal b = 2\na, b = 3, t.x\nprint(a)",
		},
		{
			name: "function statement on a dead local",
			in:   "local f\nfunction f() end\nprint(1)",
			want: "print(1)",
		},
		{
			name: "params, loop vars and globals are untouched",
			in:   "function g(a) end\nfor i = 1, 2 do end\nfor k, v in pairs(t) do end\nx = 1",
			want: "function g(a) end\nfor i = 1, 2 do end\nfor k, v in pairs(t) do end\nx = 1",
		},
		{
			name: "dead local inside a nested function",
			in:   "local function f()\n\tlocal unused = 1\n\treturn 2\nend\nprint(f())",
			want: "local function f()\n\treturn 2\nend\nprint(f())",
		},
		{
			name: "compound assignment counts as a read",
			in:   "local r = 0\nr += 1",
			want: "local r = 0\nr += 1",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := render(RemoveDead(parse(t, c.in)))
			want := render(parse(t, c.want))
			if got != want {
				t.Errorf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestRemoveDeadExample(t *testing.T) {
	src, err := os.ReadFile("../inlining/example/processing_monofile/main.luau")
	if err != nil {
		t.Fatal(err)
	}
	b := RemoveDead(inlining.Inline(parse(t, string(src))))
	ast.Inspect(b, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.LocalFunc:
			t.Errorf("local function %s left in\n%s", n.Name.Name, render(b))
		case *ast.Func:
			t.Errorf("function expression left in\n%s", render(b))
		}
		return true
	})
}

func TestRemoveDeadKeepsBehaviour(t *testing.T) {
	luau, err := exec.LookPath("luau")
	if err != nil {
		t.Skip("luau not on PATH")
	}
	files, _ := filepath.Glob("../inlining/test/*.luau")
	more, _ := filepath.Glob("test/*.luau")
	for _, file := range append(files, more...) {
		t.Run(filepath.Base(file), func(t *testing.T) {
			src, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			out := filepath.Join(t.TempDir(), "simplified.luau")
			code := render(RemoveDead(inlining.Inline(parse(t, string(src)))))
			if err := os.WriteFile(out, []byte(code), 0o644); err != nil {
				t.Fatal(err)
			}
			want, err := exec.Command(luau, file).CombinedOutput()
			if err != nil {
				t.Fatalf("original: %v\n%s", err, want)
			}
			got, err := exec.Command(luau, out).CombinedOutput()
			if err != nil {
				t.Fatalf("simplified: %v\n%s\n%s", err, got, code)
			}
			if string(got) != string(want) {
				t.Errorf("output changed\nwant:\n%s\ngot:\n%s\ncode:\n%s", want, got, code)
			}
		})
	}
}
