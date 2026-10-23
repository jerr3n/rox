package ast

// ---------------------------------------------------------------------------
// Rewriting
//
// Rewrite and RewriteBlock walk a tree like Inspect, but they can replace what
// they visit. A node's children are rewritten first, then the node itself is
// handed to the callback, and whatever the callback returns takes its place:
//
//	RewriteBlock(block, func(e Expr) Expr {
//		if n, ok := e.(*Number); ok && n.Raw == "1" {
//			return &Number{Raw: "2"}
//		}
//		return e
//	}, nil)
//
// Names that aren't expressions (the .name in x.name, a:method, bindings,
// function names) are never handed to the callbacks.
// ---------------------------------------------------------------------------

// Rewrite rewrites e's children, then returns callback(e). Expressions inside
// function bodies are rewritten too; the statements themselves are kept.
func Rewrite(e Expr, callback func(expr Expr) Expr) Expr {
	r := rewriter{expr: callback}
	return r.rewrite(e)
}

// RewriteBlock rewrites b in place: every expression with expr, then every
// statement with stmt. stmt returns what the statement becomes: nothing
// deletes it, several replace it with all of them. Either callback may be nil.
func RewriteBlock(b *Block, expr func(Expr) Expr, stmt func(Stmt) []Stmt) {
	r := rewriter{expr: expr, stmt: stmt}
	r.block(b)
}

type rewriter struct {
	expr func(Expr) Expr
	stmt func(Stmt) []Stmt
}

func (r *rewriter) rewrite(e Expr) Expr {
	if e == nil {
		return nil
	}
	// Leaves (Nil, Bool, Number, String, Vararg, Ident) have no children, so
	// they have no case.
	switch e := e.(type) {
	case *Interp:
		r.exprs(e.Exprs)
	case *Field:
		e.X = r.rewrite(e.X)
	case *Index:
		e.X = r.rewrite(e.X)
		e.Key = r.rewrite(e.Key)
	case *Call:
		e.Fn = r.rewrite(e.Fn)
		r.exprs(e.Args)
	case *MethodCall:
		e.Recv = r.rewrite(e.Recv)
		r.exprs(e.Args)
	case *Unary:
		e.X = r.rewrite(e.X)
	case *Binary:
		e.L = r.rewrite(e.L)
		e.R = r.rewrite(e.R)
	case *Paren:
		e.X = r.rewrite(e.X)
	case *Func:
		r.funcBody(e.Body)
	case *Table:
		for _, item := range e.Items {
			item.Key = r.rewrite(item.Key) // nil unless ItemKeyed
			item.Value = r.rewrite(item.Value)
		}
	case *IfExpr:
		e.Cond = r.rewrite(e.Cond)
		e.Then = r.rewrite(e.Then)
		for _, ei := range e.ElseIfs {
			ei.Cond = r.rewrite(ei.Cond)
			ei.Then = r.rewrite(ei.Then)
		}
		e.Else = r.rewrite(e.Else)
	}
	if r.expr == nil {
		return e
	}
	return r.expr(e)
}

// exprs rewrites es in place.
func (r *rewriter) exprs(es []Expr) {
	for i := range es {
		es[i] = r.rewrite(es[i])
	}
}

// block rewrites each statement's children, then the statement itself, and
// collects what stmt returns into a new Stmts.
func (r *rewriter) block(b *Block) {
	if b == nil {
		return
	}
	out := make([]Stmt, 0, len(b.Stmts))
	for _, s := range b.Stmts {
		r.children(s)
		if r.stmt == nil {
			out = append(out, s)
		} else {
			out = append(out, r.stmt(s)...)
		}
	}
	b.Stmts = out
}

func (r *rewriter) children(s Stmt) {
	switch s := s.(type) {
	case *Local:
		r.exprs(s.Values)
	case *LocalFunc:
		r.funcBody(s.Body)
	case *FuncDecl:
		r.funcBody(s.Body)
	case *Assign:
		r.exprs(s.Targets)
		r.exprs(s.Values)
	case *CompoundAssign:
		s.Target = r.rewrite(s.Target)
		s.Value = r.rewrite(s.Value)
	case *CallStmt:
		s.Call = r.rewrite(s.Call)
	case *Do:
		r.block(s.Body)
	case *While:
		s.Cond = r.rewrite(s.Cond)
		r.block(s.Body)
	case *Repeat:
		r.block(s.Body)
		s.Cond = r.rewrite(s.Cond)
	case *If:
		s.Cond = r.rewrite(s.Cond)
		r.block(s.Then)
		for _, ei := range s.ElseIfs {
			ei.Cond = r.rewrite(ei.Cond)
			r.block(ei.Body)
		}
		r.block(s.Else) // nil if there's no else
	case *NumericFor:
		s.Start = r.rewrite(s.Start)
		s.Stop = r.rewrite(s.Stop)
		s.Step = r.rewrite(s.Step) // nil if omitted
		r.block(s.Body)
	case *GenericFor:
		r.exprs(s.Exprs)
		r.block(s.Body)
	case *Return:
		r.exprs(s.Values)
	}
}

func (r *rewriter) funcBody(f *FuncBody) {
	if f != nil {
		r.block(f.Body)
	}
}
