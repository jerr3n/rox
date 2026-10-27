package inlining

import (
	"slices"

	"github.com/jerr3n/rox/ast"
)

func rewriteReturns(body *ast.Block, shape returnShape, results []string) bool {
	switch shape {
	case returnsNone:
		return true
	case returnsTrailing:
		replaceReturns(body, results)
		return true
	case returnsEarly:
		rewriteEarly(body, results)
		return true
	}
	return false
}

func rewriteEarly(body *ast.Block, results []string) {
	sinkTails(body)
	replaceReturns(body, results)
}

func sinkTails(b *ast.Block) {
	for i, st := range b.Stmts {
		ifStmt, ok := st.(*ast.If)
		if !ok || !anyBranchReturns(ifStmt) {
			continue
		}
		tail := slices.Clone(b.Stmts[i+1:])
		if len(tail) == 0 {
			return
		}
		b.Stmts = b.Stmts[:i+1]
		used := false
		give := func(branch *ast.Block) {
			if used {
				branch.Stmts = append(branch.Stmts, ast.CloneBlock(&ast.Block{Stmts: tail}, nil).Stmts...)
			} else {
				branch.Stmts = append(branch.Stmts, tail...)
				used = true
			}
			sinkTails(branch)
		}
		for _, branch := range branches(ifStmt) {
			if !endsInReturn(branch) {
				give(branch)
			}
		}
		if ifStmt.Else == nil {
			ifStmt.Else = &ast.Block{}
			give(ifStmt.Else)
		}
		return
	}
}

func endsInReturn(b *ast.Block) bool {
	if b == nil || len(b.Stmts) == 0 {
		return false
	}
	_, ok := b.Stmts[len(b.Stmts)-1].(*ast.Return)
	return ok
}

func replaceReturns(b *ast.Block, results []string) {
	if b == nil {
		return
	}
	out := make([]ast.Stmt, 0, len(b.Stmts))
	for _, st := range b.Stmts {
		switch t := st.(type) {
		case *ast.Return:
			out = append(out, returnToAssign(t, results)...)
			continue
		case *ast.If:
			for _, branch := range branches(t) {
				replaceReturns(branch, results)
			}
		case *ast.While:
			replaceReturns(t.Body, results)
		case *ast.Repeat:
			replaceReturns(t.Body, results)
		case *ast.NumericFor:
			replaceReturns(t.Body, results)
		case *ast.GenericFor:
			replaceReturns(t.Body, results)
		case *ast.Do:
			replaceReturns(t.Body, results)
		}
		out = append(out, st)
	}
	b.Stmts = out
}

func returnToAssign(ret *ast.Return, results []string) []ast.Stmt {
	if len(results) == 0 || len(ret.Values) == 0 {
		return nil
	}
	return []ast.Stmt{&ast.Assign{
		Targets: identExprs(results),
		Values:  ret.Values,
	}}
}

func branches(s *ast.If) []*ast.Block {
	out := []*ast.Block{s.Then}
	for _, ei := range s.ElseIfs {
		out = append(out, ei.Body)
	}
	if s.Else != nil {
		out = append(out, s.Else)
	}
	return out
}

func anyBranchReturns(s *ast.If) bool {
	for _, branch := range branches(s) {
		if endsInReturn(branch) {
			return true
		}
	}
	return false
}
