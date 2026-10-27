package inlining

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/lexer"
	"github.com/jerr3n/rox/parser"
	"github.com/jerr3n/rox/printer"
	"github.com/jerr3n/rox/scope"
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

func TestCandidates(t *testing.T) {
	cases := []struct {
		code    string
		ok      bool
		shape   returnShape
		returns int
	}{
		{"local function f(x) return x end", true, returnsTrailing, 1},
		{"local function f() print(1) end", true, returnsNone, 0},
		{"local function f(n) if n then return 1 end return 2 end", true, returnsEarly, 1},
		{"local function f() for i = 1, 3 do return i end end", false, 0, 0},
		{"local function f() return f() end", false, 0, 0},
		{"local function f() end f = print", false, 0, 0},
		{"local function f(...) end", false, 0, 0},
		{"local function f() g(function() return 1 end) end", true, returnsNone, 0},
		{"local f = function(x) return x end", true, returnsTrailing, 1},
		{"local f = function() f() end", true, returnsNone, 0},
		{"local function f() return g() end", true, returnsTrailing, unknownCount},
		{"local function f() return (g()) end", true, returnsTrailing, 1},
		{"local function f() return 1, 2 end", true, returnsTrailing, 2},
		{"local x\nlocal function f() end", true, returnsNone, 0},
	}
	for _, tc := range cases {
		b := parse(t, tc.code)
		var got *candidate
		for sym, c := range findCandidates(scope.NewIndex(b), b) {
			if sym.Name == "f" {
				got = c
			}
		}
		if (got != nil) != tc.ok {
			t.Errorf("%q: candidate = %v, want %v", tc.code, got != nil, tc.ok)
			continue
		}
		if got != nil && (got.Shape != tc.shape || got.Returns != tc.returns) {
			t.Errorf("%q: shape %d returns %d, want %d %d", tc.code, got.Shape, got.Returns, tc.shape, tc.returns)
		}
	}
}

func TestInlineExample(t *testing.T) {
	src, err := os.ReadFile("example/processing_monofile/main.luau")
	if err != nil {
		t.Fatal(err)
	}
	b := Inline(parse(t, string(src)))
	inlined := map[string]bool{"floor": true, "sum": true, "pow": true, "factorial": true, "sin": true}
	for _, st := range b.Stmts {
		if _, ok := st.(*ast.LocalFunc); ok {
			continue
		}
		ast.Inspect(st, func(n ast.Node) bool {
			if call, ok := n.(*ast.Call); ok {
				if id, ok := call.Fn.(*ast.Ident); ok && inlined[id.Name] {
					t.Errorf("call to %s left in\n%s", id.Name, render(b))
				}
			}
			return true
		})
	}
}

func TestInlineKeepsBehaviour(t *testing.T) {
	luau, err := exec.LookPath("luau")
	if err != nil {
		t.Skip("luau not on PATH")
	}
	files, _ := filepath.Glob("test/*.luau")
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			src, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			out := filepath.Join(t.TempDir(), "inlined.luau")
			if err := os.WriteFile(out, []byte(render(Inline(parse(t, string(src))))), 0o644); err != nil {
				t.Fatal(err)
			}
			want, err := exec.Command(luau, file).CombinedOutput()
			if err != nil {
				t.Fatalf("original: %v\n%s", err, want)
			}
			got, err := exec.Command(luau, out).CombinedOutput()
			if err != nil {
				t.Fatalf("inlined: %v\n%s", err, got)
			}
			if string(got) != string(want) {
				t.Errorf("output changed\nwant:\n%s\ngot:\n%s", want, got)
			}
		})
	}
}
