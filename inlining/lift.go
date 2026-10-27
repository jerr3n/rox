package inlining

import (
	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/scope"
)

const wantAll = -1

type lifter struct {
	ix      *scope.Index
	cands   map[*scope.Symbol]*candidate
	prelude []ast.Stmt
	stopped bool
	changed bool
}

func liftStmt(ix *scope.Index, cands map[*scope.Symbol]*candidate, st ast.Stmt) ([]ast.Stmt, bool) {
	l := lifter{ix: ix, cands: cands}
	out := l.stmt(st)
	return append(l.prelude, out...), l.changed
}

func (l *lifter) stmt(st ast.Stmt) []ast.Stmt {
	switch t := st.(type) {
	case *ast.Local:
		l.exprs(t.Values, lastWant(len(t.Names), len(t.Values)))
	case *ast.Assign:
		for _, target := range t.Targets {
			l.skip(target)
		}
		l.exprs(t.Values, lastWant(len(t.Targets), len(t.Values)))
	case *ast.CompoundAssign:
		l.skip(t.Target)
		t.Value = l.expr(t.Value, 1)
	case *ast.CallStmt:
		if call, ok := t.Call.(*ast.Call); ok {
			if l.call(call, 0) == nil {
				return nil
			}
		} else {
			t.Call = l.expr(t.Call, 1)
		}
	case *ast.Return:
		l.exprs(t.Values, wantAll)
	case *ast.If:
		t.Cond = l.expr(t.Cond, 1)
	case *ast.NumericFor:
		t.Start = l.expr(t.Start, 1)
		t.Stop = l.expr(t.Stop, 1)
		if t.Step != nil {
			t.Step = l.expr(t.Step, 1)
		}
	case *ast.GenericFor:
		l.exprs(t.Exprs, wantAll)
	}
	return []ast.Stmt{st}
}

func (l *lifter) expr(e ast.Expr, want int) ast.Expr {
	switch t := e.(type) {
	case *ast.Call:
		return l.call(t, want)
	case *ast.MethodCall:
		t.Recv = l.expr(t.Recv, 1)
		l.exprs(t.Args, wantAll)
		l.stopped = true
	case *ast.Binary:
		t.L = l.expr(t.L, 1)
		if t.Op == ast.OpAnd || t.Op == ast.OpOr {
			l.skip(t.R)
		} else {
			t.R = l.expr(t.R, 1)
		}
	case *ast.Unary:
		t.X = l.expr(t.X, 1)
	case *ast.Paren:
		t.X = l.expr(t.X, 1)
	case *ast.Field:
		t.X = l.expr(t.X, 1)
	case *ast.Index:
		t.X = l.expr(t.X, 1)
		t.Key = l.expr(t.Key, 1)
	case *ast.Interp:
		l.exprs(t.Exprs, 1)
	case *ast.Table:
		for i, item := range t.Items {
			if item.Key != nil {
				item.Key = l.expr(item.Key, 1)
			}
			w := 1
			if i == len(t.Items)-1 && item.Kind == ast.ItemPositional {
				w = wantAll
			}
			item.Value = l.expr(item.Value, w)
		}
	case *ast.IfExpr:
		t.Cond = l.expr(t.Cond, 1)
		l.skip(t.Then)
		for _, ei := range t.ElseIfs {
			l.skip(ei.Cond)
			l.skip(ei.Then)
		}
		l.skip(t.Else)
	}
	return e
}

func (l *lifter) exprs(es []ast.Expr, lastWant int) {
	for i := range es {
		w := 1
		if i == len(es)-1 {
			w = lastWant
		}
		es[i] = l.expr(es[i], w)
	}
}

func (l *lifter) call(call *ast.Call, want int) ast.Expr {
	saved := l.stopped
	call.Fn = l.expr(call.Fn, 1)
	l.exprs(call.Args, wantAll)

	c := l.callee(call)
	if c != nil && !saved && canExpand(c, want) {
		n := want
		if n == wantAll {
			n = 1
		}
		exp, ok := expand(l.ix, c, call, n)
		if ok {
			l.prelude = append(l.prelude, exp.stmts...)
			l.changed = true
			l.stopped = saved
			if n == 0 {
				return nil
			}
			return exp.returns[0]
		}
	}
	l.stopped = true
	return call
}

func (l *lifter) skip(e ast.Expr) {
	if e != nil && hasCall(e) {
		l.stopped = true
	}
}

func (l *lifter) callee(call *ast.Call) *candidate {
	id, ok := call.Fn.(*ast.Ident)
	if !ok {
		return nil
	}
	sym := l.ix.Info.Symbols[id]
	if sym == nil {
		return nil
	}
	return l.cands[sym]
}

func canExpand(c *candidate, want int) bool {
	if want != wantAll {
		return true
	}
	return c.Shape == returnsTrailing && c.Returns == 1
}

func hasCall(e ast.Expr) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		if found {
			return false
		}
		switch n.(type) {
		case *ast.Func:
			return false
		case *ast.Call, *ast.MethodCall:
			found = true
			return false
		}
		return true
	})
	return found
}

func lastWant(names, values int) int {
	if names > values {
		return wantAll
	}
	return 1
}
