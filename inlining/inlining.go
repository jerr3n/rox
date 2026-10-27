package inlining

import (
	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/scope"
)

const maxRounds = 64

func Inline(block *ast.Block) *ast.Block {
	for range maxRounds {
		if !inlineOnce(block) {
			break
		}
	}
	return block
}

func inlineOnce(block *ast.Block) bool {
	uniquify(block)
	ix := scope.NewIndex(block)
	cands := findCandidates(ix, block)
	bodies := map[*ast.FuncBody]bool{}
	for _, c := range cands {
		bodies[c.Body] = true
	}

	changed := false
	ast.Inspect(block, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.FuncBody:
			return !bodies[t]
		case *ast.Block:
			out := make([]ast.Stmt, 0, len(t.Stmts))
			for _, st := range t.Stmts {
				stmts, ok := liftStmt(ix, cands, st)
				out = append(out, stmts...)
				changed = changed || ok
			}
			t.Stmts = out
		}
		return true
	})
	return changed
}

func uniquify(block *ast.Block) {
	ix := scope.NewIndex(block)
	kept := map[string]bool{}
	for name := range ix.Info.Globals {
		kept[name] = true
		ix.Take(name)
	}
	ast.Inspect(block, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		sym := ix.Info.Symbols[id]
		if sym == nil || sym.Decl != id {
			return true
		}
		if !kept[sym.Name] {
			kept[sym.Name] = true
			ix.Take(sym.Name)
			return true
		}
		sym.Rename(ix.FreshName(sym.Name))
		return true
	})
}
