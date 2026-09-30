package lexer

import (
	"slices"
	"testing"
	"time"
)

// tok is shorthand for building an expected token.
func tok(kind TokenKind, text string) Token {
	return Token{Kind: kind, Text: text}
}

// lex runs Lexer on in, but turns a panic or a hang into a normal test
// failure instead of crashing or freezing the whole test run.
func lex(t *testing.T, in string) []Token {
	t.Helper()
	done := make(chan []Token, 1)
	crashed := make(chan any, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				crashed <- r
			}
		}()
		done <- Lexer(in)
	}()
	select {
	case toks := <-done:
		return toks
	case r := <-crashed:
		t.Fatalf("Lexer(%q) panicked: %v", in, r)
	case <-time.After(time.Second):
		t.Fatalf("Lexer(%q) didn't finish within 1s (stuck in a loop?)", in)
	}
	return nil
}

// withoutEOF drops a trailing EOF token, so the tables below only list
// the "real" tokens. TestEOF checks the EOF token on its own.
func withoutEOF(toks []Token) []Token {
	if len(toks) > 0 && toks[len(toks)-1].Kind == TokenEOF {
		return toks[:len(toks)-1]
	}
	return toks
}

type lexCase struct {
	name string
	in   string
	want []Token
}

func runCases(t *testing.T, cases []lexCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := withoutEOF(lex(t, tc.in))
			if !slices.Equal(got, tc.want) {
				t.Errorf("Lexer(%q)\n got: %v\nwant: %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestIdentifiers(t *testing.T) {
	runCases(t, []lexCase{
		{"single letter", "x", []Token{tok(TokenIdent, "x")}},
		{"word", "sum", []Token{tok(TokenIdent, "sum")}},
		{"leading underscore", "_private", []Token{tok(TokenIdent, "_private")}},
		{"just underscore", "_", []Token{tok(TokenIdent, "_")}},
		{"digits after first char", "a1b2", []Token{tok(TokenIdent, "a1b2")}},
		{"keyword prefix", "localValue", []Token{tok(TokenIdent, "localValue")}},
		{"keyword with suffix", "end_", []Token{tok(TokenIdent, "end_")}},
		{"keyword case matters", "Local", []Token{tok(TokenIdent, "Local")}},
	})
}

func TestKeywords(t *testing.T) {
	for word, kind := range keywords {
		t.Run(word, func(t *testing.T) {
			got := withoutEOF(lex(t, word))
			want := []Token{tok(kind, word)}
			if !slices.Equal(got, want) {
				t.Errorf("Lexer(%q)\n got: %v\nwant: %v", word, got, want)
			}
		})
	}
}

// type, export and continue are keywords only in some positions, so the
// lexer should leave them as plain identifiers and let the parser decide.
func TestContextualKeywords(t *testing.T) {
	runCases(t, []lexCase{
		{"type", "type", []Token{tok(TokenIdent, "type")}},
		{"export", "export", []Token{tok(TokenIdent, "export")}},
		{"continue", "continue", []Token{tok(TokenIdent, "continue")}},
	})
}

func TestWhitespace(t *testing.T) {
	runCases(t, []lexCase{
		{"empty input", "", nil},
		{"only spaces", "   ", nil},
		{"leading spaces", "   x", []Token{tok(TokenIdent, "x")}},
		{"trailing spaces", "x   ", []Token{tok(TokenIdent, "x")}},
		{"two words", "local x", []Token{tok(KeywordLocal, "local"), tok(TokenIdent, "x")}},
		{"tabs and newlines", "local\tx\nend", []Token{
			tok(KeywordLocal, "local"), tok(TokenIdent, "x"), tok(KeywordEnd, "end"),
		}},
		{"windows newline", "a\r\nb", []Token{tok(TokenIdent, "a"), tok(TokenIdent, "b")}},
	})
}

func TestOperators(t *testing.T) {
	runCases(t, []lexCase{
		{"assign", "=", []Token{tok(OpAssign, "=")}},
		{"greater than", ">", []Token{tok(OpGt, ">")}},
		{"parens", "()", []Token{tok(PunctLParen, "("), tok(PunctRParen, ")")}},
		{"equals is one token", "==", []Token{tok(OpEq, "==")}},
		{"greater or equal", ">=", []Token{tok(OpGe, ">=")}},
		{"compound add", "+=", []Token{tok(OpCompoundAdd, "+=")}},
		{"floor div", "//", []Token{tok(OpFloor, "//")}},
		{"compound floor", "//=", []Token{tok(OpCompoundFloor, "//=")}},
		{"concat", "..", []Token{tok(OpConcat, "..")}},
		{"ellipsis", "...", []Token{tok(PunctEllipsis, "...")}},
		{"arrow", "->", []Token{tok(PunctArrow, "->")}},
		{"double colon", "::", []Token{tok(PunctDoubleColon, "::")}},
		{"no spaces", "a=b", []Token{tok(TokenIdent, "a"), tok(OpAssign, "="), tok(TokenIdent, "b")}},
	})
}

// A byte the lexer doesn't understand should become one INVALID token,
// and lexing should carry on after it.
func TestInvalid(t *testing.T) {
	runCases(t, []lexCase{
		{"lone at sign", "@", []Token{tok(TokenInvalid, "@")}},
		{"keeps going after", "x @ y", []Token{
			tok(TokenIdent, "x"), tok(TokenInvalid, "@"), tok(TokenIdent, "y"),
		}},
	})
}

func TestEOF(t *testing.T) {
	for _, in := range []string{"", "x", "local x", "x   "} {
		t.Run(in, func(t *testing.T) {
			toks := lex(t, in)
			if len(toks) == 0 {
				t.Fatalf("Lexer(%q) returned no tokens, want at least EOF", in)
			}
			last := toks[len(toks)-1]
			if last != tok(TokenEOF, "") {
				t.Errorf("Lexer(%q) last token = %v, want EOF with empty text", in, last)
			}
		})
	}
}
