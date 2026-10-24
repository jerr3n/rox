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
func chooseName(ix *index, group map[*scope.Symbol]bool) (string, bool) {
	counts := map[string]int{}
	for s := range group {
		counts[s.Name]++
	}
	candidates := []string{}
	for k := range counts {
		candidates = append(candidates, k)
	}
	// readable names first, then the most common, then alphabetical so the
	// result doesn't depend on map order
	slices.SortFunc(candidates, func(a, b string) int {
		return cmp.Or(
			cmp.Compare(modRank(a), modRank(b)),
			cmp.Compare(counts[b], counts[a]),
			cmp.Compare(a, b),
		)
	})
	for _, c := range candidates {
		if ix.isFree(c, group) {
			ix.taken[c] = true
			return c, true
		}
	}
	return "", false
}

// modRank is 1 for the __mod_ names bundling.Names makes, 0 for the rest.
func modRank(name string) int {
	if strings.HasPrefix(name, "__mod_") {
		return 1
	}
	return 0
}

// inlineOnce merges each group of aliases (`local x = y`, plus every other
// alias of y) into one binding and deletes the alias statements. A y that is
// itself an alias waits for the next round. It reports whether it changed
// anything.
func inlineOnce(block *ast.Block) bool {
	ix := NewIndex(block)
	type found struct {
		stmt ast.Stmt
		x, y *scope.Symbol
	}
	var aliases []found
	ast.Inspect(block, func(node ast.Node) bool {
		if stmt, ok := node.(ast.Stmt); ok {
			if x, y, ok := alias(stmt, ix.info); ok {
				aliases = append(aliases, found{
					stmt: stmt,
					x:    x,
					y:    y,
				})
			}
		}
		return true
	})

	// group by y, keeping the order the y's first appeared so renames don't
	// depend on map order
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
			rename(s, name)
		}
		for _, st := range stmts[y] {
			del[st] = true
		}
	}

	// each alias is now `local name = name`
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
