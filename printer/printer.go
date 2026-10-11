package printer

import (
	"fmt"
	"strings"

	"github.com/jerr3n/rox/ast"
)

// Config picks the output style.
type Config struct {
	Indent string // one level of indentation, e.g. "\t" or "  "
	Minify bool   // fewest bytes: no indentation, no newlines (step 9)
}

// prec is how tightly a binary operator binds, with the same numbers as the
// parser's binaryOps.
type prec struct {
	left, right int
}

// binaryText is the source text of each binary operator.
var binaryText = map[ast.BinaryOp]string{}

// unaryText is the source text of each unary operator ("not " keeps its space).
var unaryText = map[ast.UnaryOp]string{}

// binaryPrec is each binary operator's priority, copied from the parser.
var binaryPrec = map[ast.BinaryOp]prec{}

// unaryPrec matches the parser's unaryPriority.
const unaryPrec = 8

// maxPower means "can't be pulled apart" in power.
const maxPower = 100

// isWordChar reports whether c is a letter, digit or '_'.
func isWordChar(c byte) bool {
	panic("TODO")
}

// quote turns a decoded string back into a double-quoted Luau literal.
func quote(s string) string {
	var out strings.Builder
	out.WriteByte('"')
	// bytes, not runes: a string can hold invalid UTF-8, and range would
	// turn it into U+FFFD
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			out.WriteString(`\"`)
		case c == '\\':
			out.WriteString(`\\`)
		case c == '\n':
			out.WriteString(`\n`)
		case c == '\r':
			out.WriteString(`\r`)
		case c == '\t':
			out.WriteString(`\t`)
		case c < 0x20 || c == 0x7F:
			// always 3 digits, so a digit after it isn't read as part of it
			fmt.Fprintf(&out, `\%03d`, c)
		default:
			out.WriteByte(c) // bytes >= 0x80 too, so UTF-8 passes through
		}
	}
	out.WriteByte('"')
	return out.String()
}

// escapeInterp is quote for interpolated string parts: also escapes ` and {.
func escapeInterp(s string) string {
	panic("TODO")
}

// power returns how tightly e holds on to its left and right neighbours.
func power(e ast.Expr) (left, right int) {
	switch e.(type) {
	case *ast.Binary:
		return binaryPrec[e.Op].left, binaryPrec[e.Op].right
	case *ast.Unary:
		return maxPower, unaryPrec
	case *ast.IfExpr:
		return 0, 0
	default:
		return maxPower, maxPower
	}
}

// startsWithParen reports whether a statement's text begins with "(".
func startsWithParen(s ast.Stmt) bool {
	panic("TODO")
}

// leftmost walks down the left edge of a suffix chain: (a).b[c]:d() -> (a).
func leftmost(e ast.Expr) ast.Expr {
	panic("TODO")
}

// fitsOnOneLine decides between a one-line and a multi-line table.
func fitsOnOneLine(t *ast.Table) bool {
	panic("TODO")
}

// Printer turns a block back into Luau source.
func Printer(file *ast.Block, cfg Config) string {
	var out strings.Builder // the text so far
	depth := 0              // current indent level
	lineStart := true       // true right after a newline; the next write emits the indent first
	var last byte           // last byte written, 0 at the start (step 9)
	lastNumber := false     // the last token written was a Number literal (step 9)

	// output helpers
	var write func(str string)

	var newline func()
	var indent func()
	var dedent func()
	var list func(n int, sep string, item func(i int))
	var needSpace func(a, b byte) bool

	// blocks
	var blockStmts func(b *ast.Block)
	var body func(b *ast.Block)

	// statements
	var stmt func(s ast.Stmt)
	var local func(s *ast.Local)
	var localFunc func(s *ast.LocalFunc)
	var funcDecl func(s *ast.FuncDecl)
	var assign func(s *ast.Assign)
	var compoundAssign func(s *ast.CompoundAssign)
	var callStmt func(s *ast.CallStmt)
	var doStmt func(s *ast.Do)
	var whileStmt func(s *ast.While)
	var repeatStmt func(s *ast.Repeat)
	var ifStmt func(s *ast.If)
	var numericFor func(s *ast.NumericFor)
	var genericFor func(s *ast.GenericFor)
	var returnStmt func(s *ast.Return)
	var breakStmt func(s *ast.Break)
	var continueStmt func(s *ast.Continue)

	// shared pieces
	var exprList func(es []ast.Expr)
	var bindingList func(bs []*ast.Binding)
	var attrs func(f *ast.FuncBody)
	var funcBody func(f *ast.FuncBody)
	var funcName func(n *ast.FuncName)

	// expressions
	var expr func(e ast.Expr)
	var number func(e *ast.Number)
	var prefix func(e ast.Expr)
	var field func(e *ast.Field)
	var index func(e *ast.Index)
	var call func(e *ast.Call)
	var methodCall func(e *ast.MethodCall)
	var paren func(e *ast.Paren)
	var funcExpr func(e *ast.Func)
	var table func(t *ast.Table)
	var item func(it *ast.TableItem)
	var ifExpr func(e *ast.IfExpr)
	var interp func(e *ast.Interp)

	// operators
	var unary func(e *ast.Unary)
	var binary func(e *ast.Binary)
	var operand func(e ast.Expr, wrap bool)

	// write appends s, emitting the indent first if at the start of a line.
	// In minify mode it trims edge spaces and adds one only where needSpace says.
	write = func(s string) {
		if s == "" {
			return
		}
		if lineStart {
			out.WriteString(strings.Repeat(cfg.Indent, depth))
			lineStart = false
		}
		out.WriteString(s)
	}
	// newline ends the current line. Does nothing in minify mode.
	newline = func() {
		out.WriteString("\n")
		lineStart = true
	}
	indent = func() {
		depth++
	}
	dedent = func() {
		depth--
	}
	// list prints n items with sep between them.
	list = func(n int, sep string, item func(i int)) {
		for i := range n {
			if i > 0 {
				write(sep)
			}
			item(i)
		}
	}
	// needSpace reports whether gluing a and b would turn two tokens into one,
	// or into a comment.
	needSpace = func(a, b byte) bool {
		panic("TODO")
	}

	// blockStmts prints each statement on its own line, adding ";" when the
	// next statement starts with "(".
	blockStmts = func(b *ast.Block) {
		if b == nil {
			return
		}
		for _, s := range b.Stmts {
			stmt(s)
			newline()
		}
	}
	// body prints a nested block one level deeper, leaving the cursor on a
	// fresh line at the current level, ready for "end" / "else" / "until".
	body = func(b *ast.Block) {
		newline()
		indent()
		blockStmts(b)
		dedent()
	}

	// stmt hands each statement to its own function. Unknown types panic.
	stmt = func(s ast.Stmt) {
		switch s := s.(type) {
		case *ast.CallStmt:
			callStmt(s)
		default:
			panic(fmt.Sprintf("printer: unknown statement %T", s))
		}
	}
	local = func(s *ast.Local) {
		write("local ")
		bindingList(s.Names)
		if len(s.Values) != 0 {
			write("=")
		}
		exprList(s.Values)
	}
	localFunc = func(s *ast.LocalFunc) {
	}
	funcDecl = func(s *ast.FuncDecl) {
	}
	assign = func(s *ast.Assign) {
		exprList(s.Targets)
		write("=")
		exprList(s.Values)
	}
	compoundAssign = func(s *ast.CompoundAssign) {
		expr(s.Target)
		write(binaryText[s.Op] + "=")
		expr(s.Value)
	}
	callStmt = func(s *ast.CallStmt) {
		expr(s.Call)
	}
	doStmt = func(s *ast.Do) {
		write("do")
		body(s.Body)
		write("end")
	}
	whileStmt = func(s *ast.While) {
	}
	repeatStmt = func(s *ast.Repeat) {
	}
	ifStmt = func(s *ast.If) {
	}
	numericFor = func(s *ast.NumericFor) {
	}
	genericFor = func(s *ast.GenericFor) {
	}
	returnStmt = func(s *ast.Return) {
		write("return")
		if len(s.Values) != 0 {
			write(" ")
			exprList(s.Values)
		}
	}
	breakStmt = func(s *ast.Break) {
		write("break")
	}
	continueStmt = func(s *ast.Continue) {
		write("continue")
	}

	exprList = func(es []ast.Expr) {
		list(len(es), ", ", func(i int) {
			expr(es[i])
		})
	}
	bindingList = func(bs []*ast.Binding) {
		list(len(bs), ", ", func(i int) {
			write(bs[i].Name.Name)
		})
	}
	// attrs prints "@native @checked " before the function keyword.
	attrs = func(f *ast.FuncBody) {
	}
	// funcBody prints everything after the name: "(a, b, ...)" body "end".
	funcBody = func(f *ast.FuncBody) {
	}
	// funcName prints a.b.c:d.
	funcName = func(n *ast.FuncName) {
	}

	// expr hands each expression to its own function. Unknown types panic.
	expr = func(e ast.Expr) {
		switch e := e.(type) {
		case *ast.Nil:
			write("nil")
		case *ast.Bool:
			if e.Value {
				write("true")
			} else {
				write("false")
			}
		case *ast.Number:
			number(e)
		case *ast.String:
			write(quote(e.Value))
		case *ast.Vararg:
			write("...")
		case *ast.Ident:
			write(e.Name)
		case *ast.Field:
			field(e)
		case *ast.Index:
			index(e)
		case *ast.Call:
			call(e)
		case *ast.MethodCall:
			methodCall(e)
		default:
			panic(fmt.Sprintf("printer: unknown expression %T", e))
		}
	}
	// number writes Raw and remembers that a number was the last token.
	number = func(e *ast.Number) {
		write(e.Raw)
	}
	// prefix prints e, wrapped in parens unless it's already a prefix expression.
	prefix = func(e ast.Expr) {
		switch e.(type) {
		case *ast.Ident, *ast.Field, *ast.Index, *ast.Call, *ast.MethodCall, *ast.Paren:
			expr(e)
		default:
			write("(")
			expr(e)
			write(")")
		}
	}
	field = func(e *ast.Field) {
		prefix(e.X)
		write("." + e.Name.Name)
	}
	index = func(e *ast.Index) {
		prefix(e.X)
		write("[")
		expr(e.Key)
		write("]")
	}
	call = func(e *ast.Call) {
		prefix(e.Fn)
		write("(")
		exprList(e.Args)
		write(")")
	}
	methodCall = func(e *ast.MethodCall) {
		prefix(e.Recv)
		write(":" + e.Method.Name)
		write("(")
		exprList(e.Args)
		write(")")
	}
	paren = func(e *ast.Paren) {
	}
	funcExpr = func(e *ast.Func) {
	}
	table = func(t *ast.Table) {
	}
	item = func(it *ast.TableItem) {
	}
	ifExpr = func(e *ast.IfExpr) {
	}
	interp = func(e *ast.Interp) {
	}

	unary = func(e *ast.Unary) {
		write(unaryText[e.Op])
		u, isUnary := e.(*ast.Unary)
		if e.Op == ast.OpNeg && isUnary && u.Op == ast.OpNeg {
			write(" ")
		}
		l, _ := power(e.X)
		operand(e.X, l <= unaryPrec)
	}
	binary = func(e *ast.Binary) {
		p := binaryPrec[e.Op]
		_, lr := power(e.L)
		operand(e.L, lr < p.left)
		write(" " + binaryText[e.Op] + "")
		rl, _ := power(e.R)
		operand(e.R, rl <= p.right)
	}
	// operand prints e, wrapped in parens if wrap is true.
	operand = func(e ast.Expr, wrap bool) {
		if wrap {
			write("(")
			expr(e)
			write(")")
		} else {
			expr(e)
		}
	}

	// Go rejects unused local variables. Delete names from this line as you
	// wire them up, and the whole line once everything is used.
	_, _, _, _, _, _ = depth, lineStart, last, lastNumber, cfg, out
	_, _, _, _, _, _ = write, newline, indent, dedent, list, needSpace
	_, _, _, _, _, _, _ = body, stmt, local, localFunc, funcDecl, assign, compoundAssign
	_, _, _, _, _, _, _, _, _ = doStmt, whileStmt, repeatStmt, ifStmt, numericFor, genericFor, returnStmt, breakStmt, continueStmt
	_, _, _, _, _ = exprList, bindingList, attrs, funcBody, funcName
	_, _, _, _, _, _, _, _, _, _, _, _, _ = expr, number, prefix, field, index, call, methodCall, paren, funcExpr, table, item, ifExpr, interp
	_, _, _ = unary, binary, operand

	blockStmts(file)
	return out.String()
}
