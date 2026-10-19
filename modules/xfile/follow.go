package xfile

import (
	"os"
	"path/filepath"

	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/lexer"
	"github.com/jerr3n/rox/parser"
	"github.com/jerr3n/rox/util"
)

type File struct {
	Path     string
	Block    *ast.Block // nil if Err is set
	Requires []Require
	Err      error // reading, lexing or parsing failed
	Fault    util.LevelFault
}

// Follow TODO: add detonation if any element in []File has an error
func (sm *Sourcemap) Follow(entry string) []*File {
	entry = filepath.Clean(entry)
	seen := map[string]bool{entry: true}
	queue := []string{entry}
	var out []*File
	for len(queue) > 0 {
		f := &File{Path: queue[0]}
		queue = queue[1:]
		out = append(out, f)

		//f.Block, f.Err = parseFile(f.Path)
		var lexed *[]lexer.Token
		var err error
		var block *ast.Block
		src, err := os.ReadFile(f.Path)
		if err == nil {
			lexed, err = lexer.Lexer(string(src))
			if err == nil {
				block, err = parser.Parser(*lexed)
				if err != nil {
					f.Fault = util.ParserLevelFault
				} else {
					f.Fault = util.NoFault
					f.Block = block
				}
			} else {
				f.Fault = util.LexerLevelFault
			}
		} else {
			f.Fault = util.ReadLevelFault
		}
		f.Err = err
		if f.Err != nil {
			continue
		}
		f.Requires = sm.Requires(f.Block, f.Path)
		for _, r := range f.Requires {
			if r.Path != "" && !seen[r.Path] {
				seen[r.Path] = true
				queue = append(queue, r.Path)
			}
		}
	}
	return out
}
