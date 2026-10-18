// Package astgen has short constructors for building AST nodes by hand,
// mostly for writing expected trees in tests. Spans are left empty.
package astgen

import "github.com/jerr3n/rox/ast"

// Ident builds the identifier `name`.
func Ident(name string) *ast.Ident { return &ast.Ident{Name: name} }

// Num builds a number literal from its source text, e.g. "1" or "0xFF".
func Num(raw string) *ast.Number { return &ast.Number{Raw: raw} }

// Str builds a string literal with the already-decoded value v.
func Str(v string) *ast.String { return &ast.String{Value: v} }
