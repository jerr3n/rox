package ast

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Tree dumper
//
// Prints one node per line, with children indented under their parent:
//
//	Local
//	  names: a, b
//	  values:
//	    Number 1
//	    Binary +
//	      Ident x
//	      Number 2
// ---------------------------------------------------------------------------

var binaryNames = map[BinaryOp]string{
	OpAdd: "+", OpSub: "-", OpMul: "*", OpDiv: "/",
	OpFloorDiv: "//", OpMod: "%", OpPow: "^", OpConcat: "..",
	OpEq: "==", OpNe: "~=", OpLt: "<", OpLe: "<=",
	OpGt: ">", OpGe: ">=", OpAnd: "and", OpOr: "or",
}

var unaryNames = map[UnaryOp]string{
	OpNeg: "-", OpNot: "not", OpLen: "#",
}

var itemNames = map[TableItemKind]string{
	ItemPositional: "positional", ItemNamed: "named", ItemKeyed: "keyed",
}

type dumper struct {
	b strings.Builder
}

// Dump prints any node and everything under it: a whole *Block, one
// statement, one expression or a *FuncBody.
func Dump(n Node) string {
	var p dumper
	// The concrete types come before Stmt and Expr, which are interfaces and
	// would match more than we want.
	switch n := n.(type) {
	case *Block:
		p.block(n, 0)
	case *FuncBody:
		p.funcBody(n, 0)
	case Stmt:
		p.stmt(n, 0)
	case Expr:
		p.expr(n, 0)
	case nil:
		p.line(0, "<nil node>")
	default:
		p.line(0, "<can't dump %T>", n)
	}
	return p.b.String()
}

func (p *dumper) line(depth int, format string, args ...any) {
	p.b.WriteString(strings.Repeat("  ", depth))
	fmt.Fprintf(&p.b, format, args...)
	p.b.WriteByte('\n')
}

// labeled prints "label:" and then each expression under it.
func (p *dumper) labeled(depth int, label string, exprs []Expr) {
	if len(exprs) == 0 {
		return
	}
	p.line(depth, "%s:", label)
	for _, e := range exprs {
		p.expr(e, depth+1)
	}
}

// pathName spells out a callee like a.b.c as one string. It returns "" for
// anything that isn't a plain chain of names, like f() or (x or y).
func pathName(e Expr) string {
	switch e := e.(type) {
	case *Ident:
		return e.Name
	case *Field:
		if x := pathName(e.X); x != "" {
			return x + "." + e.Name.Name
		}
	}
	return ""
}

// nameSuffix is " a.b"+rest when the callee has a pathName, or just " "+rest
// for a method call on something without one, so the line reads
// "Call t.insert", "MethodCall obj:Destroy" or "MethodCall :Destroy".
func nameSuffix(callee Expr, rest string) string {
	name := pathName(callee) + rest
	if name == "" {
		return ""
	}
	return " " + name
}

func bindingNames(bs []*Binding) string {
	names := []string{}
	for _, b := range bs {
		if b == nil || b.Name == nil {
			names = append(names, "<missing name>")
			continue
		}
		names = append(names, b.Name.Name)
	}
	return strings.Join(names, ", ")
}

func (p *dumper) block(b *Block, depth int) {
	if b == nil {
		p.line(depth, "<nil block>")
		return
	}
	p.line(depth, "Block")
	for _, s := range b.Stmts {
		p.stmt(s, depth+1)
	}
}

func (p *dumper) funcBody(f *FuncBody, depth int) {
	if f == nil {
		p.line(depth, "<nil func body>")
		return
	}
	params := bindingNames(f.Params)
	if f.Vararg {
		if params != "" {
			params += ", "
		}
		params += "..."
	}
	p.line(depth, "params: (%s)", params)
	p.block(f.Body, depth)
}

func (p *dumper) stmt(s Stmt, depth int) {
	switch s := s.(type) {
	case *Local:
		p.line(depth, "Local")
		p.line(depth+1, "names: %s", bindingNames(s.Names))
		p.labeled(depth+1, "values", s.Values)
	case *LocalFunc:
		p.line(depth, "LocalFunc %s", s.Name.Name)
		p.funcBody(s.Body, depth+1)
	case *FuncDecl:
		path := []string{}
		for _, id := range s.Name.Path {
			path = append(path, id.Name)
		}
		name := strings.Join(path, ".")
		if s.Name.Method != nil {
			name += ":" + s.Name.Method.Name
		}
		p.line(depth, "FuncDecl %s", name)
		p.funcBody(s.Body, depth+1)
	case *Assign:
		p.line(depth, "Assign")
		p.labeled(depth+1, "targets", s.Targets)
		p.labeled(depth+1, "values", s.Values)
	case *CompoundAssign:
		p.line(depth, "CompoundAssign %s=", binaryNames[s.Op])
		p.expr(s.Target, depth+1)
		p.expr(s.Value, depth+1)
	case *CallStmt:
		p.line(depth, "CallStmt")
		p.expr(s.Call, depth+1)
	case *Do:
		p.line(depth, "Do")
		p.block(s.Body, depth+1)
	case *While:
		p.line(depth, "While")
		p.expr(s.Cond, depth+1)
		p.block(s.Body, depth+1)
	case *Repeat:
		p.line(depth, "Repeat")
		p.block(s.Body, depth+1)
		p.line(depth+1, "until:")
		p.expr(s.Cond, depth+2)
	case *If:
		p.line(depth, "If")
		p.expr(s.Cond, depth+1)
		p.block(s.Then, depth+1)
		for _, ei := range s.ElseIfs {
			p.line(depth+1, "ElseIf")
			p.expr(ei.Cond, depth+2)
			p.block(ei.Body, depth+2)
		}
		if s.Else != nil {
			p.line(depth+1, "Else")
			p.block(s.Else, depth+2)
		}
	case *NumericFor:
		p.line(depth, "NumericFor %s", bindingNames([]*Binding{s.Var}))
		p.line(depth+1, "start:")
		p.expr(s.Start, depth+2)
		p.line(depth+1, "stop:")
		p.expr(s.Stop, depth+2)
		if s.Step != nil {
			p.line(depth+1, "step:")
			p.expr(s.Step, depth+2)
		}
		p.block(s.Body, depth+1)
	case *GenericFor:
		p.line(depth, "GenericFor %s", bindingNames(s.Vars))
		p.labeled(depth+1, "in", s.Exprs)
		p.block(s.Body, depth+1)
	case *Return:
		p.line(depth, "Return")
		for _, v := range s.Values {
			p.expr(v, depth+1)
		}
	case *Break:
		p.line(depth, "Break")
	case *Continue:
		p.line(depth, "Continue")
	case nil:
		p.line(depth, "<nil stmt>")
	default:
		p.line(depth, "<unknown stmt %T>", s)
	}
}

func (p *dumper) expr(e Expr, depth int) {
	switch e := e.(type) {
	case *Nil:
		p.line(depth, "Nil")
	case *Bool:
		p.line(depth, "Bool %v", e.Value)
	case *Number:
		p.line(depth, "Number %s", e.Raw)
	case *String:
		p.line(depth, "String %q", e.Value)
	case *Vararg:
		p.line(depth, "Vararg")
	case *Ident:
		p.line(depth, "Ident %s", e.Name)
	case *Field:
		p.line(depth, "Field .%s", e.Name.Name)
		p.expr(e.X, depth+1)
	case *Index:
		p.line(depth, "Index")
		p.expr(e.X, depth+1)
		p.expr(e.Key, depth+1)
	case *Call:
		p.line(depth, "Call%s", nameSuffix(e.Fn, ""))
		p.expr(e.Fn, depth+1)
		p.labeled(depth+1, "args", e.Args)
	case *MethodCall:
		p.line(depth, "MethodCall%s", nameSuffix(e.Recv, ":"+e.Method.Name))
		p.expr(e.Recv, depth+1)
		p.labeled(depth+1, "args", e.Args)
	case *Unary:
		p.line(depth, "Unary %s", unaryNames[e.Op])
		p.expr(e.X, depth+1)
	case *Binary:
		p.line(depth, "Binary %s", binaryNames[e.Op])
		p.expr(e.L, depth+1)
		p.expr(e.R, depth+1)
	case *Paren:
		p.line(depth, "Paren")
		p.expr(e.X, depth+1)
	case *Func:
		p.line(depth, "Func")
		p.funcBody(e.Body, depth+1)
	case *Table:
		p.line(depth, "Table")
		for _, item := range e.Items {
			switch item.Kind {
			case ItemNamed:
				p.line(depth+1, "%s %s =", itemNames[item.Kind], item.Name.Name)
			default:
				p.line(depth+1, "%s", itemNames[item.Kind])
			}
			if item.Kind == ItemKeyed {
				p.expr(item.Key, depth+2)
			}
			p.expr(item.Value, depth+2)
		}
	case *IfExpr:
		p.line(depth, "IfExpr")
		p.expr(e.Cond, depth+1)
		p.expr(e.Then, depth+1)
		for _, ei := range e.ElseIfs {
			p.line(depth+1, "ElseIf")
			p.expr(ei.Cond, depth+2)
			p.expr(ei.Then, depth+2)
		}
		p.line(depth+1, "Else")
		p.expr(e.Else, depth+2)
	case nil:
		p.line(depth, "<nil expr>")
	default:
		p.line(depth, "<unknown expr %T>", e)
	}
}
