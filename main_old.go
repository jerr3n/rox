package main

//
//import (
//	"fmt"
//
//	"github.com/davecgh/go-spew/spew"
//	"github.com/jerr3n/rox/ast"
//	"github.com/jerr3n/rox/modules/xfile"
//)
//
//const entry = "main.luau"
//
//type RequirePtr struct {
//	name string
//	file *xfile.File
//}
//
//func main() {
//	sm := &xfile.Sourcemap{}
//	data := sm.Follow(entry)
//	//*xfile.File
//	// m
//	m := map[*xfile.File]map[string]RequirePtr{}
//	files := map[string]*xfile.File{}
//	for _, v := range data {
//		files[v.Path] = v
//	}
//	for _, v := range data {
//		//v.Requires
//		//spew.Dump(v)
//		//spew.Dump(v.Block)
//		// file failed to read, lex or parse
//		if v.Err != nil {
//			continue
//		}
//		m[v] = map[string]RequirePtr{}
//		for _, vStmt := range v.Block.Stmts {
//			//spew.Dump(vBlock)
//			val, isLocal := vStmt.(*ast.Local)
//			// if is not local
//			if !isLocal {
//				continue
//			}
//			// if how many names is not one
//			// if how many values is not one
//			// I KNOW THIS DOESN'T ACCOUNT FOR `local x, y = require("x"), require("y")`
//			if len(val.Names) != 1 || len(val.Names) != len(val.Values) {
//				continue
//			}
//			// if expression assigned is a call
//			call, isCall := val.Values[0].(*ast.Call)
//			if !isCall {
//				continue
//			}
//
//			for _, r := range v.Requires {
//				if r.Call == call {
//					fmt.Println("found require")
//					m[v][val.Names[0].Name.Name] = RequirePtr{
//						name: r.Path,
//						file: files[r.Path],
//					}
//					break
//				}
//			}
//			//array = append(array, map[string]string{val.Names[0].Name.Name: required.Value})
//		}
//		/*
//			ref
//		*/
//		for file, ptr := range m {
//
//		}
//	}
//
//	spew.Dump(data)
//}
