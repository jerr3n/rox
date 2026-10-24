package parser

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/lexer"
	"github.com/jerr3n/rox/printer"
)

// Each test parses one file in test/. For checkAST, test/NAME.luau is
// compared against test/NAME.ast, which is the tree printed by ast.Dump.
// For checkError (the err_*.luau files) there is no .ast file; Parser only
// has to return an error.
//
// go test ./parser -v prints every tree, so you can see what the parser
// built. go test ./parser -update rewrites the .ast files from the parser's
// current output. Only do that once the parser is right, and read the diff.
var update = flag.Bool("update", false, "rewrite test/*.ast from the parser's output")

// parse lexes and parses in, but turns a panic or a hang into a normal test
// failure instead of crashing or freezing the whole test run.
func parse(t *testing.T, in string) (*ast.Block, error) {
	t.Helper()
	toks, err := lexer.Lexer(in, nil)
	if err != nil {
		t.Fatalf("lexer error (fix the test file, not the parser): %v", err)
	}
	type result struct {
		block *ast.Block
		err   error
	}
	done := make(chan result, 1)
	crashed := make(chan any, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				crashed <- r
			}
		}()
		block, err := Parser(*toks, nil)
		done <- result{block, err}
	}()
	select {
	case r := <-done:
		return r.block, r.err
	case r := <-crashed:
		t.Fatalf("Parser panicked: %v", r)
	case <-time.After(time.Second):
		t.Fatalf("Parser didn't finish within 1s (stuck in a loop?)")
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Golden-file checks
// ---------------------------------------------------------------------------

func readSource(t *testing.T, name string) string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("test", name+".luau"))
	if err != nil {
		t.Fatal(err)
	}
	return string(src)
}

// diffLines reports the first line where got and want differ, with a few
// lines of context, rather than dumping both trees in full.
func diffLines(got, want string) string {
	g := strings.Split(strings.TrimRight(got, "\n"), "\n")
	w := strings.Split(strings.TrimRight(want, "\n"), "\n")
	i := 0
	for i < len(g) && i < len(w) && g[i] == w[i] {
		i++
	}
	if i == len(g) && i == len(w) {
		return ""
	}
	window := func(lines []string) string {
		var b strings.Builder
		for j := max(0, i-3); j < min(len(lines), i+4); j++ {
			marker := "  "
			if j == i {
				marker = "> "
			}
			fmt.Fprintf(&b, "\t%s%4d  %s\n", marker, j+1, lines[j])
		}
		if i >= len(lines) {
			b.WriteString("\t>       (end of tree)\n")
		}
		return b.String()
	}
	return fmt.Sprintf("first difference at line %d\n got:\n%s want:\n%s", i+1, window(g), window(w))
}

// checkAST parses test/NAME.luau and compares the printed tree against
// test/NAME.ast.
func checkAST(t *testing.T, name string) {
	t.Helper()
	block, err := parse(t, readSource(t, name))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := ast.Dump(block)
	t.Logf("tree for %s.luau:\n%s", name, got)

	astPath := filepath.Join("test", name+".ast")
	if *update {
		if err := os.WriteFile(astPath, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(astPath)
	if err != nil {
		t.Fatal(err)
	}
	if diff := diffLines(got, string(want)); diff != "" {
		t.Error(diff)
	}
}

// checkError parses test/NAME.luau and only requires that Parser returns an
// error.
func checkError(t *testing.T, name string) {
	t.Helper()
	block, err := parse(t, readSource(t, name))
	if err == nil {
		t.Errorf("want an error, got none; tree:\n%s", ast.Dump(block))
		return
	}
	t.Logf("error (as expected): %v", err)
}

func TestLiterals(t *testing.T)   { checkAST(t, "01_literals") }
func TestOperators(t *testing.T)  { checkAST(t, "02_operators") }
func TestSuffixes(t *testing.T)   { checkAST(t, "03_suffixes") }
func TestTables(t *testing.T)     { checkAST(t, "04_tables") }
func TestFunctions(t *testing.T)  { checkAST(t, "05_functions") }
func TestControl(t *testing.T)    { checkAST(t, "06_control") }
func TestAssignment(t *testing.T) { checkAST(t, "07_assignment") }
func TestFunctional(t *testing.T) { checkAST(t, "08_functional") }

func TestErrUnclosedTable(t *testing.T)     { checkError(t, "err_unclosed_table") }
func TestErrMissingEnd(t *testing.T)        { checkError(t, "err_missing_end") }
func TestErrMissingThen(t *testing.T)       { checkError(t, "err_missing_then") }
func TestErrAssignToCall(t *testing.T)      { checkError(t, "err_assign_to_call") }
func TestErrAmbiguousCall(t *testing.T)     { checkError(t, "err_ambiguous_call") }
func TestErrNotAStatement(t *testing.T)     { checkError(t, "err_not_a_statement") }
func TestErrLeftoverTokens(t *testing.T)    { checkError(t, "err_leftover_tokens") }
func TestErrMissingExpr(t *testing.T)       { checkError(t, "err_missing_expr") }
func TestErrIfExprNoElse(t *testing.T)      { checkError(t, "err_if_expr_no_else") }
func TestErrStatementAfterRet(t *testing.T) { checkError(t, "err_statement_after_return") }

// TestSpans checks that every top-level statement in every test file has a
// sensible span: inside the source, not empty, and after the one before it.
func TestSpans(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("test", "[0-9]*.luau"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), ".luau")
		t.Run(name, func(t *testing.T) {
			src := readSource(t, name)
			block, err := parse(t, src)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			prevEnd := 0
			for i, s := range block.Stmts {
				start, end := int(s.Pos()), int(s.End())
				if start < prevEnd || end <= start || end > len(src) {
					t.Fatalf("statement %d (%T): bad span [%d:%d] (previous ended at %d)", i+1, s, start, end, prevEnd)
				}
				prevEnd = end
			}
		})
	}
}

// TestStatementSpanText checks that a statement's span covers exactly its
// own source text, for one example of each kind.
func TestStatementSpanText(t *testing.T) {
	tests := []string{
		"local a, b = 1, 2",
		"local function f() end",
		"function a.b:c() end",
		"x = 1",
		"x += 1",
		"print(\"hi\")",
		"obj:method(1, 2)",
		"do end",
		"while a do end",
		"repeat x() until done",
		"if a then elseif b then else end",
		"for i = 1, 10 do end",
		"for k, v in pairs(t) do end",
		"return a, b",
	}
	for _, src := range tests {
		// a trailing statement makes sure the span stops at the right place,
		// except after return, which has to be last
		in := src + "\nbreak"
		if strings.HasPrefix(src, "return") {
			in = src
		}
		block, err := parse(t, in)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", src, err)
			continue
		}
		s := block.Stmts[0]
		if got := in[s.Pos():s.End()]; got != src {
			t.Errorf("%q: span covers %q", src, got)
		}
	}
}

// checkRoundTrip parses src, prints it with the printer, parses that output,
// and requires both trees to be the same. ast.Dump leaves out spans, so the
// printer is free to change spacing and layout. Both modes are checked.
func checkRoundTrip(t *testing.T, src string) {
	t.Helper()
	block, err := parse(t, src)
	if err != nil {
		t.Fatalf("parsing the input: %v\n%s", err, src)
	}
	want := ast.Dump(block)

	for _, cfg := range []printer.Config{{Indent: "\t"}, {Minify: true}} {
		var out string
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("printer panicked (minify=%v): %v\ninput:\n%s", cfg.Minify, r, src)
				}
			}()
			out = printer.Printer(block, cfg, nil)
		}()

		block2, err := parse(t, out)
		if err != nil {
			t.Fatalf("parsing the printer's output (minify=%v): %v\ninput:\n%s\noutput:\n%s", cfg.Minify, err, src, out)
		}
		if diff := diffLines(ast.Dump(block2), want); diff != "" {
			t.Errorf("tree changed after printing (minify=%v)\ninput:\n%s\noutput:\n%s\n%s", cfg.Minify, src, out, diff)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	for _, src := range []string{
		`print("hi")`,
		`f(a, b, ...)`,
		`t.x.y()`,
		`obj:method(1, nil, true)`,
		`t[k]("s")`,

		// statements
		`local x, y = 1, 2`,
		`local z`,
		`a, b.c, d[1] = 1, 2, 3`,
		`x += 1 x ..= "s" x //= 2`,
		`do print(1) end`,
		`while x do break end`,
		`repeat x -= 1 until x < 0`,
		`if a then f() elseif b then g() else h() end`,
		`for i = 1, 10 do continue end`,
		`for i = 10, 1, -1 do end`,
		`for k, v in pairs(t) do end`,
		`local function f(a, ...) return a, ... end`,
		`function a.b:c() return end`,

		// precedence
		`x = -(-a)`,
		`x = - -a`,
		`x = a - -b`,
		`x = (a + b) * c`,
		`x = a - (b - c)`,
		`x = (a - b) - c`,
		`x = 2 ^ 3 ^ 4`,
		`x = (2 ^ 3) ^ 4`,
		`x = -a ^ b`,
		`x = (-a) ^ b`,
		`x = a .. b .. c`,
		`x = (a .. b) .. c`,
		`x = not a == b`,
		`x = not (a == b)`,
		`x = a or b and c`,
		`x = (a or b) and c`,
		`x = #t + 1`,
		`x = (if a then b else c) + 1`,
		`x = if a then b elseif c then d else e`,

		// tokens minify must keep apart
		`x = 1 .. 2`,
		`x = a .. .5`,
		`x = a .. ...`,
		`x = 0x1F and 1e3 or 1.5`,

		// strings
		`x = "a\"b\\c\nd\te\0f\127g"`,
		`x = "\0011"`,

		// tables and functions
		`x = {}`,
		`x = {1, 2, a = 3, [k] = 4}`,
		`for i = #t, 1, -1 do end`,
		`x = {1, 2, 3, 4, 5, 6}`,
		`x = {f = function(a) return a end, {1}}`,
		`f(function() return 1 end)`,

		// the ambiguous-call guard
		`local a = b; (f)()`,
		`f(); (g).x = 1`,
		`x = y; (z):m()`,
		`(f)()`,
	} {
		t.Run(src, func(t *testing.T) { checkRoundTrip(t, src) })
	}
}

// TestRoundTripFiles round-trips every parser test file.
func TestRoundTripFiles(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("test", "[0-9]*.luau"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".luau")
		t.Run(name, func(t *testing.T) { checkRoundTrip(t, readSource(t, name)) })
	}
}
