package hoisting

import (
	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/scope"
)

func CanHoist(in *ast.Block) bool {
	length := len(in.Stmts) - 1
	if length < 0 {
		return false
	}
	last := in.Stmts[length]
	val, ok := last.(*ast.Return)
	if !ok || len(val.Values) != 1 {
		return false
	}
	returns := 0
	vararg := false
	ast.Inspect(in, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.FuncBody:
			return false
		case *ast.Return:
			returns++
		case *ast.Vararg:
			vararg = true

		}
		return true
	})
	return returns == 1 && !vararg
}

func wrapper(stmt ast.Stmt, info *scope.Info) (name *scope.Symbol, body *ast.Block, ok bool) {
	local, localOk := stmt.(*ast.Local)
	if !localOk || len(local.Names) != 1 || len(local.Values) != 1 {
		return nil, nil, false
	}
	sym := info.Symbols[local.Names[0].Name]
	if sym == nil || len(sym.Writes) > 0 {
		return nil, nil, false
	}
	call, callOk := local.Values[0].(*ast.Call)
	if !callOk || len(call.Args) != 0 {
		return nil, nil, false
	}
	paren, parenOk := call.Fn.(*ast.Paren)
	if !parenOk {
		return nil, nil, false
	}
	fn, fnOk := paren.X.(*ast.Func)
	if !fnOk {
		return nil, nil, false
	}
	fb := fn.Body
	if fb == nil {
		return nil, nil, false
	}
	if len(fb.Params) != 0 || fb.Vararg || len(fb.Attributes) != 0 {
		return nil, nil, false
	}
	if !CanHoist(fb.Body) {
		return nil, nil, false
	}
	return sym, fb.Body, true
}

func deconflict(ix *scope.Index, body *ast.Block) {
	for _, sym := range topLevelDecls(body, ix.Info) {
		if ix.IsFree(sym.Name, map[*scope.Symbol]bool{
			sym: true,
		}) {
			ix.Take(sym.Name)
		} else {
			sym.Rename(ix.FreshName(sym.Name))
		}
	}
}

func bindReturn(body *ast.Block, name string) {
	i := len(body.Stmts) - 1
	ret := body.Stmts[i].(*ast.Return)
	v := ret.Values[0]
	if fn, ok := v.(*ast.Func); ok {
		body.Stmts[i] = &ast.LocalFunc{
			Name: &ast.Ident{Name: name},
			Body: fn.Body,
		}
	} else {
		body.Stmts[i] = &ast.Local{
			Names:  []*ast.Binding{{Name: &ast.Ident{Name: name}}},
			Values: []ast.Expr{v},
		}
	}
}

func Unwrap(block *ast.Block) *ast.Block {
	ix := scope.NewIndex(block)
	i := 0
	for i < len(block.Stmts) {
		sym, body, ok := wrapper(block.Stmts[i], ix.Info)
		if !ok {
			i++
			continue
		}
		if !ix.IsFree(sym.Name, map[*scope.Symbol]bool{sym: true}) {
			i++
			continue
		}
		deconflict(ix, body)
		bindReturn(body, sym.Name)
		out := make([]ast.Stmt, 0, len(block.Stmts)-1+len(body.Stmts))
		out = append(out, block.Stmts[:i]...)
		out = append(out, body.Stmts...)
		out = append(out, block.Stmts[i+1:]...)
		block.Stmts = out
		i += len(body.Stmts)
		ix = scope.NewIndex(block)
	}
	return block
}

func topLevelDecls(block *ast.Block, info *scope.Info) []*scope.Symbol {
	var syms []*scope.Symbol
	for _, s := range block.Stmts {
		switch s := s.(type) {
		case *ast.Local:
			for _, v := range s.Names {
				sym := info.Symbols[v.Name]
				syms = append(syms, sym)
			}
		case *ast.LocalFunc:
			sym := info.Symbols[s.Name]
			syms = append(syms, sym)
		default:
			continue
		}
	}
	return syms
}
