package bundling

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/lexer"
	"github.com/jerr3n/rox/xfile"
	"github.com/jerr3n/rox/parser"
	"github.com/jerr3n/rox/printer"
	"github.com/jerr3n/rox/scope"
	"github.com/jerr3n/rox/util"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Errors
// ---------------------------------------------------------------------------

// LoadError is returned when some files couldn't be read, lexed or parsed.
// Each one was already logged with its own error.
type LoadError struct {
	Broken []string
}

func (e *LoadError) Error() string {
	return fmt.Sprintf("failed to load %s", strings.Join(e.Broken, ", "))
}

// CycleError is returned when files require each other in a loop. Cycle
// starts and ends with the same path.
type CycleError struct {
	Cycle []string
}

func (e *CycleError) Error() string {
	return fmt.Sprintf("cyclic require: %s", strings.Join(e.Cycle, " -> "))
}

// ---------------------------------------------------------------------------
// Pipeline
// ---------------------------------------------------------------------------

// Bundle reads entry and every file it requires, and returns them as one
// Luau source. Each pass runs on the bundled tree, in order, before it's
// printed. logger may be nil.
func Bundle(sm *xfile.Sourcemap, entry string, logger *zap.Logger, passes ...func(*ast.Block) *ast.Block) (string, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	files, err := Load(sm, entry, logger)
	if err != nil {
		return "", err
	}
	ordered, err := Order(files)
	if err != nil {
		return "", err
	}
	for _, f := range ordered {
		logger.Debug("output", zap.String("path", f.Path))
	}
	block := Emit(ordered, Names(ordered), Resolve(ordered))
	for _, pass := range passes {
		block = pass(block)
	}
	return printer.Printer(block, printer.Config{Indent: "\t"}, logger), nil
}

// Load reads, lexes and parses entry and every file it requires, directly or
// not, breadth first. entry is the first file returned. A file that fails is
// logged, and Load returns a *LoadError naming every one that did. logger
// may be nil.
func Load(sm *xfile.Sourcemap, entry string, logger *zap.Logger) ([]*xfile.File, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	entry = filepath.Clean(entry)
	seen := map[string]bool{entry: true}
	queue := []string{entry}
	var files []*xfile.File
	for len(queue) > 0 {
		f := &xfile.File{Path: queue[0]}
		queue = queue[1:]
		files = append(files, f)
		src, err := os.ReadFile(f.Path)
		if err != nil {
			f.Fault, f.Err = util.ReadLevelFault, err
			logger.Error("read fail", zap.String("path", f.Path), zap.Error(err))
			continue
		}
		logger.Debug("read ok", zap.String("path", f.Path))
		fileLogger := logger.With(zap.String("path", f.Path))
		lexed, err := lexer.Lexer(string(src), fileLogger)
		if err != nil {
			f.Fault, f.Err = util.LexerLevelFault, err
			logger.Error("lex fail", zap.String("path", f.Path), zap.Error(err))
			continue
		}
		logger.Debug("lex ok", zap.String("path", f.Path))
		block, err := parser.Parser(*lexed, fileLogger)
		if err != nil {
			f.Fault, f.Err = util.ParserLevelFault, err
			logger.Error("parse fail", zap.String("path", f.Path), zap.Error(err))
			continue
		}
		logger.Debug("parse ok", zap.String("path", f.Path))
		f.Fault = util.NoFault
		f.Block = block
		f.Requires = sm.Requires(f.Block, f.Path)
		for _, r := range f.Requires {
			if r.Path != "" && !seen[r.Path] {
				seen[r.Path] = true
				queue = append(queue, r.Path)
			}
		}
	}
	// each broken file was already logged above, so collect them all and stop once
	var broken []string
	for _, file := range files {
		if file.Fault > util.NoFault {
			broken = append(broken, file.Path)
			continue
		}
		logger.Debug("file ok", zap.String("file", file.Path))
		for _, require := range file.Requires {
			if require.Path == "" {
				return nil, fmt.Errorf("%s: require at byte %d couldn't be resolved", file.Path, require.Call.Start)
			}
			logger.Debug("require ok",
				zap.String("path", file.Path),
				zap.String("require", require.Path),
			)
		}
	}
	if len(broken) > 0 {
		return nil, &LoadError{Broken: broken}
	}
	return files, nil
}

// visitStatus is how far Order has got with a file.
type visitStatus int

const (
	unvisited visitStatus = iota
	visiting
	visited
)

type orderer struct {
	lookup map[string]*xfile.File
	state  map[string]visitStatus
	stack  []string // files being visited, outermost first
	output []*xfile.File
}

// Order sorts files so every file comes after the files it requires. files[0]
// is the entry, and it comes last. A loop of requires is a *CycleError.
func Order(files []*xfile.File) ([]*xfile.File, error) {
	o := &orderer{
		lookup: map[string]*xfile.File{},
		state:  map[string]visitStatus{},
	}
	for _, file := range files {
		o.lookup[file.Path] = file
		o.state[file.Path] = unvisited
	}
	if err := o.visit(files[0].Path); err != nil {
		return nil, err
	}
	return o.output, nil
}

func (o *orderer) visit(path string) error {
	switch o.state[path] {
	case visiting:
		i := slices.Index(o.stack, path)
		return &CycleError{Cycle: append(slices.Clone(o.stack[i:]), path)}
	case visited:
		return nil
	case unvisited:
	}
	o.state[path] = visiting
	o.stack = append(o.stack, path)
	file := o.lookup[path]
	for _, r := range file.Requires {
		if err := o.visit(r.Path); err != nil {
			return err
		}
	}
	o.output = append(o.output, file)
	o.stack = o.stack[:len(o.stack)-1]
	o.state[path] = visited
	return nil
}

// Names gives every file the local its module is stored in, made from its
// path: a/b.luau becomes __mod_a_b.
func Names(files []*xfile.File) map[string]string {
	names := map[string]string{}
	for _, t := range files {
		path := t.Path
		ext := filepath.Ext(path)
		res := strings.TrimSuffix(path, ext)
		var sanitized strings.Builder
		for _, c := range res {
			if c <= utf8.RuneSelf && util.IsWordChar(byte(c)) {
				sanitized.WriteString(string(c))
			} else {
				sanitized.WriteString("_") // do not write your codebase primarily in anything that does not use a latin character set/english character set
			}
		}
		prefixed := fmt.Sprintf("__mod_%s", sanitized.String())
		vals := slices.Collect(maps.Values(names))
		if slices.Contains(vals, prefixed) {
			count := 1
			for _, v := range vals {
				if strings.HasPrefix(v, prefixed) { // this will not shoot you in the foot unless you make it
					count++
				} // if theres 2 functions in your code called something like 3dmakearray and 3dappendarray i dont feel bad for you
			}
			names[path] = fmt.Sprintf("%s_instance%d", prefixed, count)
		} else {
			names[path] = prefixed
		}
	}
	return names
}

// Resolve links every variable in each file to its symbol, keyed by path. An
// Info only describes the tree it was made from, so it goes stale once Emit
// rewrites that tree.
func Resolve(files []*xfile.File) map[string]*scope.Info {
	infos := map[string]*scope.Info{}
	for _, f := range files {
		infos[f.Path] = scope.Resolve(f.Block)
	}
	return infos
}

// Emit joins ordered files, as returned by Order, into one block. Each
// require becomes the name of the module it loads, every module but the
// entry is wrapped in a function that runs once, and the entry goes in as-is
// at the bottom. infos must come from Resolve on the same, unrewritten files.
func Emit(ordered []*xfile.File, names map[string]string, infos map[string]*scope.Info) *ast.Block {
	entry := ordered[len(ordered)-1]
	var bundle []ast.Stmt
	for _, t := range ordered {
		m := map[*ast.Call]string{}
		for _, r := range t.Requires {
			m[r.Call] = names[r.Path]
		}
		ast.RewriteBlock(t.Block, func(expr ast.Expr) ast.Expr {
			call, ok := expr.(*ast.Call)
			if !ok {
				return expr
			}
			name, inMap := m[call]
			if inMap {
				return &ast.Ident{
					Name: name,
				}
			}
			return expr
		}, func(stmt ast.Stmt) []ast.Stmt {
			callstmt, ok := stmt.(*ast.CallStmt)
			if !ok {
				return []ast.Stmt{stmt}
			}
			_, ok2 := callstmt.Call.(*ast.Ident)
			if ok2 {
				return nil
			}
			return []ast.Stmt{stmt}
		})
		if t == entry {
			bundle = append(bundle, t.Block.Stmts...)
			continue
		}
		// local __mod_x = (function() <t.Block> end)()
		wrapped := &ast.Local{
			Names: []*ast.Binding{{Name: &ast.Ident{Name: names[t.Path]}}},
			Values: []ast.Expr{&ast.Call{
				Fn: &ast.Paren{X: &ast.Func{Body: &ast.FuncBody{Body: t.Block}}},
			}},
		}
		bundle = append(bundle, wrapped)
	}
	return &ast.Block{Stmts: bundle}
}
