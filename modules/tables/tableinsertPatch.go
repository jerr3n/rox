package tables

import (
	"fmt"

	"github.com/jerr3n/rox/ast"
)

func TableInsertPatch(in *ast.Block) *ast.Block {

	for i, v := range in.Stmts {
		c, ok := v.(*ast.CallStmt)
		if !ok {
			continue
		}
		fmt.Println("found call statement")
		var target ast.Expr
		var expect ast.Expr
		if call, ok := c.Call.(*ast.Call); ok {
			fmt.Println("found call ")
			if field, ok := call.Fn.(*ast.Field); ok {
				fmt.Println("found field")
				if obj, ok := field.X.(*ast.Ident); ok {
					fmt.Println("found object")
					if field.Name.Name == "insert" && obj.Name == "table" {
						target = call.Args[0]
						expect = call.Args[1]
						fmt.Println("found table.insert")
					} else {
						continue
					}
				} else {
					continue
				}
			} else {
				continue
			}
		} else {
			continue
		}
		replacement := &ast.Assign{
			Span: ast.Span{},
			Targets: []ast.Expr{
				&ast.Index{
					Span: ast.Span{},
					X:    target,
					Key: &ast.Binary{
						Span: ast.Span{},
						Op:   ast.OpAdd,
						L: &ast.Unary{
							Span: ast.Span{},
							Op:   ast.OpLen,
							X:    target,
						},
						R: &ast.Number{
							Span: ast.Span{},
							Raw:  "1",
						},
					},
				},
			},
			Values: []ast.Expr{
				expect,
			},
		}
		in.Stmts[i] = replacement
	}
	return in
}
