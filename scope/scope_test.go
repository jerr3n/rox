package scope

import (
	"sort"
	"testing"

	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/lexer"
	"github.com/jerr3n/rox/parser"
)

// resolve parses code and returns its resolve info, plus every Ident in it
// in source order, so a test can say "the third x" as ids[2].
func resolve(t *testing.T, code string) (*Info, []*ast.Ident) {
	t.Helper()
	lex, err := lexer.Lexer(code, nil)
	if err != nil {
		t.Fatal(err)
	}
	block, err := parser.Parser(*lex, nil)
	if err != nil {
		t.Fatal(err)
	}
	var ids []*ast.Ident
	ast.Inspect(block, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			ids = append(ids, id)
		}
		return true
	})
	sort.Slice(ids, func(i, j int) bool { return ids[i].Pos() < ids[j].Pos() })
	return Resolve(block), ids
}

// sameSym fails unless every ident in ids resolves to the same symbol.
func sameSym(t *testing.T, info *Info, ids ...*ast.Ident) *Symbol {
	t.Helper()
	sym := info.Symbols[ids[0]]
	if sym == nil {
		t.Fatalf("%s at %d: no symbol", ids[0].Name, ids[0].Pos())
	}
	for _, id := range ids[1:] {
		if info.Symbols[id] != sym {
			t.Errorf("%s at %d: different symbol from %s at %d", id.Name, id.Pos(), ids[0].Name, ids[0].Pos())
		}
	}
	return sym
}

func diffSym(t *testing.T, info *Info, a, b *ast.Ident) {
	t.Helper()
	if info.Symbols[a] == info.Symbols[b] {
		t.Errorf("%s at %d and %s at %d: same symbol, want different", a.Name, a.Pos(), b.Name, b.Pos())
	}
}

func TestLocalAndGlobal(t *testing.T) {
	info, ids := resolve(t, `local x = 1 print(x)`)
	// x print x
	x := sameSym(t, info, ids[0], ids[2])
	if x.Kind != Local || x.Decl != ids[0] {
		t.Errorf("x: kind %d decl %v", x.Kind, x.Decl)
	}
	if len(x.Reads) != 1 || len(x.Writes) != 0 {
		t.Errorf("x: %d reads %d writes, want 1 0", len(x.Reads), len(x.Writes))
	}
	p := info.Globals["print"]
	if p == nil || info.Symbols[ids[1]] != p || p.Kind != Global {
		t.Errorf("print: not the global symbol")
	}
}

func TestLocalSeesOuter(t *testing.T) {
	info, ids := resolve(t, `local x = x`)
	// x(new) x(global)
	diffSym(t, info, ids[0], ids[1])
	if info.Symbols[ids[1]] != info.Globals["x"] {
		t.Error("right side x: want the global")
	}
}

func TestShadowSameBlock(t *testing.T) {
	info, ids := resolve(t, `local x = 1 local x = x print(x)`)
	// x1 x2 x1 print x2
	diffSym(t, info, ids[0], ids[1])
	sameSym(t, info, ids[0], ids[2])
	sameSym(t, info, ids[1], ids[4])
}

func TestLocalFunctionRecursion(t *testing.T) {
	info, ids := resolve(t, `local function f() f() end`)
	sameSym(t, info, ids[0], ids[1])

	info, ids = resolve(t, `local f = function() f() end`)
	// f(new) f(global)
	diffSym(t, info, ids[0], ids[1])
}

func TestParamsAndCapture(t *testing.T) {
	info, ids := resolve(t, `
local total = 0
local function add(n)
	local total = total + n
	print(total)
end`)
	// total1 add n total4 total1 n print total4
	outer := sameSym(t, info, ids[0], ids[4])
	inner := sameSym(t, info, ids[3], ids[7])
	n := sameSym(t, info, ids[2], ids[5])
	if !outer.Captured {
		t.Error("outer total: want captured")
	}
	if inner.Captured || n.Captured {
		t.Error("inner total / n: want not captured")
	}
	if n.Kind != Param || n.Func == nil {
		t.Errorf("n: kind %d func %v", n.Kind, n.Func)
	}
	if outer.Func != nil {
		t.Error("outer total: want top level")
	}
}

func TestCaptureInNestedBlock(t *testing.T) {
	// a do block inside a function is still that function, so this isn't
	// a capture
	info, ids := resolve(t, `local function f() local x = 1 do print(x) end end`)
	// f x print x
	x := sameSym(t, info, ids[1], ids[3])
	if x.Captured {
		t.Error("x: want not captured")
	}
}

func TestWrites(t *testing.T) {
	info, ids := resolve(t, `local x = 1 x = 2 x += 1 t.k = x`)
	// x x x t k x
	x := sameSym(t, info, ids[0], ids[1], ids[2], ids[5])
	if len(x.Writes) != 2 || len(x.Reads) != 2 {
		t.Errorf("x: %d writes %d reads, want 2 2", len(x.Writes), len(x.Reads))
	}
	tg := info.Globals["t"]
	if tg == nil || len(tg.Reads) != 1 || len(tg.Writes) != 0 {
		t.Error("t: want one read, no writes")
	}
	if info.Symbols[ids[4]] != nil {
		t.Error("k in t.k: want no symbol")
	}
}

func TestRepeatSeesBody(t *testing.T) {
	info, ids := resolve(t, `repeat local done = true until done`)
	sameSym(t, info, ids[0], ids[1])
}

func TestForRanges(t *testing.T) {
	info, ids := resolve(t, `for i = i, 2 do print(i) end`)
	// i(loop) i(global) print i(loop)
	sameSym(t, info, ids[0], ids[3])
	diffSym(t, info, ids[0], ids[1])
	if info.Symbols[ids[0]].Kind != ForVar {
		t.Error("i: want ForVar")
	}

	info, ids = resolve(t, `for k, v in pairs(k) do print(k, v) end`)
	// k v pairs k(global) print k v
	sameSym(t, info, ids[0], ids[5])
	sameSym(t, info, ids[1], ids[6])
	diffSym(t, info, ids[0], ids[3])
}

func TestFuncDecl(t *testing.T) {
	info, ids := resolve(t, `function a.b:c() return self end`)
	// a b c self
	a := info.Globals["a"]
	if a == nil || len(a.Reads) != 1 || len(a.Writes) != 0 {
		t.Error("a: want one read")
	}
	if info.Symbols[ids[1]] != nil || info.Symbols[ids[2]] != nil {
		t.Error("b, c: want no symbol")
	}
	if s := info.Symbols[ids[3]]; s == nil || s.Kind != Self {
		t.Error("self: want the implicit self")
	}

	info, _ = resolve(t, `function f() end`)
	if f := info.Globals["f"]; f == nil || len(f.Writes) != 1 {
		t.Error("f: want one write")
	}
}

func TestNotVariables(t *testing.T) {
	info, ids := resolve(t, `local t = {name = 1, [k] = 2} print(t.name, t:m())`)
	// t name k print t name t m
	for _, i := range []int{1, 5, 7} {
		if info.Symbols[ids[i]] != nil {
			t.Errorf("%s at %d: want no symbol", ids[i].Name, ids[i].Pos())
		}
	}
	if info.Symbols[ids[2]] != info.Globals["k"] {
		t.Error("[k]: want the global k")
	}
	sameSym(t, info, ids[0], ids[4], ids[6])
}
