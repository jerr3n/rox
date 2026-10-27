package hoisting

import (
	"cmp"
	"slices"
	"strings"

	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/scope"
)

func alias(stmt ast.Stmt, info *scope.Info) (x, y *scope.Symbol, ok bool) {
	local, localOk := stmt.(*ast.Local)
	if !localOk || len(local.Names) != 1 || len(local.Values) != 1 {
		return nil, nil, false
	}
	id, idOk := local.Values[0].(*ast.Ident)
	if !idOk {
		return nil, nil, false
	}
	x = info.Symbols[local.Names[0].Name]
	y = info.Symbols[id]
	if (x == nil || y == nil) || y.Decl == nil || (len(x.Writes) > 0 || len(y.Writes) > 0) {
		return nil, nil, false
	}
	ok = true
	return
}
func chooseName(ix *scope.Index, group map[*scope.Symbol]bool) (string, bool) {
	counts := map[string]int{}
	for s := range group {
		counts[s.Name]++
	}
	candidates := []string{}
	for k := range counts {
		candidates = append(candidates, k)
	}

	slices.SortFunc(candidates, func(a, b string) int {
		return cmp.Or(
			cmp.Compare(modRank(a), modRank(b)),
			cmp.Compare(counts[b], counts[a]),
			cmp.Compare(a, b),
		)
	})
	for _, c := range candidates {
		if ix.IsFree(c, group) {
			ix.Take(c)
			return c, true
		}
	}
	return "", false
}

func modRank(name string) int {
	if strings.HasPrefix(name, "__mod_") {
		return 1
	}
	return 0
}

func inlineOnce(block *ast.Block) bool {
	ix := scope.NewIndex(block)
	type found struct {
		stmt ast.Stmt
		x, y *scope.Symbol
	}
	var aliases []found
	ast.Inspect(block, func(node ast.Node) bool {
		if stmt, ok := node.(ast.Stmt); ok {
			if x, y, ok := alias(stmt, ix.Info); ok {
				aliases = append(aliases, found{
					stmt: stmt,
					x:    x,
					y:    y,
				})
			}
		}
		return true
	})

	isX := map[*scope.Symbol]bool{}
	groups := map[*scope.Symbol]map[*scope.Symbol]bool{}
	stmts := map[*scope.Symbol][]ast.Stmt{}
	var order []*scope.Symbol
	for _, a := range aliases {
		isX[a.x] = true
		if groups[a.y] == nil {
			groups[a.y] = map[*scope.Symbol]bool{a.y: true}
			order = append(order, a.y)
		}
		groups[a.y][a.x] = true
		stmts[a.y] = append(stmts[a.y], a.stmt)
	}

	del := map[ast.Stmt]bool{}
	for _, y := range order {
		if isX[y] {
			continue
		}
		name, ok := chooseName(ix, groups[y])
		if !ok {
			continue
		}
		for s := range groups[y] {
			s.Rename(name)
		}
		for _, st := range stmts[y] {
			del[st] = true
		}
	}

	ast.RewriteBlock(block, nil, func(stmt ast.Stmt) []ast.Stmt {
		if del[stmt] {
			return nil
		}
		return []ast.Stmt{stmt}
	})
	return len(del) > 0
}

func InlineAliases(block *ast.Block) *ast.Block {
	for inlineOnce(block) {
	}
	return block
}
