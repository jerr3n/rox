// Runs xfile over this Rojo project: for every synced file, list its requires
// and what they resolve to.
//
//	go run ./playground   (from the repo root)
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jerr3n/rox/lexer"
	"github.com/jerr3n/rox/modules/xfile"
	"github.com/jerr3n/rox/parser"
)

func main() {
	dir := "playground"
	if _, err := os.Stat("default.project.json"); err == nil {
		dir = "." // run from inside playground/
	}

	// sourcemap.json is gitignored, so make a fresh one each run.
	rojo := exec.Command("rojo", "sourcemap", "default.project.json", "-o", "sourcemap.json")
	rojo.Dir = dir
	rojo.Stderr = os.Stderr
	if err := rojo.Run(); err != nil {
		fmt.Println("rojo sourcemap:", err)
		os.Exit(1)
	}

	sm, err := xfile.LoadSourcemap(filepath.Join(dir, "sourcemap.json"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	files := sm.Files()
	sort.Strings(files)
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			fmt.Println(err)
			continue
		}
		toks, err := lexer.Lexer(string(src), nil)
		if err != nil {
			fmt.Printf("%s: lex error: %v\n", path, err)
			continue
		}
		block, err := parser.Parser(*toks, nil)
		if err != nil {
			fmt.Printf("%s: parse error: %v\n", path, err)
			continue
		}

		inst := sm.File(path)
		fmt.Printf("%s  (%s %s)\n", path, inst.ClassName, inst.FullName())
		reqs := sm.Requires(block, path)
		if len(reqs) == 0 {
			fmt.Println("    no requires")
		}
		for _, r := range reqs {
			line := 1 + strings.Count(string(src[:r.Call.Pos()]), "\n")
			if r.Target == nil {
				fmt.Printf("    L%-2d ✗ unresolved\n", line)
				continue
			}
			fmt.Printf("    L%-2d → %-34s %s\n", line, r.Target.FullName(), filepath.Join(dir, r.Target.Source()))
		}
		fmt.Println()
	}
}
