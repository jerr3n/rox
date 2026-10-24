package hoisting

import (
	"fmt"

	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/scope"
)

type index struct {
	info   *scope.Info
	byName map[string][]*scope.Symbol
	taken  map[string]bool
}

func NewIndex(block *ast.Block) *index {
	info := scope.Resolve(block)
	byName := map[string][]*scope.Symbol{}
	taken := map[string]bool{}
	seen := map[*scope.Symbol]bool{}

	for _, v := range info.Symbols {
		_, ok := seen[v]
		if !ok {
			seen[v] = true
			byName[v.Name] = append(byName[v.Name], v)
		}
	}
	for _, v := range info.Globals {
		_, ok := seen[v]
		if !ok {
			seen[v] = true
			byName[v.Name] = append(byName[v.Name], v)
		}
	}
	return &index{
		info:   info,
		byName: byName,
		taken:  taken,
	}
}

func (ix *index) isFree(name string, allowed map[*scope.Symbol]bool) bool {
	if ix.taken[name] {
		return false
	}
	for _, s := range ix.byName[name] {
		if !allowed[s] {
			return false
		}
	}
	return true
}

func (ix *index) freshName(base string) string {
	if ix.isFree(base, nil) {
		ix.taken[base] = true
		return base
	}
	count := 2
	candidate := fmt.Sprintf("%s_%d", base, count)
	for !ix.isFree(candidate, nil) {
		count++
		candidate = fmt.Sprintf("%s_%d", base, count)
	}
	ix.taken[candidate] = true
	return candidate
}

func rename(sym *scope.Symbol, name string) {
	if sym == nil || sym.Decl == nil {
		return
	}
	sym.Decl.Name = name
	for _, target := range [][]*ast.Ident{sym.Reads, sym.Writes} { // 2 birds are no longer with us
		for _, v := range target {
			v.Name = name
		}
	}
	sym.Name = name
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
