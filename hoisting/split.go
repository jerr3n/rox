package hoisting

import (
	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/scope"
)

func emptyTableDecl(stmt ast.Stmt, info *scope.Info) (*scope.Symbol, bool) {
	local, localOk := stmt.(*ast.Local)
	if !localOk || (len(local.Names) != 1 || len(local.Values) != 1) {
		return nil, false
	}
	table, tableOk := local.Values[0].(*ast.Table)
	if !tableOk || len(table.Items) != 0 {
		return nil, false
	}
	sym := info.Symbols[local.Names[0].Name]
	if sym == nil || len(sym.Writes) > 0 {
		return nil, false
	}
	return sym, true
}

func fieldAssigns(block *ast.Block, i int, sym *scope.Symbol, info *scope.Info) (assigns map[string]*ast.Assign, targets map[*ast.Field]bool, last int, ok bool) {
	assigns = map[string]*ast.Assign{}
	targets = map[*ast.Field]bool{}
	last = i
	for j := i + 1; j < len(block.Stmts); j++ {
		a, aok := block.Stmts[j].(*ast.Assign)
		if !aok || (len(a.Targets) != 1 || len(a.Values) != 1) {
			break
		}
		field, fieldOk := a.Targets[0].(*ast.Field)
		if !fieldOk {
			break
		}
		id, idOk := field.X.(*ast.Ident)
		if !idOk || (info.Symbols[id] != sym) {
			break
		}
		name := field.Name.Name
		_, inAssigns := assigns[name]
		if inAssigns {
			return nil, nil, 0, false
		}
		assigns[name] = a
		targets[field] = true
		last = j
	}
	if len(assigns) == 0 {
		return nil, nil, 0, false
	}
	return assigns, targets, last, true
}

func readsOK(block *ast.Block, sym *scope.Symbol, info *scope.Info, assigns map[string]*ast.Assign, targets map[*ast.Field]bool, last int) bool {
	uses := 0
	ok := true
	for k, v := range block.Stmts {
		ast.Inspect(v, func(n ast.Node) bool {
			field, fieldOk := n.(*ast.Field)
			if !fieldOk {
				return true
			}
			id, idOk := field.X.(*ast.Ident)
			if !idOk || info.Symbols[id] != sym {
				return true
			}
			uses++
			_, targetsOk := targets[field]
			if targetsOk {
				return true
			}
			_, assignsOk := assigns[field.Name.Name]
			if !assignsOk || k <= last {
				ok = false
			}
			return true
		})
	}
	return ok && (uses == len(sym.Reads))
}

func splitOnce(block *ast.Block) bool {
	ix := NewIndex(block)
	for i, stmt := range block.Stmts {
		sym, ok := emptyTableDecl(stmt, ix.info)
		if !ok {
			continue
		}
		assigns, targets, last, ok := fieldAssigns(block, i, sym, ix.info)
		if !ok {
			continue
		}
		if !readsOK(block, sym, ix.info, assigns, targets, last) {
			continue
		}
		fresh := map[string]string{}
		for j := i + 1; j <= last; j++ {
			name := block.Stmts[j].(*ast.Assign).Targets[0].(*ast.Field).Name.Name
			fresh[name] = ix.freshName(name)
		}
		for j := i + 1; j <= last; j++ {
			a := block.Stmts[j].(*ast.Assign)
			name := a.Targets[0].(*ast.Field).Name.Name
			block.Stmts[j] = &ast.Local{
				Names:  []*ast.Binding{{Name: &ast.Ident{Name: fresh[name]}}},
				Values: a.Values,
			}
		}
		ast.RewriteBlock(block, func(expr ast.Expr) ast.Expr {
			field, fieldOk := expr.(*ast.Field)
			if !fieldOk {
				return expr
			}
			id, idOk := field.X.(*ast.Ident)
			if !idOk {
				return expr
			}
			if ix.info.Symbols[id] != sym {
				return expr
			}
			return &ast.Ident{
				Name: fresh[field.Name.Name],
			}
		}, nil)
		block.Stmts = append(block.Stmts[:i], block.Stmts[i+1:]...)
		return true
	}
	return false
}

func SplitTables(block *ast.Block) *ast.Block {
	for splitOnce(block) {
	}
	return block
}
