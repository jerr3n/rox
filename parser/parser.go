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

var compoundOps = map[lexer.TokenKind]ast.BinaryOp{
	lexer.OpCompoundAdd:    ast.OpAdd,
	lexer.OpCompoundSub:    ast.OpSub,
	lexer.OpCompoundMul:    ast.OpMul,
	lexer.OpCompoundDiv:    ast.OpDiv,
	lexer.OpCompoundFloor:  ast.OpFloorDiv,
	lexer.OpCompoundMod:    ast.OpMod,
	lexer.OpCompoundPow:    ast.OpPow,
	lexer.OpCompoundConcat: ast.OpConcat,
}

const unaryPriority = 8

func isAssignable(target ast.Expr) bool {
	switch target.(type) {
	case *ast.Ident, *ast.Field, *ast.Index:
		return true
	default:
		return false
	}
}

func Parser(in []lexer.Token) (*ast.Block, error) {
	pos := 0
	// prev TODO: generic.Pos() prev
	prev := 0
	var parseLocal func() (ast.Stmt, error)
	var parseIf func() (ast.Stmt, error)
	var parseWhile func() (ast.Stmt, error)
	var parseDo func() (ast.Stmt, error)
	var parseFor func() (ast.Stmt, error)
	var parseRepeat func() (ast.Stmt, error)
	var parseFunctionDecl func() (ast.Stmt, error)
	var parseExprStatement func() (ast.Stmt, error)
	var parseStatement func() (ast.Stmt, error)
	var parseBlock func() (*ast.Block, error)
	var parseReturn func() (ast.Stmt, error)
	var parseExprList func() ([]ast.Expr, error)
	var parseBinding func() (*ast.Binding, error)
	var parseFuncBody func() (*ast.FuncBody, error)
	var peek func() lexer.Token
	var peekAt func(n int) lexer.Token
	var advance func() lexer.Token
	var accept func(kind lexer.TokenKind) bool
	var expect func(kind lexer.TokenKind) (*lexer.Token, error)
	var spanOf func(tok lexer.Token) ast.Span
	var dcodeStr func(str string) (string, error)
	var parseName func() (*ast.Ident, error)
	var parseLiteral func() (ast.Expr, error)
	var parsePrimary func() (ast.Expr, error)
	var parseArgs func() ([]ast.Expr, generic.Pos, error)
	var parseSuffixed func() (ast.Expr, error)
	var parseBindingList func() ([]*ast.Binding, error)
	var blockEnd func(kind lexer.TokenKind) bool
	var parseTable func() (ast.Expr, error)
	var parseTernary func() (ast.Expr, error)
	var parseExpr func(limit int) (ast.Expr, error)
	var parseOperand func() (ast.Expr, error)

	peek = func() lexer.Token {
		return in[pos]
	}
	peekAt = func(n int) lexer.Token {
		// past the end, keep returning the last token, which is EOF
		if pos+n >= len(in) {
			return in[len(in)-1]
		}
		return in[pos+n]
	}
	advance = func() lexer.Token {
		tok := in[pos]
		if tok.Kind != lexer.TokenEOF {
			pos++
		}
		prev = int(tok.End)
		return tok
	}
	accept = func(kind lexer.TokenKind) bool {
		tok := peek()
		if tok.Kind == kind {
			advance()
			return true
		}
		return false
	}
	expect = func(kind lexer.TokenKind) (*lexer.Token, error) {
		tok := peek()
		if tok.Kind != kind {
			return nil, fmt.Errorf("expected kind %q, got %q", lexer.Names[kind], lexer.Names[tok.Kind])
		}
		res := advance()
		return &res, nil
	}
	spanOf = func(tok lexer.Token) ast.Span {
		return ast.Span{
			Start: tok.Pos,
			Stop:  tok.End,
		}
	}
	dcodeStr = func(str string) (string, error) {
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
	parseName = func() (*ast.Ident, error) {
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
	parseLiteral = func() (ast.Expr, error) {
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
	parsePrimary = func() (ast.Expr, error) {
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
	parseArgs = func() ([]ast.Expr, generic.Pos, error) {
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
		if tok.Kind == lexer.PunctLBrace {
			// f{...} is f({...})
			table, err := parseTable()
			if err != nil {
				return nil, 0, err
			}
			return []ast.Expr{table}, table.End(), nil
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
	parseSuffixed = func() (ast.Expr, error) {
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
			case lexer.PunctLParen, lexer.TokenString, lexer.PunctLBrace:
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
		tok := peek()
		var val ast.Expr
		var err error
		switch tok.Kind {
		case lexer.TokenIdent, lexer.PunctLParen:
			val, err = parseSuffixed()
		case lexer.PunctLBrace:
			val, err = parseTable()
		case lexer.KeywordIf:
			val, err = parseTernary()
		case lexer.KeywordFunction:
			advance()
			body, err := parseFuncBody()
			if err != nil {
				return nil, err
			}
			val = &ast.Func{
				Span: ast.Span{Start: tok.Pos, Stop: body.End()},
				Body: body,
			}
		}
		if err != nil {
			return nil, err
		}
		if val != nil {
			return val, nil
		}
		return parseLiteral()
	}
	parseStatement = func() (ast.Stmt, error) {
		tok := peek()
		var target ast.Stmt
		var err error
		switch tok.Kind {
		case lexer.KeywordLocal:
			target, err = parseLocal()
		case lexer.KeywordIf:
			target, err = parseIf()
		case lexer.KeywordWhile:
			target, err = parseWhile()
		case lexer.KeywordDo:
			target, err = parseDo()
		case lexer.KeywordFor:
			target, err = parseFor()
		case lexer.KeywordRepeat:
			target, err = parseRepeat()
		case lexer.KeywordFunction:
			target, err = parseFunctionDecl()
		case lexer.KeywordBreak:
			advance()
			return &ast.Break{Span: spanOf(tok)}, nil
		default:
			target, err = parseExprStatement()
		}
		if err != nil {
			return nil, err
		}
		return target, nil
	}
	parseExprList = func() ([]ast.Expr, error) {
		exprs := []ast.Expr{}
		initial, err := parseExpr(0)
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, initial)
		for {
			target := accept(lexer.PunctComma)
			if target {
				expr, err := parseExpr(0)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)

			} else {
				break
			}
		}
		return exprs, nil
	}
	parseBinding = func() (*ast.Binding, error) {
		name, err := parseName()
		if err != nil {
			return nil, err
		}
		if peek().Kind == lexer.PunctColon {
			return nil, errors.New("type implementation coming out soon (if it does come out)")
		}
		return &ast.Binding{
			Span: ast.Span{
				Start: name.Pos(),
				Stop:  generic.Pos(prev),
			},
			Name: name,
		}, nil
	}
	parseBindingList = func() ([]*ast.Binding, error) {
		target, err := parseBinding()
		if err != nil {
			return nil, err
		}
		names := []*ast.Binding{
			target,
		}
		for {
			if accept(lexer.PunctComma) {
				target, err = parseBinding()
				if err != nil {
					return nil, err
				}
				names = append(names, target)
			} else {
				break
			}
		}
		return names, nil
	}
	blockEnd = func(kind lexer.TokenKind) bool {
		switch kind {
		case lexer.KeywordEnd, lexer.KeywordElse, lexer.KeywordElseIf, lexer.KeywordUntil, lexer.TokenEOF:
			return true
		default:
			return false
		}
	}
	parseBlock = func() (*ast.Block, error) {
		start := peek().Pos
		var stmts []ast.Stmt
		for {
			cur := peek()
			if blockEnd(cur.Kind) {
				break
			} else {
				if cur.Kind == lexer.KeywordReturn {
					target, err := parseReturn()
					if err != nil {
						return nil, err
					}
					stmts = append(stmts, target)
					accept(lexer.PunctSemicolon)
					break
				}
				target, err := parseStatement()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, target)
				accept(lexer.PunctSemicolon)
			}
		}
		return &ast.Block{
			Span:  ast.Span{Start: start, Stop: generic.Pos(prev)},
			Stmts: stmts,
		}, nil
	}
	parseReturn = func() (ast.Stmt, error) {
		start, err := expect(lexer.KeywordReturn)
		if err != nil {
			return nil, err
		}
		vals := []ast.Expr{}
		cur := peek()
		if !blockEnd(cur.Kind) && cur.Kind != lexer.PunctSemicolon {
			vals, err = parseExprList()
			if err != nil {
				return nil, err
			}
		}
		return &ast.Return{
			Span: ast.Span{
				Start: start.Pos,
				Stop:  generic.Pos(prev),
			},
			Values: vals,
		}, nil
	}
	parseLocal = func() (ast.Stmt, error) {
		start, err := expect(lexer.KeywordLocal)
		if err != nil {
			return nil, err
		}
		if accept(lexer.KeywordFunction) {
			name, err := parseName()
			if err != nil {
				return nil, err
			}
			body, err := parseFuncBody()
			if err != nil {
				return nil, err
			}
			return &ast.LocalFunc{
				Span: ast.Span{
					Start: start.Pos,
					Stop:  body.Stop,
				},
				Name: name,
				Body: body,
			}, nil
		}
		names, err := parseBindingList()
		if err != nil {
			return nil, err
		}
		var vals []ast.Expr
		if accept(lexer.OpAssign) {
			vals, err = parseExprList()
			if err != nil {
				return nil, err
			}
		}
		return &ast.Local{
			Span: ast.Span{
				Start: start.Pos,
				Stop:  generic.Pos(prev),
			},
			Names:  names, //??????
			Values: vals,
		}, nil
	}
	parseDo = func() (ast.Stmt, error) {
		start, err := expect(lexer.KeywordDo)
		if err != nil {
			return nil, err
		}
		body, err := parseBlock()
		if err != nil {
			return nil, err
		}
		finish, err := expect(lexer.KeywordEnd)
		if err != nil {
			return nil, err
		}
		return &ast.Do{
			Span: ast.Span{
				Start: start.Pos,
				Stop:  finish.End,
			},
			Body: body,
		}, nil
	}
	parseWhile = func() (ast.Stmt, error) {
		start, err := expect(lexer.KeywordWhile)
		if err != nil {
			return nil, err
		}
		cond, err := parseExpr(0)
		if err != nil {
			return nil, err
		}
		_, err = expect(lexer.KeywordDo)
		if err != nil {
			return nil, err
		}
		body, err := parseBlock()
		if err != nil {
			return nil, err
		}
		finish, err := expect(lexer.KeywordEnd)
		if err != nil {
			return nil, err
		}
		return &ast.While{
			Span: ast.Span{
				Start: start.Pos,
				Stop:  finish.End,
			},
			Cond: cond,
			Body: body,
		}, nil
	}
	parseRepeat = func() (ast.Stmt, error) {
		start, err := expect(lexer.KeywordRepeat)
		if err != nil {
			return nil, err
		}
		body, err := parseBlock()
		if err != nil {
			return nil, err
		}
		_, err = expect(lexer.KeywordUntil)
		if err != nil {
			return nil, err
		}
		cond, err := parseExpr(0)
		return &ast.Repeat{
			Span: ast.Span{
				Start: start.Pos,
				Stop:  cond.End(),
			},
			Body: body,
			Cond: cond,
		}, nil
	}
	parseFor = func() (ast.Stmt, error) {
		start, err := expect(lexer.KeywordFor)
		if err != nil {
			return nil, err
		}
		first, err := parseBinding()
		if err != nil {
			return nil, err
		}
		if accept(lexer.OpAssign) {
			from, err := parseExpr(0)
			if err != nil {
				return nil, err
			}
			_, err = expect(lexer.PunctComma)
			if err != nil {
				return nil, err
			}
			to, err := parseExpr(0)
			if err != nil {
				return nil, err
			}
			var step ast.Expr
			if accept(lexer.PunctComma) {
				step, err = parseExpr(0)
				if err != nil {
					return nil, err
				}
			}
			_, err = expect(lexer.KeywordDo)
			if err != nil {
				return nil, err
			}
			body, err := parseBlock()
			if err != nil {
				return nil, err
			}
			finish, err := expect(lexer.KeywordEnd)
			return &ast.NumericFor{
				Span: ast.Span{
					Start: start.Pos,
					Stop:  finish.End,
				},
				Var:   first,
				Start: from,
				Stop:  to,
				Step:  step,
				Body:  body,
			}, nil
		}
		vars := []*ast.Binding{first}
		for {
			if accept(lexer.PunctComma) {
				target, err := parseBinding()
				if err != nil {
					return nil, err
				}
				vars = append(vars, target)
			} else {
				break
			}
		}
		_, err = expect(lexer.KeywordIn)
		if err != nil {
			return nil, err
		}
		exprs, err := parseExprList()
		if err != nil {
			return nil, err
		}
		_, err = expect(lexer.KeywordDo)
		if err != nil {
			return nil, err
		}
		body, err := parseBlock()
		if err != nil {
			return nil, err
		}
		finish, err := expect(lexer.KeywordEnd)
		if err != nil {
			return nil, err
		}
		return &ast.GenericFor{
			Span: ast.Span{
				Start: start.Pos,
				Stop:  finish.End,
			},
			Vars:  vars,
			Exprs: exprs,
			Body:  body,
		}, nil
	}
	parseIf = func() (ast.Stmt, error) {
		start, err := expect(lexer.KeywordIf)
		if err != nil {
			return nil, err
		}
		cond, err := parseExpr(0)
		if err != nil {
			return nil, err
		}
		_, err = expect(lexer.KeywordThen)
		if err != nil {
			return nil, err
		}
		then, err := parseBlock()
		if err != nil {
			return nil, err
		}
		elseifs := []*ast.ElseIf{} // the more i make rox the less respect i have for luau devs
		for {
			if peek().Kind == lexer.KeywordElseIf {
				tok := advance()
				cond, err := parseExpr(0)
				if err != nil {
					return nil, err
				}
				_, err = expect(lexer.KeywordThen)
				if err != nil {
					return nil, err
				}
				body, err := parseBlock()
				if err != nil {
					return nil, err
				}
				elseifs = append(elseifs, &ast.ElseIf{
					Span: ast.Span{
						Start: tok.Pos,
						Stop:  generic.Pos(prev),
					},
					Cond: cond,
					Body: body,
				})
			} else {
				break
			}
		}
		var elseBlock *ast.Block
		if accept(lexer.KeywordElse) {
			target, err := parseBlock()
			if err != nil {
				return nil, err
			}
			elseBlock = target
		}
		finish, err := expect(lexer.KeywordEnd)
		if err != nil {
			return nil, err
		}
		return &ast.If{
			Span: ast.Span{
				Start: start.Pos,
				Stop:  finish.End,
			},
			Cond:    cond,
			Then:    then,
			ElseIfs: elseifs,
			Else:    elseBlock,
		}, nil
	}
	parseFunctionDecl = func() (ast.Stmt, error) {
		start, err := expect(lexer.KeywordFunction)
		if err != nil {
			return nil, err
		}
		first, err := parseName()
		if err != nil {
			return nil, err
		}
		path := []*ast.Ident{first}
		for {
			if accept(lexer.PunctDot) {
				target, err := parseName()
				if err != nil {
					return nil, err
				}
				path = append(path, target)
			} else {
				break
			}
		}
		var method *ast.Ident
		if accept(lexer.PunctColon) {
			method, err = parseName()
			if err != nil {
				return nil, err
			}
		}
		name := ast.FuncName{
			Span: ast.Span{
				Start: first.Pos(),
				Stop:  generic.Pos(prev),
			},
			Path:   path,
			Method: method,
		}
		body, err := parseFuncBody()
		if err != nil {
			return nil, err
		}
		return &ast.FuncDecl{
			Span: ast.Span{
				Start: start.Pos,
				Stop:  body.End(),
			},
			Name: &name,
			Body: body,
		}, nil
	}
	parseFuncBody = func() (*ast.FuncBody, error) {
		open, err := expect(lexer.PunctLParen)
		vararg := false
		var params []*ast.Binding
		if peek().Kind != lexer.PunctRParen {
			for {
				if accept(lexer.PunctEllipsis) {
					vararg = true
					break
				}
				target, err := parseBinding()
				if err != nil {
					return nil, err
				}
				params = append(params, target)
				if !accept(lexer.PunctComma) {
					break
				}
			}
		}
		_, err = expect(lexer.PunctRParen)
		if err != nil {
			return nil, err
		}
		if peek().Kind == lexer.PunctColon {
			return nil, errors.New("type support not yet implemented")
		}
		body, err := parseBlock()
		if err != nil {
			return nil, err
		}
		finish, err := expect(lexer.KeywordEnd)
		if err != nil {
			return nil, err
		}
		return &ast.FuncBody{
			Span: ast.Span{
				Start: open.Pos,
				Stop:  finish.End,
			},
			Params: params,
			Vararg: vararg,
			Body:   body,
		}, nil
	}
	parseExprStatement = func() (ast.Stmt, error) {
		first, err := parseSuffixed()
		if err != nil {
			return nil, err
		}
		tok := peek()
		val, ok := first.(*ast.Ident)
		which, exists := compoundOps[tok.Kind]
		if ok && val.Name == "continue" && tok.Kind != lexer.OpAssign && tok.Kind != lexer.PunctComma && !exists {
			return &ast.Continue{Span: val.Span}, nil
		}
		if tok.Kind == lexer.OpAssign || tok.Kind == lexer.PunctComma {
			var targets []ast.Expr = []ast.Expr{first}
			for {
				if accept(lexer.PunctComma) {
					target, err := parseSuffixed()
					if err != nil {
						return nil, err
					}
					targets = append(targets, target)
				} else {
					break
				}
			}
			for _, t := range targets {
				if !isAssignable(t) {
					return nil, fmt.Errorf("cannot assign to <%s>", t)
				}
			}
			_, err = expect(lexer.OpAssign)
			if err != nil {
				return nil, err
			}
			vals, err := parseExprList()
			if err != nil {
				return nil, err
			}
			return &ast.Assign{
				Span: ast.Span{
					Start: first.Pos(),
					Stop:  generic.Pos(prev),
				},
				Targets: targets,
				Values:  vals, // my coding style is so inconsistent
			}, nil
		}
		if exists {
			if !isAssignable(first) {
				return nil, fmt.Errorf("cannot assign to <%s>", first)
			}
			advance()
			value, err := parseExpr(0)
			if err != nil {
				return nil, err
			}
			return &ast.CompoundAssign{
				Span: ast.Span{
					Start: first.Pos(),
					Stop:  value.End(),
				},
				Op:     which,
				Target: first,
				Value:  value,
			}, nil
		}
		switch first.(type) {
		case *ast.Call, *ast.MethodCall:
			return &ast.CallStmt{Span: ast.Span{Start: first.Pos(), Stop: first.End()}, Call: first}, nil
		default:
			break
		}
		return nil, fmt.Errorf("expected a statement, got %s", first)
	}
	parseTable = func() (ast.Expr, error) {
		open, err := expect(lexer.PunctLBrace)
		if err != nil {
			return nil, err
		}
		var items []*ast.TableItem
		for {
			if peek().Kind != lexer.PunctRBrace {
				tok := peek()
				var item ast.TableItem
				if tok.Kind == lexer.PunctLBracket {
					advance()
					key, err := parseExpr(0)
					if err != nil {
						return nil, err
					}
					_, err = expect(lexer.PunctRBracket)
					if err != nil {
						return nil, err
					}
					_, err = expect(lexer.OpAssign)
					if err != nil {
						return nil, err
					}
					value, err := parseExpr(0)
					if err != nil {
						return nil, err
					}
					item = ast.TableItem{
						Kind:  ast.ItemKeyed,
						Key:   key,
						Value: value,
					}
				} else if tok.Kind == lexer.TokenIdent && peekAt(1).Kind == lexer.OpAssign {
					name, err := parseName()
					if err != nil {
						return nil, err
					}
					advance()
					value, err := parseExpr(0)
					if err != nil {
						return nil, err
					}
					item = ast.TableItem{
						Kind:  ast.ItemNamed,
						Name:  name,
						Value: value,
					}
				} else {
					value, err := parseExpr(0)
					if err != nil {
						return nil, err
					}
					item = ast.TableItem{Kind: ast.ItemPositional, Value: value}
				}
				item.Span = ast.Span{
					Start: tok.Pos,
					Stop:  generic.Pos(prev),
				}
				items = append(items, &item)
				if !accept(lexer.PunctComma) && !accept(lexer.PunctSemicolon) {
					break
				}
			} else {
				break
			}
		}
		c, err := expect(lexer.PunctRBrace)
		if err != nil {
			return nil, err
		}
		return &ast.Table{
			Span: ast.Span{
				Start: open.Pos,
				Stop:  c.End,
			},
			Items: items,
		}, nil
	}
	parseTernary = func() (ast.Expr, error) {
		start, err := expect(lexer.KeywordIf)
		if err != nil {
			return nil, err
		}
		cond, err := parseExpr(0)
		if err != nil {
			return nil, err
		}
		_, err = expect(lexer.KeywordThen)
		if err != nil {
			return nil, err
		}
		then, err := parseExpr(0)
		if err != nil {
			return nil, err
		}
		var elseifs []*ast.ElseIfExpr
		for {
			if peek().Kind == lexer.KeywordElseIf {
				tok := advance()
				cond, err := parseExpr(0)
				if err != nil {
					return nil, err
				}
				_, err = expect(lexer.KeywordThen)
				if err != nil {
					return nil, err
				}
				then, err := parseExpr(0)
				if err != nil {
					return nil, err
				}
				elseifs = append(elseifs, &ast.ElseIfExpr{
					Span: ast.Span{
						Start: tok.Pos,
						Stop:  then.End(),
					},
					Cond: cond,
					Then: then,
				})
			} else {
				break
			}
		}
		_, err = expect(lexer.KeywordElse)
		if err != nil {
			return nil, err
		}
		elseval, err := parseExpr(0)
		if err != nil {
			return nil, err
		}
		return &ast.IfExpr{
			Span: ast.Span{
				Start: start.Pos,
				Stop:  elseval.End(),
			},
			Cond:    cond,
			Then:    then,
			ElseIfs: elseifs,
			Else:    elseval,
		}, nil
	}
	block, err := parseBlock()
	if err != nil {
		return nil, err
	}
	// anything left over means a statement ended early, like the :upper() in "s":upper()
	if _, err := expect(lexer.TokenEOF); err != nil {
		return nil, err
	}
	return block, nil
}
