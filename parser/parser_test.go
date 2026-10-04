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
)

// Each test parses one file in test/. For checkAST, test/NAME.luau is
// compared against test/NAME.ast, which is the tree printed by printTree.
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
	toks, err := lexer.Lexer(in)
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
		block, err := Parser(*toks)
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
// Tree printer
//
// Prints one node per line, with children indented under their parent:
//
//	Local
//	  names: a, b
//	  values:
//	    Number 1
//	    Binary +
//	      Ident x
//	      Number 2
// ---------------------------------------------------------------------------

var binaryNames = map[ast.BinaryOp]string{
	ast.OpAdd: "+", ast.OpSub: "-", ast.OpMul: "*", ast.OpDiv: "/",
	ast.OpFloorDiv: "//", ast.OpMod: "%", ast.OpPow: "^", ast.OpConcat: "..",
	ast.OpEq: "==", ast.OpNe: "~=", ast.OpLt: "<", ast.OpLe: "<=",
	ast.OpGt: ">", ast.OpGe: ">=", ast.OpAnd: "and", ast.OpOr: "or",
}

var unaryNames = map[ast.UnaryOp]string{
	ast.OpNeg: "-", ast.OpNot: "not", ast.OpLen: "#",
}

var itemNames = map[ast.TableItemKind]string{
	ast.ItemPositional: "positional", ast.ItemNamed: "named", ast.ItemKeyed: "keyed",
}

type printer struct {
	b strings.Builder
}

func printTree(block *ast.Block) string {
	var p printer
	p.block(block, 0)
	return p.b.String()
}

func (p *printer) line(depth int, format string, args ...any) {
	p.b.WriteString(strings.Repeat("  ", depth))
	fmt.Fprintf(&p.b, format, args...)
	p.b.WriteByte('\n')
}

// labeled prints "label:" and then each expression under it.
func (p *printer) labeled(depth int, label string, exprs []ast.Expr) {
	if len(exprs) == 0 {
		return
	}
	p.line(depth, "%s:", label)
	for _, e := range exprs {
		p.expr(e, depth+1)
	}
}

func bindingNames(bs []*ast.Binding) string {
	names := []string{}
	for _, b := range bs {
		if b == nil || b.Name == nil {
			names = append(names, "<missing name>")
			continue
		}
		names = append(names, b.Name.Name)
	}
	return strings.Join(names, ", ")
}

func (p *printer) block(b *ast.Block, depth int) {
	if b == nil {
		p.line(depth, "<nil block>")
		return
	}
	p.line(depth, "Block")
	for _, s := range b.Stmts {
		p.stmt(s, depth+1)
	}
}

func (p *printer) funcBody(f *ast.FuncBody, depth int) {
	if f == nil {
		p.line(depth, "<nil func body>")
		return
	}
	params := bindingNames(f.Params)
	if f.Vararg {
		if params != "" {
			params += ", "
		}
		params += "..."
	}
	p.line(depth, "params: (%s)", params)
	p.block(f.Body, depth)
}

func (p *printer) stmt(s ast.Stmt, depth int) {
	switch s := s.(type) {
	case *ast.Local:
		p.line(depth, "Local")
		p.line(depth+1, "names: %s", bindingNames(s.Names))
		p.labeled(depth+1, "values", s.Values)
	case *ast.LocalFunc:
		p.line(depth, "LocalFunc %s", s.Name.Name)
		p.funcBody(s.Body, depth+1)
	case *ast.FuncDecl:
		path := []string{}
		for _, id := range s.Name.Path {
			path = append(path, id.Name)
		}
		name := strings.Join(path, ".")
		if s.Name.Method != nil {
			name += ":" + s.Name.Method.Name
		}
		p.line(depth, "FuncDecl %s", name)
		p.funcBody(s.Body, depth+1)
	case *ast.Assign:
		p.line(depth, "Assign")
		p.labeled(depth+1, "targets", s.Targets)
		p.labeled(depth+1, "values", s.Values)
	case *ast.CompoundAssign:
		p.line(depth, "CompoundAssign %s=", binaryNames[s.Op])
		p.expr(s.Target, depth+1)
		p.expr(s.Value, depth+1)
	case *ast.CallStmt:
		p.line(depth, "CallStmt")
		p.expr(s.Call, depth+1)
	case *ast.Do:
		p.line(depth, "Do")
		p.block(s.Body, depth+1)
	case *ast.While:
		p.line(depth, "While")
		p.expr(s.Cond, depth+1)
		p.block(s.Body, depth+1)
	case *ast.Repeat:
		p.line(depth, "Repeat")
		p.block(s.Body, depth+1)
		p.line(depth+1, "until:")
		p.expr(s.Cond, depth+2)
	case *ast.If:
		p.line(depth, "If")
		p.expr(s.Cond, depth+1)
		p.block(s.Then, depth+1)
		for _, ei := range s.ElseIfs {
			p.line(depth+1, "ElseIf")
			p.expr(ei.Cond, depth+2)
			p.block(ei.Body, depth+2)
		}
		if s.Else != nil {
			p.line(depth+1, "Else")
			p.block(s.Else, depth+2)
		}
	case *ast.NumericFor:
		p.line(depth, "NumericFor %s", bindingNames([]*ast.Binding{s.Var}))
		p.line(depth+1, "start:")
		p.expr(s.Start, depth+2)
		p.line(depth+1, "stop:")
		p.expr(s.Stop, depth+2)
		if s.Step != nil {
			p.line(depth+1, "step:")
			p.expr(s.Step, depth+2)
		}
		p.block(s.Body, depth+1)
	case *ast.GenericFor:
		p.line(depth, "GenericFor %s", bindingNames(s.Vars))
		p.labeled(depth+1, "in", s.Exprs)
		p.block(s.Body, depth+1)
	case *ast.Return:
		p.line(depth, "Return")
		for _, v := range s.Values {
			p.expr(v, depth+1)
		}
	case *ast.Break:
		p.line(depth, "Break")
	case *ast.Continue:
		p.line(depth, "Continue")
	case nil:
		p.line(depth, "<nil stmt>")
	default:
		p.line(depth, "<unknown stmt %T>", s)
	}
}

func (p *printer) expr(e ast.Expr, depth int) {
	switch e := e.(type) {
	case *ast.Nil:
		p.line(depth, "Nil")
	case *ast.Bool:
		p.line(depth, "Bool %v", e.Value)
	case *ast.Number:
		p.line(depth, "Number %s", e.Raw)
	case *ast.String:
		p.line(depth, "String %q", e.Value)
	case *ast.Vararg:
		p.line(depth, "Vararg")
	case *ast.Ident:
		p.line(depth, "Ident %s", e.Name)
	case *ast.Field:
		p.line(depth, "Field .%s", e.Name.Name)
		p.expr(e.X, depth+1)
	case *ast.Index:
		p.line(depth, "Index")
		p.expr(e.X, depth+1)
		p.expr(e.Key, depth+1)
	case *ast.Call:
		p.line(depth, "Call")
		p.expr(e.Fn, depth+1)
		p.labeled(depth+1, "args", e.Args)
	case *ast.MethodCall:
		p.line(depth, "MethodCall :%s", e.Method.Name)
		p.expr(e.Recv, depth+1)
		p.labeled(depth+1, "args", e.Args)
	case *ast.Unary:
		p.line(depth, "Unary %s", unaryNames[e.Op])
		p.expr(e.X, depth+1)
	case *ast.Binary:
		p.line(depth, "Binary %s", binaryNames[e.Op])
		p.expr(e.L, depth+1)
		p.expr(e.R, depth+1)
	case *ast.Paren:
		p.line(depth, "Paren")
		p.expr(e.X, depth+1)
	case *ast.Func:
		p.line(depth, "Func")
		p.funcBody(e.Body, depth+1)
	case *ast.Table:
		p.line(depth, "Table")
		for _, item := range e.Items {
			switch item.Kind {
			case ast.ItemNamed:
				p.line(depth+1, "%s %s =", itemNames[item.Kind], item.Name.Name)
			default:
				p.line(depth+1, "%s", itemNames[item.Kind])
			}
			if item.Kind == ast.ItemKeyed {
				p.expr(item.Key, depth+2)
			}
			p.expr(item.Value, depth+2)
		}
	case *ast.IfExpr:
		p.line(depth, "IfExpr")
		p.expr(e.Cond, depth+1)
		p.expr(e.Then, depth+1)
		for _, ei := range e.ElseIfs {
			p.line(depth+1, "ElseIf")
			p.expr(ei.Cond, depth+2)
			p.expr(ei.Then, depth+2)
		}
		p.line(depth+1, "Else")
		p.expr(e.Else, depth+2)
	case nil:
		p.line(depth, "<nil expr>")
	default:
		p.line(depth, "<unknown expr %T>", e)
	}
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
	got := printTree(block)
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
		t.Errorf("want an error, got none; tree:\n%s", printTree(block))
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
