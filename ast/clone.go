package ast

import (
	"fmt"
	"slices"
)

// ---------------------------------------------------------------------------
// Cloning
//
// CloneBlock, CloneStmt and CloneExpr make deep copies. The copy shares no
// nodes, slices or idents with the original, so either one can be rewritten
// without touching the other.
//
// ident, if not nil, is called for every Ident that is copied, variable or
// not, with the original and its copy. It may change the copy, e.g. to rename
// it:
//
//	CloneBlock(body, func(orig, copy *Ident) {
//		if sym := info.Symbols[orig]; internal[sym] {
//			copy.Name = names[sym]
//		}
//	})
//
// Look things up by orig: a scope.Info only knows the original idents.
// Non-variable idents (the .name in x.name, a:method, ...) aren't in
// info.Symbols, so a lookup like the one above skips them.
// ---------------------------------------------------------------------------

// CloneBlock returns a deep copy of b.
func CloneBlock(b *Block, ident func(orig, copy *Ident)) *Block {
	c := cloner{ident: ident}
	return c.block(b)
}

// CloneStmt returns a deep copy of s.
func CloneStmt(s Stmt, ident func(orig, copy *Ident)) Stmt {
	c := cloner{ident: ident}
	return c.stmt(s)
}

// CloneExpr returns a deep copy of e.
func CloneExpr(e Expr, ident func(orig, copy *Ident)) Expr {
	c := cloner{ident: ident}
	return c.expr(e)
}

type cloner struct {
	ident func(orig, copy *Ident)
}

// Each case copies the node by value, which takes care of Span, Op, Name and
// the other plain fields, then replaces every pointer, interface and slice
// field with a copy of its own.

func (c *cloner) expr(e Expr) Expr {
	// a nil Expr has to come back as an untyped nil, not a nil *Something,
	// or `x != nil` checks on optional fields stop working
	if e == nil {
		return nil
	}
	switch e := e.(type) {
	case *Nil:
		n := *e
		return &n
	case *Bool:
		n := *e
		return &n
	case *Number:
		n := *e
		return &n
	case *String:
		n := *e
		return &n
	case *Vararg:
		n := *e
		return &n
	case *Ident:
		return c.identNode(e)
	case *Field:
		n := *e
		n.X = c.expr(e.X)
		n.Name = c.identNode(e.Name)
		return &n
	case *Index:
		n := *e
		n.X = c.expr(e.X)
		n.Key = c.expr(e.Key)
		return &n
	case *Call:
		n := *e
		n.Fn = c.expr(e.Fn)
		n.Args = cloneAll(e.Args, c.expr)
		return &n
	case *MethodCall:
		n := *e
		n.Recv = c.expr(e.Recv)
		n.Method = c.identNode(e.Method)
		n.Args = cloneAll(e.Args, c.expr)
		return &n
	case *Unary:
		n := *e
		n.X = c.expr(e.X)
		return &n
	case *Binary:
		n := *e
		n.L = c.expr(e.L)
		n.R = c.expr(e.R)
		return &n
	case *Paren:
		n := *e
		n.X = c.expr(e.X)
		return &n
	case *Func:
		n := *e
		n.Body = c.funcBody(e.Body)
		return &n
	case *Table:
		n := *e
		n.Items = cloneAll(e.Items, c.tableItem)
		return &n
	case *IfExpr:
		n := *e
		n.Cond = c.expr(e.Cond)
		n.Then = c.expr(e.Then)
		n.ElseIfs = cloneAll(e.ElseIfs, c.elseIfExpr)
		n.Else = c.expr(e.Else)
		return &n
	case *Interp:
		n := *e
		n.Parts = slices.Clone(e.Parts) // strings are immutable, the slice isn't
		n.Exprs = cloneAll(e.Exprs, c.expr)
		return &n
	default:
		panic(fmt.Sprintf("ast: clone: unhandled expression %T", e))
	}
}

func (c *cloner) stmt(s Stmt) Stmt {
	if s == nil {
		return nil
	}
	switch s := s.(type) {
	case *Local:
		n := *s
		n.Names = cloneAll(s.Names, c.binding)
		n.Values = cloneAll(s.Values, c.expr)
		return &n
	case *LocalFunc:
		n := *s
		n.Name = c.identNode(s.Name)
		n.Body = c.funcBody(s.Body)
		return &n
	case *FuncDecl:
		n := *s
		n.Name = c.funcName(s.Name)
		n.Body = c.funcBody(s.Body)
		return &n
	case *Assign:
		n := *s
		n.Targets = cloneAll(s.Targets, c.expr)
		n.Values = cloneAll(s.Values, c.expr)
		return &n
	case *CompoundAssign:
		n := *s
		n.Target = c.expr(s.Target)
		n.Value = c.expr(s.Value)
		return &n
	case *CallStmt:
		n := *s
		n.Call = c.expr(s.Call)
		return &n
	case *Do:
		n := *s
		n.Body = c.block(s.Body)
		return &n
	case *While:
		n := *s
		n.Cond = c.expr(s.Cond)
		n.Body = c.block(s.Body)
		return &n
	case *Repeat:
		n := *s
		n.Body = c.block(s.Body)
		n.Cond = c.expr(s.Cond)
		return &n
	case *If:
		n := *s
		n.Cond = c.expr(s.Cond)
		n.Then = c.block(s.Then)
		n.ElseIfs = cloneAll(s.ElseIfs, c.elseIf)
		n.Else = c.block(s.Else) // nil if there's no else
		return &n
	case *NumericFor:
		n := *s
		n.Var = c.binding(s.Var)
		n.Start = c.expr(s.Start)
		n.Stop = c.expr(s.Stop)
		n.Step = c.expr(s.Step) // nil if omitted
		n.Body = c.block(s.Body)
		return &n
	case *GenericFor:
		n := *s
		n.Vars = cloneAll(s.Vars, c.binding)
		n.Exprs = cloneAll(s.Exprs, c.expr)
		n.Body = c.block(s.Body)
		return &n
	case *Return:
		n := *s
		n.Values = cloneAll(s.Values, c.expr)
		return &n
	case *Break:
		n := *s
		return &n
	case *Continue:
		n := *s
		return &n
	default:
		panic(fmt.Sprintf("ast: clone: unhandled statement %T", s))
	}
}

// The support structs have concrete pointer types, so returning a nil one is
// fine here.

func (c *cloner) block(b *Block) *Block {
	if b == nil {
		return nil
	}
	n := *b
	n.Stmts = cloneAll(b.Stmts, c.stmt)
	return &n
}

// identNode copies id and hands both to the ident callback.
func (c *cloner) identNode(id *Ident) *Ident {
	if id == nil {
		return nil
	}
	n := *id
	if c.ident != nil {
		c.ident(id, &n)
	}
	return &n
}

func (c *cloner) binding(b *Binding) *Binding {
	if b == nil {
		return nil
	}
	n := *b
	n.Name = c.identNode(b.Name)
	return &n
}

func (c *cloner) funcBody(f *FuncBody) *FuncBody {
	if f == nil {
		return nil
	}
	n := *f
	n.Attributes = cloneAll(f.Attributes, c.attribute)
	n.Params = cloneAll(f.Params, c.binding)
	n.Body = c.block(f.Body)
	return &n
}

func (c *cloner) funcName(f *FuncName) *FuncName {
	if f == nil {
		return nil
	}
	n := *f
	n.Path = cloneAll(f.Path, c.identNode)
	n.Method = c.identNode(f.Method)
	return &n
}

func (c *cloner) attribute(a *Attribute) *Attribute {
	if a == nil {
		return nil
	}
	n := *a
	return &n
}

func (c *cloner) tableItem(t *TableItem) *TableItem {
	if t == nil {
		return nil
	}
	n := *t
	n.Name = c.identNode(t.Name) // ItemNamed only
	n.Key = c.expr(t.Key)        // ItemKeyed only
	n.Value = c.expr(t.Value)
	return &n
}

func (c *cloner) elseIf(e *ElseIf) *ElseIf {
	if e == nil {
		return nil
	}
	n := *e
	n.Cond = c.expr(e.Cond)
	n.Body = c.block(e.Body)
	return &n
}

func (c *cloner) elseIfExpr(e *ElseIfExpr) *ElseIfExpr {
	if e == nil {
		return nil
	}
	n := *e
	n.Cond = c.expr(e.Cond)
	n.Then = c.expr(e.Then)
	return &n
}

// cloneAll returns a new slice holding f of each element of s. A nil s stays
// nil, so the copy compares equal to the original under reflect.DeepEqual.
func cloneAll[T any](s []T, f func(T) T) []T {
	if s == nil {
		return nil
	}
	out := make([]T, len(s))
	for i, v := range s {
		out[i] = f(v)
	}
	return out
}
