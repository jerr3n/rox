package lexer

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Each test lexes one file in test/. For checkTokens, test/NAME.luau is
// compared against test/NAME.tokens, which lists the expected tokens one
// per line. For checkError (the err_*.luau files) there is no .tokens file;
// Lexer only has to return an error.
//
// go test ./lexer -update rewrites the .tokens files from the lexer's
// current output. Only do that once the lexer is right, and read the diff.
var update = flag.Bool("update", false, "rewrite test/*.tokens from the lexer's output")

// lex runs Lexer on in, but turns a panic or a hang into a normal test
// failure instead of crashing or freezing the whole test run.
func lex(t *testing.T, in string) ([]Token, error) {
	t.Helper()
	type result struct {
		toks []Token
		err  error
	}
	done := make(chan result, 1)
	crashed := make(chan any, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				crashed <- r
			}
		}()
		toks, err := Lexer(in)
		var list []Token
		if toks != nil {
			list = *toks
		}
		done <- result{list, err}
	}()
	select {
	case r := <-done:
		return r.toks, r.err
	case r := <-crashed:
		t.Fatalf("Lexer panicked: %v", r)
	case <-time.After(time.Second):
		t.Fatalf("Lexer didn't finish within 1s (stuck in a loop?)")
	}
	return nil, nil
}

// formatToken writes a token the way .tokens files do:
//
//	local          keywords, operators and punctuation are just their text
//	IDENT x        identifiers and numbers are the kind, then the raw text
//	NUMBER 0xFF
//	STRING "'hi'"  strings are Go-quoted so newlines and quotes stay on one line
func formatToken(tk Token) string {
	switch tk.Kind {
	case TokenIdent, TokenNumber:
		return fmt.Sprintf("%s %s", tk.Kind, tk.Text)
	case TokenString, TokenInvalid:
		return fmt.Sprintf("%s %s", tk.Kind, strconv.Quote(tk.Text))
	}
	if tk.Text == tk.Kind.String() {
		return tk.Text
	}
	// Kind and text disagree, e.g. an OpAdd whose text is "-". Show both
	// so the mismatch is visible in the diff.
	return fmt.Sprintf("%s %q", tk.Kind, tk.Text)
}

func formatTokens(toks []Token) []string {
	lines := []string{}
	for _, tk := range toks {
		if tk.Kind == TokenEOF {
			continue
		}
		lines = append(lines, formatToken(tk))
	}
	return lines
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.TrimRight(string(data), "\n")
	if text == "" {
		return []string{}
	}
	return strings.Split(text, "\n")
}

// diffLines reports the first line where got and want differ, with a few
// lines of context, rather than dumping both token lists in full.
func diffLines(got, want []string) string {
	i := 0
	for i < len(got) && i < len(want) && got[i] == want[i] {
		i++
	}
	if i == len(got) && i == len(want) {
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
			b.WriteString("\t>       (end of tokens)\n")
		}
		return b.String()
	}
	return fmt.Sprintf("first difference at token %d (got %d tokens, want %d)\n got:\n%s want:\n%s",
		i+1, len(got), len(want), window(got), window(want))
}

func readSource(t *testing.T, name string) string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("test", name+".luau"))
	if err != nil {
		t.Fatal(err)
	}
	return string(src)
}

// checkTokens lexes test/NAME.luau and compares it against test/NAME.tokens.
func checkTokens(t *testing.T, name string) {
	t.Helper()
	toks, err := lex(t, readSource(t, name))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	got := formatTokens(toks)
	tokensPath := filepath.Join("test", name+".tokens")
	if *update {
		out := strings.Join(got, "\n") + "\n"
		if err := os.WriteFile(tokensPath, []byte(out), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if diff := diffLines(got, readLines(t, tokensPath)); diff != "" {
		t.Error(diff)
	}
}

// checkError lexes test/NAME.luau and only requires that Lexer returns an error.
func checkError(t *testing.T, name string) {
	t.Helper()
	toks, err := lex(t, readSource(t, name))
	if err == nil {
		t.Errorf("want an error, got none; tokens:\n\t%s",
			strings.Join(formatTokens(toks), "\n\t"))
	}
}

func TestKeywords(t *testing.T)    { checkTokens(t, "01_keywords") }
func TestIdentifiers(t *testing.T) { checkTokens(t, "02_identifiers") }
func TestOperators(t *testing.T)   { checkTokens(t, "03_operators") }
func TestNumbers(t *testing.T)     { checkTokens(t, "04_numbers") }
func TestStrings(t *testing.T)     { checkTokens(t, "05_strings") }
func TestComments(t *testing.T)    { checkTokens(t, "06_comments") }
func TestProgram(t *testing.T)     { checkTokens(t, "07_program") }

func TestErrUnterminatedString(t *testing.T)      { checkError(t, "err_unterminated_string") }
func TestErrUnterminatedSingleQuote(t *testing.T) { checkError(t, "err_unterminated_single_quote") }
func TestErrNewlineInString(t *testing.T)         { checkError(t, "err_newline_in_string") }
func TestErrUnterminatedLongString(t *testing.T)  { checkError(t, "err_unterminated_long_string") }
func TestErrLongStringWrongLevel(t *testing.T)    { checkError(t, "err_long_string_wrong_level") }
func TestErrUnterminatedLongComment(t *testing.T) { checkError(t, "err_unterminated_long_comment") }
func TestErrUnknownCharacter(t *testing.T)        { checkError(t, "err_unknown_character") }

// TestPositions checks that every token in every test file points back at
// its own text, i.e. in[Pos:End] == Text.
func TestPositions(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("test", "*.luau"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), ".luau")
		t.Run(name, func(t *testing.T) {
			src := readSource(t, name)
			toks, _ := lex(t, src)
			for i, tk := range toks {
				if tk.Pos < 0 || tk.End < tk.Pos || int(tk.End) > len(src) {
					t.Fatalf("token %d %s: bad range [%d:%d]", i+1, formatToken(tk), tk.Pos, tk.End)
				}
				if got := src[tk.Pos:tk.End]; got != tk.Text {
					t.Fatalf("token %d %s: in[%d:%d] is %q", i+1, formatToken(tk), tk.Pos, tk.End, got)
				}
			}
		})
	}
}

func TestNewlineBefore(t *testing.T) {
	tests := []struct {
		in   string
		want []bool // NewlineBefore for each token, in order
	}{
		{"f(x)", []bool{false, false, false, false}},
		{"f\n(x)", []bool{false, true, false, false}},
		{"\nf", []bool{true}},
		{"f -- comment\n(x)", []bool{false, true, false, false}},
		{"f --[[ long\ncomment ]] (x)", []bool{false, true, false, false}},
		{"f --[[ one line ]] (x)", []bool{false, false, false, false}},
		// a newline inside a token belongs to that token, not the next one
		{"f [[a\nb]] (x)", []bool{false, false, false, false, false}},
	}
	for _, tt := range tests {
		toks, err := lex(t, tt.in)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", tt.in, err)
			continue
		}
		got := []bool{}
		for _, tk := range toks {
			if tk.Kind == TokenEOF {
				continue
			}
			got = append(got, tk.NewlineBefore)
		}
		if fmt.Sprint(got) != fmt.Sprint(tt.want) {
			t.Errorf("%q: NewlineBefore is %v, want %v", tt.in, got, tt.want)
		}
	}
}
