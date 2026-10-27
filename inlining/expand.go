package inlining

import (
	"strings"

	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/scope"
)

type expansion struct {
	stmts   []ast.Stmt
	returns []ast.Expr
}

type renamer struct {
	ix    *scope.Index
	c     *candidate
	names map[*scope.Symbol]string
}

func newRenamer(ix *scope.Index, c *candidate) *renamer {
	return &renamer{
		ix:    ix,
		c:     c,
		names: map[*scope.Symbol]string{},
	}
}
func (r *renamer) nameFor(sym *scope.Symbol) string {
	v, e := r.names[sym]
	if e {
		return v
	}
	name := r.ix.FreshName(baseName(sym.Name))
	r.names[sym] = name
	return name
}

func (r *renamer) hook(orig, copy *ast.Ident) {
	val := r.ix.Info.Symbols[orig]
	_, exists := r.c.Internal[val]
	if exists {
		copy.Name = r.nameFor(val)
	}
}

func identExprs(names []string) []ast.Expr {
	out := make([]ast.Expr, 0, len(names))
	for _, name := range names {
		out = append(out, &ast.Ident{Name: name})
	}
	return out
}

func bindings(names []string) []*ast.Binding {
	out := make([]*ast.Binding, 0, len(names))
	for _, name := range names {
		out = append(out, &ast.Binding{Name: &ast.Ident{Name: name}})
	}
	return out
}

func (r *renamer) bindParams(params []*ast.Binding, args []ast.Expr) []ast.Stmt {
	names := make([]string, 0, len(params))
	for _, p := range params {
		names = append(names, r.nameFor(r.ix.Info.Symbols[p.Name]))
	}
	if len(names) == 0 && len(args) == 0 {
		return nil
	}
	if len(names) == 0 {
		names = append(names, r.ix.FreshName("_"))
	}
	if len(names) == len(args) && !multiValue(args[len(args)-1]) {
		out := make([]ast.Stmt, 0, len(names))
		for i, name := range names {
			out = append(out, &ast.Local{
				Names:  bindings([]string{name}),
				Values: []ast.Expr{args[i]},
			})
		}
		return out
	}
	return []ast.Stmt{&ast.Local{
		Names:  bindings(names),
		Values: args,
	}}
}

func multiValue(e ast.Expr) bool {
	switch e.(type) {
	case *ast.Call, *ast.MethodCall, *ast.Vararg:
		return true
	}
	return false
}

func (r *renamer) resultNames(n int) []string {
	out := make([]string, 0, n)
	for range n {
		out = append(out, r.ix.FreshName(baseName(r.c.Sym.Name)+"_result"))
	}
	return out
}

func expand(ix *scope.Index, c *candidate, call *ast.Call, want int) (expansion, bool) {
	r := newRenamer(ix, c)
	var stmts []ast.Stmt

	var results []string
	if c.Shape != returnsNone {
		n := want
		if n == 0 {
			n = 1
		}
		results = r.resultNames(n)
		stmts = append(stmts, &ast.Local{Names: bindings(results)})
	}

	var inner []ast.Stmt
	inner = append(inner, r.bindParams(c.Body.Params, call.Args)...)
	body := ast.CloneBlock(c.Body.Body, r.hook)
	if !rewriteReturns(body, c.Shape, results) {
		return expansion{}, false
	}
	inner = append(inner, body.Stmts...)
	stmts = append(stmts, &ast.Do{Body: &ast.Block{Stmts: inner}})

	var returns []ast.Expr
	if c.Shape == returnsNone {
		for range want {
			returns = append(returns, &ast.Nil{})
		}
	} else {
		returns = identExprs(results[:want])
	}
	return expansion{stmts: stmts, returns: returns}, true
}

func baseName(name string) string {
	for {
		i := strings.LastIndexByte(name, '_')
		if i <= 0 || i == len(name)-1 || strings.Trim(name[i+1:], "0123456789") != "" {
			return name
		}
		name = name[:i]
	}
}
