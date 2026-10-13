package main

import (
	"fmt"

	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/lexer"
	"github.com/jerr3n/rox/parser"
)

var table = `
local t = {1,2}
table.insert(t,3)
`

func playground() {
	var psuedoarg = &table
	lex, err := lexer.Lexer(*psuedoarg)
	if err != nil {
		panic(err)
	}
	parse, err := parser.Parser(*lex)
	if err != nil {
		panic(err)
	}
	fmt.Println(ast.Dump(parse))

	fmt.Println(ast.Dump(parse.Stmts[1]))
	for i, v := range parse.Stmts {
		c, ok := v.(*ast.CallStmt)
		if !ok {
			continue
		}
		var upper string
		var lower string
		var target ast.Expr
		var expect ast.Expr
		var ready bool
		if call, ok := c.Call.(*ast.Call); ok {
			fmt.Println(ast.Dump(call.Fn))
			if field, ok := call.Fn.(*ast.Field); ok {
				fmt.Println(field.Name.Name) // "insert"
				fmt.Println(ast.Dump(field))
				lower = field.Name.Name
				if obj, ok := field.X.(*ast.Ident); ok {
					fmt.Println(obj.Name) // "table"
					upper = obj.Name
					target = call.Args[0]
					expect = call.Args[1]
					if field.Name.Name == "insert" && obj.Name == "table" {
						
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
		if upper == "table" && lower == "insert" {
			fmt.Println("!!!!!!!!!!!!!!!!!! TABLE.INSERT FOUND!!")
			fmt.Println("here i have to take out that part of the ast but i dont know how to...")
		}
	}
	//printed := printer.Printer(parse, printer.Config{
	//	Indent: " ",
	//	Minify: false,
	//})
	//
}
