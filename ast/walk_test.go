package ast_test

import (
	"testing"

	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/lexer"
	"github.com/jerr3n/rox/parser"
)

func TestInspectFindsNestedCalls(t *testing.T) {
	code := `local a = math.random(1, 6)
local t = {roll = function() return math.random() end}
if x then
	print(if y then math.random(2) else 0)
end
local r = math.random
local s = mymath.random(1)`

	lex, err := lexer.Lexer(code, nil)
	if err != nil {
		t.Fatal(err)
	}
	block, err := parser.Parser(*lex, nil)
	if err != nil {
		t.Fatal(err)
	}

	calls, paths := 0, 0
	ast.Inspect(block, func(n ast.Node) bool {
		if _, ok := ast.CallTo(n, "math.random"); ok {
			calls++
		}
		if ast.IsPath(n, "math.random") {
			paths++
		}
		return true
	})
	// Three calls, plus the bare `local r = math.random`.
	if calls != 3 {
		t.Errorf("found %d calls to math.random, want 3", calls)
	}
	if paths != 4 {
		t.Errorf("found %d uses of math.random, want 4", paths)
	}
}

func TestInspectSkipsChildren(t *testing.T) {
	code := `local f = function() return math.random() end
local x = math.random()`

	lex, err := lexer.Lexer(code, nil)
	if err != nil {
		t.Fatal(err)
	}
	block, err := parser.Parser(*lex, nil)
	if err != nil {
		t.Fatal(err)
	}

	calls := 0
	ast.Inspect(block, func(n ast.Node) bool {
		if _, ok := n.(*ast.Func); ok {
			return false // don't look inside function bodies
		}
		if _, ok := ast.CallTo(n, "math.random"); ok {
			calls++
		}
		return true
	})
	if calls != 1 {
		t.Errorf("found %d calls to math.random, want 1", calls)
	}
}
