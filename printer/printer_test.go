package printer

import (
	"testing"

	"github.com/jerr3n/rox/ast"
)

// The real printer tests are round trips, in parser/parser_test.go: they need
// the parser, and the parser's tests already have a tree printer to compare with.

func TestEmpty(t *testing.T) {
	cfg := Config{Indent: "\t"}
	if got := Printer(&ast.Block{}, cfg, nil); got != "" {
		t.Errorf("empty block: got %q, want \"\"", got)
	}
	if got := Printer(nil, cfg, nil); got != "" {
		t.Errorf("nil block: got %q, want \"\"", got)
	}
}
