package lexer

import (
	"errors"
	"fmt"
	"unicode"
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

// names maps each kind to how it's written, for String() and error messages.
// The [Kind]: "text" syntax sets entries by index, so order doesn't matter.
var names = [...]string{
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
	if k >= 0 && int(k) < len(names) && names[k] != "" {
		return names[k]
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

type Pos struct {
	Offset int
	//Line   int // i dont care for you
	Column int
}

type Token struct {
	Kind TokenKind
	// TODO: make `Text` *string
	Text string // the exact source text, e.g. "sum", "10", "\"hi\"", "+="
	//Pos  Pos
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
func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

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

	for pos < len(in) {
	WhiteSpaceChecker:
		for {
			switch {
			case peek(0) == ' ', peek(0) == '\n', peek(0) == '\t', peek(0) == '\r':
				pos++

			case peek(0) == '-' && peek(1) == '-': // no idea starting here
				pos += 2
				if peek(0) == '[' && (peek(1) == '[' || peek(1) == '=') {
					pos++
					pos++
					long := false
					lvl := 0
					if peek(0) == '[' {
						for {
							if peek(1+lvl) == '=' {
								lvl++
							} else {
								break
							}
						}
						if peek(1+lvl) == '[' {
							long = true
						}
					}
					if long {
						pos = pos + lvl + 2
						for {
							c := peek(0)
							if c == 0 {
								err = errors.New("unknown")
							}
							if c == ']' {
								count := 0
								for {
									if peek(1+count) == '=' {
										count++
									} else {
										break
									}
								}
								if count == lvl && peek(1+count) == ']' {
									pos = pos + lvl + 2
									break
								}
							}
							pos++
						}
					} else {
						break
					}
				} //ending here, this was hand-written pseudocode translation...

			default:
				break WhiteSpaceChecker
			}
		} // goddamn weird goland formatting
		// trailing whitespace can leave us at the end with nothing left to lex
		if peek(0) == '\x03' {
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
			if peek(0) == '0' {
				if peek(1) == 'x' || peek(1) == 'X' {
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
				}
				if peek(1) == 'b' || peek(1) == 'B' {
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
				}
			} else {
			OuterSwitch:
				switch {
				case peek(0) == '.' && peek(1) != '.':
					pos++
					cont()
					break OuterSwitch
				case peek(0) == 'e' || peek(0) == 'E':
					pos++
					if peek(0) == '+' || peek(0) == '-' {
						pos++
					}
					for {
						if isDigit(peek(0)) {
							pos++
						} else {
							break OuterSwitch
						}
					}
				default:
					cont()
					break OuterSwitch
				}

			}
			tokens = append(tokens, Token{
				Kind: TokenNumber,
				Text: in[start:pos],
			})

		case cur == '"', cur == '\'':
			pos++
			for {
				if peek(0) != '"' && peek(0) != '\x03' {
					pos++
				} else {
					break
				}
			}
			if peek(0) == '"' {
				pos++
			} else {
				err = errors.New("unterminated string")
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
			err = errors.New("operator unknown")
		}
	}
	return &tokens, err
}
