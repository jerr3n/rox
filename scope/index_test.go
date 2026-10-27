package scope

import (
	"testing"

	"github.com/jerr3n/rox/lexer"
	"github.com/jerr3n/rox/parser"
)

func index(t *testing.T, code string) *Index {
	t.Helper()
	lex, err := lexer.Lexer(code, nil)
	if err != nil {
		t.Fatal(err)
	}
	block, err := parser.Parser(*lex, nil)
	if err != nil {
		t.Fatal(err)
	}
	return NewIndex(block)
}

func TestFreshName(t *testing.T) {
	ix := index(t, "local x = 1\nlocal x_2 = x\nprint(y)")
	// x and x_2 are locals, y and print are globals
	for _, tc := range []struct{ base, want string }{
		{"z", "z"},
		{"z", "z_2"}, // the first z was taken
		{"x", "x_3"},
		{"y", "y_2"},
	} {
		if got := ix.FreshName(tc.base); got != tc.want {
			t.Errorf("FreshName(%q) = %q, want %q", tc.base, got, tc.want)
		}
	}
}

func TestIsFree(t *testing.T) {
	ix := index(t, "local x = 1\ndo local x = 2 end")
	var xs []*Symbol
	for id, sym := range ix.Info.Symbols {
		if sym.Decl == id && sym.Name == "x" {
			xs = append(xs, sym)
		}
	}
	if len(xs) != 2 {
		t.Fatalf("got %d x symbols, want 2", len(xs))
	}
	if ix.IsFree("x", nil) {
		t.Error("x is free with no symbols allowed")
	}
	if ix.IsFree("x", map[*Symbol]bool{xs[0]: true}) {
		t.Error("x is free with only one of its two symbols allowed")
	}
	if !ix.IsFree("x", map[*Symbol]bool{xs[0]: true, xs[1]: true}) {
		t.Error("x isn't free with both of its symbols allowed")
	}
	ix.Take("w")
	if ix.IsFree("w", nil) {
		t.Error("w is free after Take")
	}
}

func TestRename(t *testing.T) {
	info, ids := resolve(t, "local x = 1\nx = x + 1\nprint(x)")
	info.Symbols[ids[0]].Rename("y")
	for _, id := range ids {
		if sym := info.Symbols[id]; sym != nil && sym.Kind == Local && id.Name != "y" {
			t.Errorf("ident at %d is %q, want y", id.Start, id.Name)
		}
	}
	if info.Globals["print"].Name != "print" {
		t.Error("Rename touched a global")
	}
}
