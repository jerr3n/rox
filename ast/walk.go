package ast

// ---------------------------------------------------------------------------
// Walking and matching
//
// Inspect visits every node in a tree, and IsPath / CallTo recognise common
// shapes. Together they find a pattern anywhere in the tree:
//
//	Inspect(block, func(n Node) bool {
//		if call, ok := CallTo(n, "math.random"); ok {
//			// found one; call.Args holds the arguments
//		}
//		return true
//	})
// ---------------------------------------------------------------------------

// Inspect calls f on n, then on each of n's children, depth first in source
// order. If f returns false, n's children are skipped.
//
// f may change the node it is given before returning, e.g. replace entries in
// a *Block's Stmts. Inspect then walks the new children, not the old ones.
func Inspect(n Node, f func(Node) bool) {
	if n == nil || !f(n) {
		return
	}
	// Leaves (Nil, Bool, Number, String, Vararg, Ident, Attribute) have no
	// children, so they have no case.
	switch n := n.(type) {
	// Expressions
	case *Interp:
		inspectExprs(n.Exprs, f)
	case *Field:
		Inspect(n.X, f)
		Inspect(n.Name, f)
	case *Index:
		Inspect(n.X, f)
		Inspect(n.Key, f)
	case *Call:
		Inspect(n.Fn, f)
		inspectExprs(n.Args, f)
	case *MethodCall:
		Inspect(n.Recv, f)
		Inspect(n.Method, f)
		inspectExprs(n.Args, f)
	case *Unary:
		Inspect(n.X, f)
	case *Binary:
		Inspect(n.L, f)
		Inspect(n.R, f)
	case *Paren:
		Inspect(n.X, f)
	case *Func:
		if n.Body != nil {
			Inspect(n.Body, f)
		}
	case *Table:
		for _, item := range n.Items {
			Inspect(item, f)
		}
	case *IfExpr:
		Inspect(n.Cond, f)
		Inspect(n.Then, f)
		for _, e := range n.ElseIfs {
			Inspect(e, f)
		}
		Inspect(n.Else, f)

	// Statements
	case *Local:
		for _, b := range n.Names {
			Inspect(b, f)
		}
		inspectExprs(n.Values, f)
	case *LocalFunc:
		Inspect(n.Name, f)
		if n.Body != nil {
			Inspect(n.Body, f)
		}
	case *FuncDecl:
		if n.Name != nil {
			Inspect(n.Name, f)
		}
		if n.Body != nil {
			Inspect(n.Body, f)
		}
	case *Assign:
		inspectExprs(n.Targets, f)
		inspectExprs(n.Values, f)
	case *CompoundAssign:
		Inspect(n.Target, f)
		Inspect(n.Value, f)
	case *CallStmt:
		Inspect(n.Call, f)
	case *Do:
		inspectBlock(n.Body, f)
	case *While:
		Inspect(n.Cond, f)
		inspectBlock(n.Body, f)
	case *Repeat:
		inspectBlock(n.Body, f)
		Inspect(n.Cond, f)
	case *If:
		Inspect(n.Cond, f)
		inspectBlock(n.Then, f)
		for _, e := range n.ElseIfs {
			Inspect(e, f)
		}
		inspectBlock(n.Else, f)
	case *NumericFor:
		Inspect(n.Var, f)
		Inspect(n.Start, f)
		Inspect(n.Stop, f)
		Inspect(n.Step, f)
		inspectBlock(n.Body, f)
	case *GenericFor:
		for _, b := range n.Vars {
			Inspect(b, f)
		}
		inspectExprs(n.Exprs, f)
		inspectBlock(n.Body, f)
	case *Return:
		inspectExprs(n.Values, f)

	// Support structs
	case *Block:
		for _, s := range n.Stmts {
			Inspect(s, f)
		}
	case *ElseIf:
		Inspect(n.Cond, f)
		inspectBlock(n.Body, f)
	case *ElseIfExpr:
		Inspect(n.Cond, f)
		Inspect(n.Then, f)
	case *TableItem:
		if n.Name != nil {
			Inspect(n.Name, f)
		}
		Inspect(n.Key, f)
		Inspect(n.Value, f)
	case *Binding:
		if n.Name != nil {
			Inspect(n.Name, f)
		}
	case *FuncBody:
		for _, a := range n.Attributes {
			Inspect(a, f)
		}
		for _, p := range n.Params {
			Inspect(p, f)
		}
		inspectBlock(n.Body, f)
	case *FuncName:
		for _, id := range n.Path {
			Inspect(id, f)
		}
		if n.Method != nil {
			Inspect(n.Method, f)
		}
	}
}

func inspectExprs(exprs []Expr, f func(Node) bool) {
	for _, e := range exprs {
		Inspect(e, f)
	}
}

// inspectBlock skips nil blocks, like an If with no else. Passing a nil
// *Block straight to Inspect would hand f a non-nil Node holding a nil
// pointer.
func inspectBlock(b *Block, f func(Node) bool) {
	if b != nil {
		Inspect(b, f)
	}
}

// IsPath reports whether n is the plain chain of names path, e.g.
// IsPath(n, "math.random") matches the expression `math.random`.
//
// It only looks at names, so it can't tell the global `math` from a local
// that shadows it.
func IsPath(n Node, path string) bool {
	e, ok := n.(Expr)
	return ok && pathName(e) == path
}

// CallTo returns n as a *Call if it calls path, e.g. CallTo(n, "table.insert")
// matches `table.insert(t, v)`. Method calls like `obj:f()` never match.
func CallTo(n Node, path string) (*Call, bool) {
	call, ok := n.(*Call)
	if !ok || !IsPath(call.Fn, path) {
		return nil, false
	}
	return call, true
}
