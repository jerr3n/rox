package inlining

import (
	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/scope"
)

type returnShape int

const (
	returnsNone returnShape = iota
	returnsTrailing
	returnsEarly
	returnsUnsupported
)

type candidate struct {
	Sym      *scope.Symbol
	Decl     ast.Stmt
	Body     *ast.FuncBody
	Internal map[*scope.Symbol]bool
	Shape    returnShape
	Returns  int
}

type returnScan struct {
	Shape   returnShape
	Returns int
}

type place int

const (
	atTop place = iota
	atTopIf
	atOther
)

const unknownCount int = -1

func returnCount(ret *ast.Return) int {
	if len(ret.Values) == 0 {
		return 0
	}
	last := ret.Values[len(ret.Values)-1]
	switch last.(type) {
	case *ast.Call, *ast.MethodCall, *ast.Vararg:
		return unknownCount
	}
	return len(ret.Values)

}

func (s *returnScan) found(shape returnShape, ret *ast.Return) {
	if shape > s.Shape {
		s.Shape = shape
	}

	n := returnCount(ret)
	if n == unknownCount || s.Returns == unknownCount {
		s.Returns = unknownCount
	} else if n > s.Returns {
		s.Returns = n
	}
}

func (s *returnScan) block(b *ast.Block, at place) {
	if b == nil {
		return
	}
	for _, st := range b.Stmts {
		s.stmt(st, at)
	}
}

func (s *returnScan) stmt(st ast.Stmt, at place) {
	switch t := st.(type) {
	case *ast.Return:
		if at == atTop {
			s.found(returnsTrailing, t)
		} else if at == atTopIf {
			s.found(returnsEarly, t)
		} else {
			s.found(returnsUnsupported, t)
		}
	case *ast.If:
		var inner place
		if at == atTop {
			inner = atTopIf
		} else {
			inner = atOther
		}
		s.block(t.Then, inner)
		for _, elif := range t.ElseIfs {
			s.block(elif.Body, inner)
		}
		s.block(t.Else, inner)
	case *ast.While:
		s.block(t.Body, atOther)
	case *ast.Repeat:
		s.block(t.Body, atOther)
	case *ast.NumericFor:
		s.block(t.Body, atOther)
	case *ast.GenericFor:
		s.block(t.Body, atOther)
	case *ast.Do:
		s.block(t.Body, atOther)
	}
}

func scanReturns(body *ast.FuncBody) (returnShape, int) {
	s := returnScan{
		Shape:   returnsNone,
		Returns: 0,
	}
	s.block(body.Body, atTop)
	return s.Shape, s.Returns
}

func isRecursive(sym *scope.Symbol, body *ast.FuncBody, info *scope.Info) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		ident, isIdent := n.(*ast.Ident)
		if isIdent && info.Symbols[ident] == sym {
			found = true
			return false
		}
		return true
	})
	return found
}

func internalSymbols(body *ast.FuncBody, info *scope.Info) map[*scope.Symbol]bool {
	set := map[*scope.Symbol]bool{}
	ast.Inspect(body, func(node ast.Node) bool {
		ident, isIdent := node.(*ast.Ident)
		if isIdent {
			sym := info.Symbols[ident]
			if sym != nil && sym.Decl == ident {
				set[sym] = true
			}
		}
		return true
	})
	return set
}

func newCandidate(sym *scope.Symbol, decl ast.Stmt, body *ast.FuncBody, info *scope.Info) (*candidate, bool) {
	if len(sym.Writes) != 0 || body.Vararg || len(body.Attributes) != 0 || isRecursive(sym, body, info) {
		return nil, false
	}
	shape, returns := scanReturns(body)
	if shape == returnsUnsupported {
		return nil, false
	}
	return &candidate{
		Sym:      sym,
		Decl:     decl,
		Body:     body,
		Internal: internalSymbols(body, info),
		Shape:    shape,
		Returns:  returns,
	}, true
}

func findCandidates(ix *scope.Index, block *ast.Block) map[*scope.Symbol]*candidate {
	res := map[*scope.Symbol]*candidate{}
	ast.Inspect(block, func(node ast.Node) bool {
		sym, body, ok := declOf(node, ix.Info)
		if ok {

			c, ok := newCandidate(sym, node.(ast.Stmt), body, ix.Info)
			if ok {
				res[sym] = c
			}
		}
		return true
	})
	return res
}

func declOf(n ast.Node, info *scope.Info) (sym *scope.Symbol, body *ast.FuncBody, ok bool) {
	local, isLocal := n.(*ast.Local)
	localFunc, isLocalFunc := n.(*ast.LocalFunc)
	if isLocalFunc {
		sym = info.Symbols[localFunc.Name]
		if sym == nil {
			ok = false
			return
		}
		body = localFunc.Body
		ok = true
		return
	}

	if isLocal {
		if len(local.Values) != 1 || len(local.Names) != 1 {
			return nil, nil, false
		}
		fn, isFn := local.Values[0].(*ast.Func)
		if !isFn {
			return nil, nil, false
		}
		sym = info.Symbols[local.Names[0].Name]
		if sym == nil {
			return nil, nil, false
		}
		body = fn.Body
		ok = true
		return
	}
	return nil, nil, false
}
