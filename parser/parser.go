package parser

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/generic"
	"github.com/jerr3n/rox/lexer"
)

// binaryOp is how tightly a binary operator binds. An operator only joins
// an expression if its left priority is above the current limit, and its
// right side is parsed with right as the new limit. right < left makes it
// right-associative, so 2^3^2 is 2^(3^2).
type binaryOp struct {
	left, right int
	op          ast.BinaryOp
}

// binaryOps uses Luau's priorities. Lowest binds loosest.
var binaryOps = map[lexer.TokenKind]binaryOp{
	lexer.KeywordOr:  {1, 1, ast.OpOr},
	lexer.KeywordAnd: {2, 2, ast.OpAnd},

	lexer.OpEq: {3, 3, ast.OpEq},
	lexer.OpNe: {3, 3, ast.OpNe},
	lexer.OpLt: {3, 3, ast.OpLt},
	lexer.OpLe: {3, 3, ast.OpLe},
	lexer.OpGt: {3, 3, ast.OpGt},
	lexer.OpGe: {3, 3, ast.OpGe},

	lexer.OpConcat: {5, 4, ast.OpConcat}, // right-associative

	lexer.OpAdd: {6, 6, ast.OpAdd},
	lexer.OpSub: {6, 6, ast.OpSub},

	lexer.OpMul:   {7, 7, ast.OpMul},
	lexer.OpDiv:   {7, 7, ast.OpDiv},
	lexer.OpFloor: {7, 7, ast.OpFloorDiv},
	lexer.OpMod:   {7, 7, ast.OpMod},

	lexer.OpPow: {10, 9, ast.OpPow}, // right-associative
}

// unaryOps are the prefix operators. They all parse their operand with
// unaryPriority as the limit, so -x^2 is -(x^2) but -x+1 is (-x)+1.
var unaryOps = map[lexer.TokenKind]ast.UnaryOp{
	lexer.OpSub:      ast.OpNeg,
	lexer.KeywordNot: ast.OpNot,
	lexer.OpLen:      ast.OpLen,
}

const unaryPriority = 8

func Parser(in []lexer.Token) {
	pos := 0
	peek := func() lexer.Token {
		return in[pos]
	}
	advance := func() lexer.Token {
		tok := in[pos]
		if tok.Kind != lexer.TokenEOF {
			pos++
		}
		return tok
	}
	accept := func(kind lexer.TokenKind) bool {
		tok := peek()
		if tok.Kind == kind {
			advance()
			return true
		}
		return false
	}
	expect := func(kind lexer.TokenKind) (*lexer.Token, error) {
		tok := peek()
		if tok.Kind != kind {
			return nil, fmt.Errorf("expected kind %q, got %q", lexer.Names[kind], lexer.Names[tok.Kind])
		}
		res := advance()
		return &res, nil
	}
	spanOf := func(tok lexer.Token) ast.Span {
		return ast.Span{
			Start: tok.Pos,
			Stop:  tok.End,
		}
	}
	dcodeStr := func(str string) (string, error) {
		mut := strings.Clone(str) // the mutable one we can easily mess with
		var out = []byte{}
		if str[0] == '"' || str[0] == '\'' {
			mut = mut[1 : len(mut)-1]
			for i := 0; i < len(mut); i++ {
				if mut[i] == '\x5c' {
					switch mut[i+1] {
					case 'n':
						out = append(out, '\n') // bullcrap
						i++
					case 't':
						out = append(out, '\t')
						i++
					case 'r':
						out = append(out, '\r')
						i++
					case 'a':
						out = append(out, '\a')
						i++
					case 'b':
						out = append(out, '\b')
						i++
					case 'f':
						out = append(out, '\f')
						i++
					case 'v':
						out = append(out, '\v')
						i++
					case '\\':
						out = append(out, '\\')
						i++
					case '"', '\'':
						out = append(out, mut[i+1])
						i++
					case '\n':
						out = append(out, '\n')
						i++
					case '\r':
						out = append(out, '\n')
						i++
						if i+1 < len(mut) && mut[i+1] == '\n' {
							i++
						}
					case 'z':
						i++
						for i+1 < len(mut) && strings.IndexByte(" \t\n\r\v\f", mut[i+1]) >= 0 {
							i++
						}
					case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
						end := i + 1
						for end < len(mut) && end < i+4 && mut[end] >= '0' && mut[end] <= '9' {
							end++
						}
						n, err := strconv.ParseUint(mut[i+1:end], 10, 8)
						if err != nil {
							return "", fmt.Errorf("escape \\%s is bigger than 255", mut[i+1:end])
						}
						out = append(out, byte(n))
						i = end - 1
					case 'x':
						if i+4 > len(mut) {
							return "", fmt.Errorf("\\x needs two hex digits")
						}
						n, err := strconv.ParseUint(mut[i+2:i+4], 16, 8)
						if err != nil {
							return "", fmt.Errorf("\\x needs two hex digits, got %q", mut[i+2:i+4])
						}
						out = append(out, byte(n))
						i += 3
					case 'u':
						if i+2 >= len(mut) || mut[i+2] != '{' {
							return "", fmt.Errorf("\\u must be followed by {")
						}
						end := strings.IndexByte(mut[i+3:], '}')
						if end < 0 {
							return "", fmt.Errorf("\\u{ is missing its closing }")
						}
						digits := mut[i+3 : i+3+end]
						n, err := strconv.ParseUint(digits, 16, 32)
						if err != nil || n > unicode.MaxRune {
							return "", fmt.Errorf("invalid code point \\u{%s}", digits)
						}
						out = utf8.AppendRune(out, rune(n))
						i += 3 + end
					default:
						// unknown escapes like \q just keep the character, as Luau does
						out = append(out, mut[i+1])
						i++
					}
				} else {
					out = append(out, mut[i])
				}
			}
			return string(out), nil
		}
		// long string: [==[ ... ]==]. The level is the number of = signs, and
		// each bracket is that many plus the two [ or ] around them.
		lvl := strings.IndexByte(mut[1:], '[')
		mut = mut[lvl+2 : len(mut)-lvl-2]
		// a newline right after the opening bracket isn't part of the string
		if strings.HasPrefix(mut, "\r\n") {
			mut = mut[2:]
		} else if strings.HasPrefix(mut, "\n") || strings.HasPrefix(mut, "\r") {
			mut = mut[1:]
		}
		return mut, nil
	}
	parseName := func() (*ast.Ident, error) {
		tok, err := expect(lexer.TokenIdent)
		if err != nil {
			return nil, err
		}
		ident := ast.Ident{
			Span: spanOf(*tok),
			Name: tok.Text,
		}
		return &ident, nil
	}
	parseLiteral := func() (ast.Expr, error) {
		tok := peek()
		advance()
		span := spanOf(tok)
		switch tok.Kind {
		case lexer.KeywordNil:
			return &ast.Nil{Span: span}, nil
		case lexer.KeywordTrue:
			return &ast.Bool{Span: span, Value: true}, nil
		case lexer.KeywordFalse:
			return &ast.Bool{Span: span, Value: false}, nil
		case lexer.TokenNumber:
			return &ast.Number{Span: span, Raw: tok.Text}, nil
		case lexer.TokenString:
			str, err := dcodeStr(tok.Text)
			if err != nil {
				return nil, err
			}
			return &ast.String{Span: span, Value: str}, nil
		case lexer.PunctEllipsis:
			return &ast.Vararg{Span: span}, nil
		default:
			return nil, fmt.Errorf("expected an expression, got <%q>", tok.Text)
		}
	}
	var parseExpr func(limit int) (ast.Expr, error)
	var parseOperand func() (ast.Expr, error)
	parseExpr = func(limit int) (ast.Expr, error) {
		tok := peek()
		x, exists := unaryOps[tok.Kind]
		var left ast.Expr
		if exists {
			advance()
			operand, err := parseExpr(unaryPriority)
			if err != nil {
				return nil, err
			}
			left = &ast.Unary{
				Span: ast.Span{Start: tok.Pos, Stop: operand.End()},
				Op:   x,
				X:    operand,
			}
		} else {
			operand, err := parseOperand()
			if err != nil {
				return nil, err
			}
			left = operand
		}
		for {
			tok := peek()
			y, exists := binaryOps[tok.Kind]
			if !exists || y.left <= limit {
				return left, nil
			}
			advance()
			right, err := parseExpr(y.right)
			if err != nil {
				return nil, err
			}
			left = &ast.Binary{
				Span: ast.Span{Start: left.Pos(), Stop: right.End()},
				Op:   y.op,
				L:    left,
				R:    right,
			}
		}
	}
	parsePrimary := func() (ast.Expr, error) {
		tok := peek()
		if tok.Kind == lexer.TokenIdent {
			advance()
			return &ast.Ident{Span: spanOf(tok), Name: tok.Text}, nil
		}
		if tok.Kind == lexer.PunctLParen {
			advance()
			inner, err := parseExpr(0)
			if err != nil {
				return nil, err
			}
			closing, err := expect(lexer.PunctRParen)
			if err != nil {
				return nil, err
			}
			return &ast.Paren{
				Span: ast.Span{Start: tok.Pos, Stop: closing.End},
				X:    inner,
			}, nil
		}
		return nil, fmt.Errorf("expected a name or (, got %q", tok.Text)
	}
	parseArgs := func() ([]ast.Expr, generic.Pos, error) {
		tok := peek()
		if tok.Kind == lexer.TokenString {
			// f"str" is f("str")
			advance()
			text, err := dcodeStr(tok.Text)
			if err != nil {
				return nil, 0, err
			}
			return []ast.Expr{&ast.String{Span: spanOf(tok), Value: text}}, tok.End, nil
		}
		_, err := expect(lexer.PunctLParen)
		if err != nil {
			return nil, 0, err
		}
		args := []ast.Expr{}
		if peek().Kind != lexer.PunctRParen {
			for {
				arg, err := parseExpr(0)
				if err != nil {
					return nil, 0, err
				}
				args = append(args, arg)
				if !accept(lexer.PunctComma) {
					break
				}
			}
		}
		closing, err := expect(lexer.PunctRParen)
		if err != nil {
			return nil, 0, err
		}
		return args, closing.End, nil
	}
	parseSuffixed := func() (ast.Expr, error) {
		expr, err := parsePrimary()
		if err != nil {
			return nil, err
		}
		for {
			tok := peek()
			switch tok.Kind {
			case lexer.PunctDot:
				advance()
				name, err := parseName()
				if err != nil {
					return nil, err
				}
				expr = &ast.Field{
					Span: ast.Span{Start: expr.Pos(), Stop: name.End()},
					X:    expr,
					Name: name,
				}
			case lexer.PunctLBracket:
				advance()
				key, err := parseExpr(0)
				if err != nil {
					return nil, err
				}
				closing, err := expect(lexer.PunctRBracket)
				if err != nil {
					return nil, err
				}
				expr = &ast.Index{
					Span: ast.Span{Start: expr.Pos(), Stop: closing.End},
					X:    expr,
					Key:  key,
				}
			case lexer.PunctColon:
				advance()
				method, err := parseName()
				if err != nil {
					return nil, err
				}
				args, end, err := parseArgs()
				if err != nil {
					return nil, err
				}
				expr = &ast.MethodCall{
					Span:   ast.Span{Start: expr.Pos(), Stop: end},
					Recv:   expr,
					Method: method,
					Args:   args,
				}
			case lexer.PunctLParen, lexer.TokenString:
				// f
				// (g)()
				// could be one call or two statements, so Luau refuses to guess
				if tok.Kind == lexer.PunctLParen && tok.NewlineBefore {
					return nil, errors.New("ambiguous syntax: this looks like a call, but could be the start of a new statement")
				}
				args, end, err := parseArgs()
				if err != nil {
					return nil, err
				}
				expr = &ast.Call{
					Span: ast.Span{Start: expr.Pos(), Stop: end},
					Fn:   expr,
					Args: args,
				}
			default:
				return expr, nil
			}
		}
	}
	parseOperand = func() (ast.Expr, error) {
		switch peek().Kind {
		case lexer.TokenIdent, lexer.PunctLParen:
			return parseSuffixed()
		}
		return parseLiteral()
	}
}
