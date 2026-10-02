package lexer

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/jerr3n/rox/generic"
)

// TokenKind says what sort of token something is. Every family
// (keywords, operators, punctuation, literals) shares this one type.
type TokenKind int

const (
	TokenInvalid TokenKind = iota
	TokenEOF

	TokenIdent  // x, sum, Add
	TokenNumber // 10, 0x1F, 1e6
	TokenString // "hi", 'hi', [[hi]]

	KeywordAnd
	KeywordBreak
	KeywordDo
	KeywordElse
	KeywordElseIf
	KeywordEnd
	KeywordFalse
	KeywordFor
	KeywordFunction
	KeywordIf
	KeywordIn
	KeywordLocal
	KeywordNil
	KeywordNot
	KeywordOr
	KeywordRepeat
	KeywordReturn
	KeywordThen
	KeywordTrue
	KeywordUntil
	KeywordWhile

	// Arithmetic operators
	OpAdd    // +
	OpSub    // -
	OpMul    // *
	OpDiv    // /
	OpFloor  // //
	OpMod    // %
	OpPow    // ^
	OpLen    // #
	OpConcat // ..

	// Comparison operators
	OpEq // ==
	OpNe // ~=
	OpLt // <
	OpLe // <=
	OpGt // >
	OpGe // >=

	// Assignment
	OpAssign         // =
	OpCompoundAdd    // +=
	OpCompoundSub    // -=
	OpCompoundMul    // *=
	OpCompoundDiv    // /=
	OpCompoundFloor  // //=
	OpCompoundMod    // %=
	OpCompoundPow    // ^=
	OpCompoundConcat // ..=

	// Punctuation
	PunctLParen      // (
	PunctRParen      // )
	PunctLBracket    // [
	PunctRBracket    // ]
	PunctLBrace      // {
	PunctRBrace      // }
	PunctComma       // ,
	PunctSemicolon   // ;
	PunctDot         // .
	PunctColon       // :
	PunctDoubleColon // ::
	PunctEllipsis    // ...
	PunctArrow       // ->

	// Type syntax (only meaningful in type annotations)
	PunctQuestion // ?
	PunctPipe     // |
	PunctAmp      // &
)

// compounds early lexer will be built
var compounds = []TokenKind{
	OpAdd,
	OpSub,
	OpMul,
	OpDiv,
	OpFloor,
	OpMod,
	OpPow,
	OpLen,
}

// Names maps each kind to how it's written, for String() and error messages.
// The [Kind]: "text" syntax sets entries by index, so order doesn't matter.
var Names = [...]string{
	TokenInvalid: "INVALID",
	TokenEOF:     "EOF",
	TokenIdent:   "IDENT",
	TokenNumber:  "NUMBER",
	TokenString:  "STRING",

	KeywordAnd:      "and",
	KeywordBreak:    "break",
	KeywordDo:       "do",
	KeywordElse:     "else",
	KeywordElseIf:   "elseif",
	KeywordEnd:      "end",
	KeywordFalse:    "false",
	KeywordFor:      "for",
	KeywordFunction: "function",
	KeywordIf:       "if",
	KeywordIn:       "in",
	KeywordLocal:    "local",
	KeywordNil:      "nil",
	KeywordNot:      "not",
	KeywordOr:       "or",
	KeywordRepeat:   "repeat",
	KeywordReturn:   "return",
	KeywordThen:     "then",
	KeywordTrue:     "true",
	KeywordUntil:    "until",
	KeywordWhile:    "while",

	OpAdd:    "+",
	OpSub:    "-",
	OpMul:    "*",
	OpDiv:    "/",
	OpFloor:  "//",
	OpMod:    "%",
	OpPow:    "^",
	OpLen:    "#",
	OpConcat: "..",

	OpEq: "==",
	OpNe: "~=",
	OpLt: "<",
	OpLe: "<=",
	OpGt: ">",
	OpGe: ">=",

	OpAssign:         "=",
	OpCompoundAdd:    "+=",
	OpCompoundSub:    "-=",
	OpCompoundMul:    "*=",
	OpCompoundDiv:    "/=",
	OpCompoundFloor:  "//=",
	OpCompoundMod:    "%=",
	OpCompoundPow:    "^=",
	OpCompoundConcat: "..=",

	PunctLParen:      "(",
	PunctRParen:      ")",
	PunctLBracket:    "[",
	PunctRBracket:    "]",
	PunctLBrace:      "{",
	PunctRBrace:      "}",
	PunctComma:       ",",
	PunctSemicolon:   ";",
	PunctDot:         ".",
	PunctColon:       ":",
	PunctDoubleColon: "::",
	PunctEllipsis:    "...",
	PunctArrow:       "->",

	PunctQuestion: "?",
	PunctPipe:     "|",
	PunctAmp:      "&",
}

// String lets fmt print a kind as its text, e.g. fmt.Println(KeywordLocal) prints "local".
func (k TokenKind) String() string {
	if k >= 0 && int(k) < len(Names) && Names[k] != "" {
		return Names[k]
	}
	return fmt.Sprintf("TokenKind(%d)", int(k))
}

var keywords = map[string]TokenKind{
	"and":      KeywordAnd,
	"break":    KeywordBreak,
	"do":       KeywordDo,
	"else":     KeywordElse,
	"elseif":   KeywordElseIf,
	"end":      KeywordEnd,
	"false":    KeywordFalse,
	"for":      KeywordFor,
	"function": KeywordFunction,
	"if":       KeywordIf,
	"in":       KeywordIn,
	"local":    KeywordLocal,
	"nil":      KeywordNil,
	"not":      KeywordNot,
	"or":       KeywordOr,
	"repeat":   KeywordRepeat,
	"return":   KeywordReturn,
	"then":     KeywordThen,
	"true":     KeywordTrue,
	"until":    KeywordUntil,
	"while":    KeywordWhile,
}

var operators = map[string]TokenKind{
	// Arithmetic operators
	"+":  OpAdd,
	"-":  OpSub,
	"*":  OpMul,
	"/":  OpDiv,
	"//": OpFloor,
	"%":  OpMod,
	"^":  OpPow,
	"#":  OpLen,
	"..": OpConcat,

	// Comparison operators
	"==": OpEq,
	"~=": OpNe,
	"<":  OpLt,
	"<=": OpLe,
	">":  OpGt,
	">=": OpGe,

	// Assignment
	"=":   OpAssign,
	"+=":  OpCompoundAdd,
	"-=":  OpCompoundSub,
	"*=":  OpCompoundMul,
	"/=":  OpCompoundDiv,
	"//=": OpCompoundFloor,
	"%=":  OpCompoundMod,
	"^=":  OpCompoundPow,
	"..=": OpCompoundConcat,

	// Punctuation
	"(":   PunctLParen,
	")":   PunctRParen,
	"[":   PunctLBracket,
	"]":   PunctRBracket,
	"{":   PunctLBrace,
	"}":   PunctRBrace,
	",":   PunctComma,
	";":   PunctSemicolon,
	".":   PunctDot,
	":":   PunctColon,
	"::":  PunctDoubleColon,
	"...": PunctEllipsis,
	"->":  PunctArrow,

	// Type syntax
	"?": PunctQuestion,
	"|": PunctPipe,
	"&": PunctAmp,
}

// Pos is a byte offset into the source, the same as ast.Pos.
type Pos int

type Token struct {
	Kind TokenKind
	// TODO: make `Text` *string
	Text string      // the exact source text, e.g. "sum", "10", "\"hi\"", "+="
	Pos  generic.Pos // first byte of the token
	End  generic.Pos // one past the last byte, so in[Pos:End] == Text
	// NewlineBefore is true if a newline was skipped between the previous
	// token and this one, including one inside a comment. The parser needs
	// it for the f\n(x) ambiguity.
	NewlineBefore bool
}

/*
local a = 2
^^^^^^^   ^
IDENT A = NUM 2
if a      > 1     then
^^ ^^^^^^^  ^     ^^^^
IF IDENT A  NUM 1 THEN

		print         (           "larger"      )
	    ^^^^^         ^           ^^^^^^^^      ^
	    IDENT PRINT   RPAREN      STRING larger LPAREN

end
^^^
END
*/

func Lexer(in string) (*[]Token, error) {
	pos := 0
	tokens := []Token{}
	var err error = nil

	peek := func(n int) rune {
		if pos+n >= len(in) {
			return '\x03'
		}
		return rune(in[pos+n])
	}

	isDigit := func(c rune) bool {
		if c >= '0' && c <= '9' {
			return true
		}
		return false
	}

	// longLevel reports the level of the long bracket at pos: 0 for [[,
	// 1 for [=[, 2 for [==[ and so on. It returns -1 if pos isn't at one.
	longLevel := func() int {
		if peek(0) != '[' {
			return -1
		}
		lvl := 0
		for peek(1+lvl) == '=' {
			lvl++
		}
		if peek(1+lvl) != '[' {
			return -1
		}
		return lvl
	}

	// skipLong moves pos past a long bracket like [==[ ... ]==]. Only a
	// closing bracket of the same level ends it. It returns false if the
	// input runs out first.
	skipLong := func(lvl int) bool {
		pos += lvl + 2
		for pos < len(in) {
			if peek(0) == ']' {
				count := 0
				for peek(1+count) == '=' {
					count++
				}
				if count == lvl && peek(1+count) == ']' {
					pos += lvl + 2
					return true
				}
			}
			pos++
		}
		return false
	}

	for pos < len(in) {
		skipStart := pos
	WhiteSpaceChecker:
		for {
			switch {
			case peek(0) == ' ', peek(0) == '\n', peek(0) == '\t', peek(0) == '\r':
				pos++

			case peek(0) == '-' && peek(1) == '-':
				pos += 2
				if lvl := longLevel(); lvl >= 0 {
					// --[[ long comment ]], which can span lines
					if !skipLong(lvl) {
						err = errors.New("unterminated long comment")
					}
				} else {
					// short comment: runs to the end of the line. This also
					// covers --[ and --[= that aren't a full long bracket.
					for pos < len(in) && peek(0) != '\n' {
						pos++
					}
				}

			default:
				break WhiteSpaceChecker
			}
		} // goddamn weird goland formatting
		// trailing whitespace can leave us at the end with nothing left to lex
		if pos >= len(in) {
			break
		}
		start := pos
		cur := peek(0)

		switch {
		case unicode.IsLetter(cur), cur == '_':
			for {
				if unicode.IsLetter(peek(0)) || isDigit(peek(0)) || peek(0) == '_' {
					pos++
				} else {
					break
				}
			}
			word := in[start:pos]
			enum, ok := keywords[word]
			if ok {
				tokens = append(tokens, Token{
					Kind: enum,
					Text: word,
				})
			} else {
				tokens = append(tokens, Token{
					Kind: TokenIdent,
					Text: word,
				})
			}
		case isDigit(cur), cur == '.' && isDigit(peek(1)):
			cont := func() {
				for {
					if isDigit(peek(0)) || peek(0) == '_' {
						pos++
					} else {
						break
					}
				}
			}
			if peek(0) == '0' && (peek(1) == 'x' || peek(1) == 'X') {
				pos++
				pos++
				for {
					c := peek(0)
					if (isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) || c == '_' {
						pos++
					} else {
						break
					}
				}
			} else if peek(0) == '0' && (peek(1) == 'b' || peek(1) == 'B') {
				pos++
				pos++
				for {
					c := peek(0)
					if c == '0' || c == '1' || c == '_' {
						pos++
					} else {
						break
					}
				}
			} else {
				// decimal: whole part, then an optional fraction, then an
				// optional exponent, e.g. 10, 3.14, .5, 2.5e-3. Either part
				// can be missing (".5" has no whole part, "1e6" no fraction).
				cont()
				// a second dot means .. (concat), as in 1..2, so leave it
				if peek(0) == '.' && peek(1) != '.' {
					pos++
					cont()
				}
				if peek(0) == 'e' || peek(0) == 'E' {
					pos++
					if peek(0) == '+' || peek(0) == '-' {
						pos++
					}
					cont()
				}
			}
			tokens = append(tokens, Token{
				Kind: TokenNumber,
				Text: in[start:pos],
			})

		case cur == '"', cur == '\'':
			pos++
			// a raw newline ends the string early, which is an error
			for pos < len(in) && peek(0) != cur && peek(0) != '\n' {
				// skip whatever follows a backslash, so \" and \' don't end
				// the string, \\ doesn't escape the quote after it, and
				// \<newline> carries the string onto the next line
				if peek(0) == '\\' && pos+1 < len(in) {
					pos++
				}
				pos++
			}
			if peek(0) == cur {
				pos++
			} else {
				err = errors.New("unterminated string")
			}
			tokens = append(tokens, Token{
				Kind: TokenString,
				Text: in[start:pos],
			})

		case cur == '[' && longLevel() >= 0:
			// [[long string]] or [==[long string]==], no escapes inside
			if !skipLong(longLevel()) {
				err = errors.New("unterminated long string")
			}
			tokens = append(tokens, Token{
				Kind: TokenString,
				Text: in[start:pos],
			})

		default:
			next := peek(1)
			triple := string([]rune{cur, next, peek(2)})
			op, exists := operators[triple]
			if exists {
				tokens = append(tokens, Token{
					Kind: op,
					Text: triple,
				})
				pos += 3
				break
			}
			double := string([]rune{cur, next}) //see what i did there
			op, exists = operators[double]
			if exists {
				tokens = append(tokens, Token{
					Kind: op,
					Text: double,
				})
				pos += 2
				break
			}
			op, exists = operators[string(cur)]
			if exists {
				tokens = append(tokens, Token{
					Kind: op,
					Text: string(cur),
				})
				pos++
				break
			}
			// not something Luau knows. Keep it as an INVALID token and step
			// past it, otherwise we'd look at the same byte forever.
			err = fmt.Errorf("unknown character %q", in[pos])
			tokens = append(tokens, Token{
				Kind: TokenInvalid,
				Text: in[pos : pos+1],
			})
			pos++
		}

		// every case above appends exactly one token, so fill in where it is
		// here instead of in each case
		tk := &tokens[len(tokens)-1]
		tk.Pos = generic.Pos(start)
		tk.End = generic.Pos(pos)
		tk.NewlineBefore = strings.IndexByte(in[skipStart:start], '\n') >= 0
	}
	// EOF sits at the very end of the source, so the parser always has a
	// token to look at, even when the input stops mid-statement
	tokens = append(tokens, Token{
		Kind: TokenEOF,
		Pos:  generic.Pos(len(in)),
		End:  generic.Pos(len(in)),
	})
	return &tokens, err
}
