package scope

import (
	"fmt"

	"github.com/jerr3n/rox/ast"
)

// ---------------------------------------------------------------------------
// Index
//
// Index is a file's resolve info plus which names are in use, for passes that
// rename symbols or make up new ones. A name is in use if any symbol in the
// file has it, or if a pass has taken it since the Index was made.
//
// An Index goes stale when the tree changes; make a new one each round.
// ---------------------------------------------------------------------------

type Index struct {
	Info   *Info
	byName map[string][]*Symbol
	taken  map[string]bool
}

// NewIndex resolves block and indexes its symbols by name.
func NewIndex(block *ast.Block) *Index {
	info := Resolve(block)
	byName := map[string][]*Symbol{}
	taken := map[string]bool{}
	seen := map[*Symbol]bool{}

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
	return &Index{
		Info:   info,
		byName: byName,
		taken:  taken,
	}
}

// IsFree reports whether name can be used without clashing: nobody has taken
// it, and every symbol that already has it is in allowed.
func (ix *Index) IsFree(name string, allowed map[*Symbol]bool) bool {
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

// Take marks name as used, so IsFree and FreshName won't hand it out again.
func (ix *Index) Take(name string) {
	ix.taken[name] = true
}

// FreshName returns base if it's free, or else the first free base_2,
// base_3, ..., and takes it.
func (ix *Index) FreshName(base string) string {
	if ix.IsFree(base, nil) {
		ix.Take(base)
		return base
	}
	count := 2
	candidate := fmt.Sprintf("%s_%d", base, count)
	for !ix.IsFree(candidate, nil) {
		count++
		candidate = fmt.Sprintf("%s_%d", base, count)
	}
	ix.Take(candidate)
	return candidate
}
