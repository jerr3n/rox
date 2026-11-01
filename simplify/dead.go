package simplify

import (
	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/scope"
)

// maxRounds caps the fixpoint loop in RemoveDead, like inlining.maxRounds.
const maxRounds = 64

// RemoveDead deletes local variables and local functions that are never read
// and whose values have no side effects. Removing one thing can make another
// dead (a deleted closure was the only reader of an outer local), so it
// repeats until a round removes nothing.
func RemoveDead(block *ast.Block) *ast.Block {
	for range maxRounds {
		if !removeOnce(block) {
			break
		}
	}
	return block
}

// deadSet is the symbols that will be deleted this round.
type deadSet map[*scope.Symbol]bool

// removeOnce resolves block once, picks the dead symbols, and deletes every
// statement that declares or writes them. It reports whether anything changed.
//
// A symbol is only deleted if every place that declares or writes it can go.
// Otherwise a leftover `x = ...` would turn into a write to a global (or to
// an outer variable with the same name) once its declaration is gone.
func removeOnce(block *ast.Block) bool {
	info := scope.Resolve(block)
	owner := owners(block)

	dead := deadSet{}
	for _, sym := range info.Symbols {
		if isDead(sym, owner) {
			dead[sym] = true
		}
	}

	// Dropping one symbol can block another that shares its statement
	// (local a, b = ...), so repeat until the set settles.
	for changed := true; changed; {
		changed = false
		for sym := range dead {
			if !deletable(info, sym, owner, dead) {
				delete(dead, sym)
				changed = true
			}
		}
	}
	if len(dead) == 0 {
		return false
	}

	ast.RewriteBlock(block, nil, func(st ast.Stmt) []ast.Stmt {
		out, ok := plan(info, st, dead)
		if !ok {
			// deletable checked every statement that touches a dead symbol
			panic("simplify: statement with a dead target can't be deleted")
		}
		return out
	})
	return true
}

// owners maps every Ident that a statement declares or assigns directly
// (not t.x or t[k]) to that statement.
func owners(block *ast.Block) map[*ast.Ident]ast.Stmt {
	owner := map[*ast.Ident]ast.Stmt{}
	ast.Inspect(block, func(n ast.Node) bool {
		switch st := n.(type) {
		case *ast.Local:
			for _, b := range st.Names {
				owner[b.Name] = st
			}
		case *ast.LocalFunc:
			owner[st.Name] = st
		case *ast.Assign:
			for _, t := range st.Targets {
				if id, ok := t.(*ast.Ident); ok {
					owner[id] = st
				}
			}
		case *ast.FuncDecl:
			if len(st.Name.Path) == 1 && st.Name.Method == nil {
				owner[st.Name.Path[0]] = st
			}
		}
		return true
	})
	return owner
}

// deletable reports whether every statement that declares or writes sym can
// drop it, given that everything in dead is being dropped too.
func deletable(info *scope.Info, sym *scope.Symbol, owner map[*ast.Ident]ast.Stmt, dead deadSet) bool {
	for _, site := range append([]*ast.Ident{sym.Decl}, sym.Writes...) {
		st := owner[site]
		if st == nil {
			return false
		}
		if _, ok := plan(info, st, dead); !ok {
			return false
		}
	}
	return true
}

// plan returns what st turns into once the symbols in dead are gone:
//   - nil            → st is deleted
//   - []Stmt{st}     → st is kept as it is
//   - something else → st is rewritten, e.g. `local x = g()` → `g()`
//
// ok is false if st has a dead target it can't drop, e.g. `local x = a + b`
// (the + may call a metamethod) or `local a, b = ...` with only a dead.
func plan(info *scope.Info, st ast.Stmt, dead deadSet) (out []ast.Stmt, ok bool) {
	keep := []ast.Stmt{st}
	isDeadIdent := func(e ast.Expr) bool {
		id, isIdent := e.(*ast.Ident)
		return isIdent && dead[info.Symbols[id]]
	}

	switch t := st.(type) {
	case *ast.LocalFunc:
		if dead[info.Symbols[t.Name]] {
			return nil, true
		}
	case *ast.FuncDecl:
		if len(t.Name.Path) == 1 && t.Name.Method == nil && isDeadIdent(t.Name.Path[0]) {
			// a function value has no side effects
			return nil, true
		}
	case *ast.Local:
		n := 0
		for _, b := range t.Names {
			if isDeadIdent(b.Name) {
				n++
			}
		}
		switch n {
		case 0:
			return keep, true
		case len(t.Names):
			return effects(info, t.Values)
		default:
			return nil, false
		}
	case *ast.Assign:
		n := 0
		for _, target := range t.Targets {
			if isDeadIdent(target) {
				n++
			}
		}
		switch {
		case n == 0:
			return keep, true
		case n == len(t.Targets):
			return effects(info, t.Values)
		case len(t.Targets) != len(t.Values):
			return nil, false
		}
		// Partly dead with one value per target: drop the dead pairs, as
		// long as dropping their values changes nothing.
		var targets, values []ast.Expr
		for i, target := range t.Targets {
			if !isDeadIdent(target) {
				targets = append(targets, target)
				values = append(values, t.Values[i])
				continue
			}
			if !pure(info, t.Values[i]) {
				return nil, false
			}
		}
		// a new node, because deletable calls plan before anything is
		// actually being rewritten
		return []ast.Stmt{&ast.Assign{Span: t.Span, Targets: targets, Values: values}}, true
	}
	return keep, true
}

// effects turns the values of a fully dead local or assignment into the
// statements that still have to run: pure values vanish, calls become call
// statements. Anything else (a + b, t.x, ...) might have an effect that
// can't be written as a statement, so it fails.
func effects(info *scope.Info, values []ast.Expr) ([]ast.Stmt, bool) {
	var out []ast.Stmt
	for _, v := range values {
		switch v.(type) {
		case *ast.Call, *ast.MethodCall:
			out = append(out, &ast.CallStmt{Call: v})
			continue
		}
		if !pure(info, v) {
			return nil, false
		}
	}
	return out, true
}

// isDead reports whether sym is a kind this pass may delete and nothing
// reads it. A local function calling itself doesn't count as a read.
func isDead(sym *scope.Symbol, owner map[*ast.Ident]ast.Stmt) bool {
	if sym == nil || sym.Decl == nil || !removableKind(sym.Kind) {
		return false
	}
	if len(sym.Reads) == 0 {
		return true
	}
	fn, ok := owner[sym.Decl].(*ast.LocalFunc)
	if !ok {
		return false
	}
	inside := map[*ast.Ident]bool{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			inside[id] = true
		}
		return true
	})
	for _, r := range sym.Reads {
		if !inside[r] {
			return false
		}
	}
	return true
}

// removableKind reports whether symbols of kind k may be deleted at all.
// Only Local and LocalFunc qualify (not globals, params, for-loop vars, self).
func removableKind(k scope.Kind) bool {
	return k == scope.Local || k == scope.LocalFunc
}

// pure reports whether evaluating e can't do anything observable: no calls,
// no errors, no metamethods. It's conservative; when unsure, it says false.
func pure(info *scope.Info, e ast.Expr) bool {
	switch t := e.(type) {
	case nil:
		return true
	case *ast.Nil, *ast.Bool, *ast.Number, *ast.String, *ast.Vararg, *ast.Func:
		return true
	case *ast.Ident:
		// a global read can go through an environment metamethod
		sym := info.Symbols[t]
		return sym != nil && sym.Kind != scope.Global
	case *ast.Paren:
		return pure(info, t.X)
	case *ast.Table:
		for _, item := range t.Items {
			if item.Kind == ast.ItemKeyed && !safeKey(item.Key) {
				return false
			}
			if !pure(info, item.Value) {
				return false
			}
		}
		return true
	case *ast.IfExpr:
		if !pure(info, t.Cond) || !pure(info, t.Then) || !pure(info, t.Else) {
			return false
		}
		for _, ei := range t.ElseIfs {
			if !pure(info, ei.Cond) || !pure(info, ei.Then) {
				return false
			}
		}
		return true
	case *ast.Unary:
		if t.Op == ast.OpNot {
			return pure(info, t.X)
		}
		return t.Op == ast.OpNeg && numeric(t.X)
	case *ast.Binary:
		if t.Op == ast.OpAnd || t.Op == ast.OpOr {
			return pure(info, t.L) && pure(info, t.R)
		}
		return numeric(t)
	}
	return false
}

// safeKey reports whether e is fine as a [key] in a table constructor.
// [nil] and [0/0] raise an error, so only plain literals are allowed.
func safeKey(e ast.Expr) bool {
	switch e.(type) {
	case *ast.String, *ast.Bool, *ast.Number:
		return true
	}
	return false
}

// numeric reports whether e is arithmetic on number literals only, like
// 1 / 0 or -1. Arithmetic on plain numbers never errors in Luau.
func numeric(e ast.Expr) bool {
	switch t := e.(type) {
	case *ast.Number:
		return true
	case *ast.Paren:
		return numeric(t.X)
	case *ast.Unary:
		return t.Op == ast.OpNeg && numeric(t.X)
	case *ast.Binary:
		if t.Op > ast.OpPow { // OpAdd..OpPow are the arithmetic ops
			return false
		}
		return numeric(t.L) && numeric(t.R)
	}
	return false
}
