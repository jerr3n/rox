package tables

import (
	"strings"
	"testing"

	"github.com/jerr3n/rox/lexer"
	"github.com/jerr3n/rox/parser"
	"github.com/jerr3n/rox/printer"
)

func TestMascots(t *testing.T) {
	code := `local mascots = {"gopher", "crab"}
table.insert(mascots,"seal")`
	lex, err := lexer.Lexer(code)
	if err != nil {
		t.Fatal(err)
	}
	parse, err := parser.Parser(*lex)
	if err != nil {
		t.Fatal(err)
	}
	expect := `local mascots = {"gopher", "crab"}
mascots[#mascots + 1] = "seal"`
	out := TableInsertPatch(parse)

	printed := printer.Printer(out, printer.Config{
		Indent: " ",
		Minify: false,
	})
	if !strings.Contains(printed, expect) {
		t.Fatalf("incorrect, expected <%s>, got <%s>", expect, printed)
	}
}
func TestVariable(t *testing.T) {
	code := `local numbers = {1, 2}
table.insert(numbers,3)`
	lex, err := lexer.Lexer(code)
	if err != nil {
		t.Fatal(err)
	}
	parse, err := parser.Parser(*lex)
	if err != nil {
		t.Fatal(err)
	}
	expect := `local numbers = {1, 2}
numbers[#numbers + 1] = 3`
	out := TableInsertPatch(parse)

	printed := printer.Printer(out, printer.Config{
		Indent: " ",
		Minify: false,
	})
	if !strings.Contains(printed, expect) {
		t.Fatalf("incorrect, expected <%s>, got <%s>", expect, printed)
	}
}
func TestLoop(t *testing.T) {
	code := `local numbers = {}
for i = 1, 10 do
	table.insert(numbers,i)
end`
	lex, err := lexer.Lexer(code)
	if err != nil {
		t.Fatal(err)
	}
	parse, err := parser.Parser(*lex)
	if err != nil {
		t.Fatal(err)
	}
	expect := `local numbers = {}
for i = 1, 10 do
	numbers[#numbers + 1] = i
end`
	out := TableInsertPatch(parse)

	printed := printer.Printer(out, printer.Config{
		Indent: "\t",
		Minify: false,
	})
	if !strings.Contains(printed, expect) {
		t.Fatalf("incorrect, expected <%s>, got <%s>", expect, printed)
	}
}
