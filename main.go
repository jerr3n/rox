package main

import (
	"fmt"

	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/lexer"
	"github.com/jerr3n/rox/parser"
)

const test = `
local t = {1,2}
t[#t+1]=3
`

func main() {
	lex, _ := lexer.Lexer(test)
	parse, _ := parser.Parser(*lex)
	x := ast.Block{
		Span: ast.Span{},
		Stmts: []ast.Stmt{
			&ast.Assign{
				Span: ast.Span{},
				Targets: []ast.Expr{
					&ast.Index{
						Span: ast.Span{},
						X: &ast.Ident{
							Span: ast.Span{},
							Name: "t",
						}, // from ln27-30 and ln37-40, two birds can be executed
						Key: &ast.Binary{
							Span: ast.Span{},
							Op:   ast.BinaryOp(lexer.OpAdd),
							L: &ast.Unary{
								Span: ast.Span{},
								Op:   ast.UnaryOp(lexer.OpLen),
								X: &ast.Ident{
									Span: ast.Span{},
									Name: "t",
								},
							},
							R: &ast.Number{
								Span: ast.Span{},
								Raw:  "1",
							},
						},
					},
				},
				Values: []ast.Expr{
					&ast.Number{
						Span: ast.Span{},
						Raw:  "3",
					},
				},
			},
		},
	}
	fmt.Println(ast.Dump(parse))
}
