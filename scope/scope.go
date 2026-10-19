package scope

import "github.com/jerr3n/rox/ast"

// ---------------------------------------------------------------------------
// Symbols
// ---------------------------------------------------------------------------

// Kind says how a symbol was declared.
type Kind uint8

const (
	Local     Kind = iota // local x
	Param                 // function(x)
	LocalFunc             // local function x
	ForVar                // for x = ..., for x in ...
	Self                  // implicit self in function a:b()
	Global                // never declared; made on first use
)

// Symbol is one variable. Two locals with the same name are two symbols.
type Symbol struct {
	Name string
	Kind Kind
	Decl *ast.Ident    // the declaring name; nil for Self and Global
	Func *ast.FuncBody // function it was declared in; nil for the top level and globals

	Reads  []*ast.Ident
	Writes []*ast.Ident

	Captured bool // used from inside a function other than Func
}

// ---------------------------------------------------------------------------
// Scopes
// ---------------------------------------------------------------------------

// Scope is the set of names visible in one block.
type Scope struct {
	Parent *Scope
	Func   *ast.FuncBody // function this scope belongs to; nil for the top level
	names  map[string]*Symbol
}

// Declare adds a new symbol to s, shadowing any earlier one with the same name.
func (s *Scope) Declare(name string, kind Kind, decl *ast.Ident) *Symbol {
	if s.names == nil {
		s.names = map[string]*Symbol{}
	}
	sym := Symbol{
		Name:     name,
		Kind:     kind,
		Decl:     decl,
		Func:     s.Func,
		Reads:    nil,
		Writes:   nil,
		Captured: false,
	}
	s.names[name] = &sym
	return &sym
}

// Lookup finds name in s or its parents. It also returns the scope the
// symbol was found in, or nil if it wasn't found.
func (s *Scope) Lookup(name string) (*Symbol, *Scope) {
	// TODO
	panic("TODO")
}

// ---------------------------------------------------------------------------
// Results
// ---------------------------------------------------------------------------

// Info is what Resolve finds out about a file.
type Info struct {
	// Symbols maps every variable Ident, declaring or using, to its symbol.
	// Idents that aren't variables (x.name, obj:method, ...) aren't in it.
	Symbols map[*ast.Ident]*Symbol
	Globals map[string]*Symbol
}

// Resolve links every variable Ident in file to its symbol.
func Resolve(file *ast.Block) *Info {
	// TODO
	panic("TODO")
}

// ---------------------------------------------------------------------------
// Walking
// ---------------------------------------------------------------------------

// access is how a use touches a variable.
type access uint8

const (
	read      access = iota
	write            // assignment target
	readWrite        // compound assignment target
)

type resolver struct {
	info  *Info
	scope *Scope // innermost scope
}

// push opens a new scope inside the current one. fn is the function it
// belongs to if it starts a new function, or nil to stay in the current one.
func (r *resolver) push(fn *ast.FuncBody) {
	// TODO
}

// pop closes the current scope.
func (r *resolver) pop() {
	// TODO
}

// declare makes a new symbol for id in the current scope.
func (r *resolver) declare(id *ast.Ident, kind Kind) {
	// TODO
}

// use resolves id and records it as a read and/or write.
func (r *resolver) use(id *ast.Ident, acc access) {
	// TODO
}

// global returns the global symbol for name, making it if needed.
func (r *resolver) global(name string) *Symbol {
	// TODO
	panic("TODO")
}

func (r *resolver) block(b *ast.Block) {
	// TODO
}

func (r *resolver) stmt(s ast.Stmt) {
	switch s.(type) {
	case *ast.Local:
	case *ast.LocalFunc:
	case *ast.FuncDecl:
	case *ast.Assign:
	case *ast.CompoundAssign:
	case *ast.CallStmt:
	case *ast.Do:
	case *ast.While:
	case *ast.Repeat:
	case *ast.If:
	case *ast.NumericFor:
	case *ast.GenericFor:
	case *ast.Return:
	case *ast.Break, *ast.Continue:
	}
}

func (r *resolver) expr(e ast.Expr) {
	switch e.(type) {
	case *ast.Nil, *ast.Bool, *ast.Number, *ast.String:
	case *ast.Vararg:
	case *ast.Ident:
	case *ast.Field:
	case *ast.Index:
	case *ast.Call:
	case *ast.MethodCall:
	case *ast.Unary:
	case *ast.Binary:
	case *ast.Paren:
	case *ast.Func:
	case *ast.Table:
	case *ast.IfExpr:
	case *ast.Interp:
	}
}

// funcBody resolves a function's params and body in a new function scope.
// method is true for function a:b(), which gets an implicit self.
func (r *resolver) funcBody(f *ast.FuncBody, method bool) {
	// TODO
}
