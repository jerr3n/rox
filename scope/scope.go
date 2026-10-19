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
	var res *Symbol
	for cur := s; cur != nil; cur = cur.Parent {
		res = cur.names[name]
		if res != nil {
			return res, cur
		}
	}
	return nil, nil
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
	r := &resolver{info: &Info{
		Symbols: map[*ast.Ident]*Symbol{},
		Globals: map[string]*Symbol{},
	}}
	r.block(file)
	return r.info
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
	s := Scope{
		Parent: r.scope,
	}
	if fn != nil {
		s.Func = fn
	} else if s.Parent != nil {
		s.Func = s.Parent.Func
	}
	r.scope = &s
}

// pop closes the current scope.
func (r *resolver) pop() {
	r.scope = r.scope.Parent
}

// declare makes a new symbol for id in the current scope.
func (r *resolver) declare(id *ast.Ident, kind Kind) {
	sym := r.scope.Declare(id.Name, kind, id)
	r.info.Symbols[id] = sym
}

// use resolves id and records it as a read and/or write.
func (r *resolver) use(id *ast.Ident, acc access) {
	sym, scp := r.scope.Lookup(id.Name)
	if sym == nil {
		sym = r.global(id.Name)
	}
	r.info.Symbols[id] = sym
	switch acc {
	case read:
		sym.Reads = append(sym.Reads, id)
	case write:
		sym.Writes = append(sym.Writes, id)
	case readWrite:
		sym.Writes = append(sym.Writes, id)
		sym.Reads = append(sym.Reads, id)
	}
	if scp != nil && scp.Func != r.scope.Func {
		sym.Captured = true
	}
}

// global returns the global symbol for name, making it if needed.
func (r *resolver) global(name string) *Symbol {
	t := r.info.Globals[name]
	if t != nil {
		return t
	}
	sym := Symbol{
		Name:     name,
		Kind:     Global,
		Decl:     nil,
		Func:     nil,
		Reads:    nil,
		Writes:   nil,
		Captured: false,
	}
	r.info.Globals[name] = &sym
	return &sym
}

// block resolves b in a new scope.
func (r *resolver) block(b *ast.Block) {
	r.push(nil)
	r.stmts(b)
	r.pop()
}

// stmts resolves b's statements in the current scope, for the places that
// push their own scope first (loops, repeat, function bodies).
func (r *resolver) stmts(b *ast.Block) {
	if b == nil {
		return
	}
	for _, s := range b.Stmts {
		r.stmt(s)
	}
}

func (r *resolver) stmt(s ast.Stmt) {
	switch s := s.(type) {
	case *ast.Local:
		// values first: in `local x = x` the right side is the outer x
		r.exprs(s.Values)
		for _, b := range s.Names {
			r.declare(b.Name, Local)
		}
	case *ast.LocalFunc:
		// name first, so the body can call itself
		r.declare(s.Name, LocalFunc)
		r.funcBody(s.Body, false)
	case *ast.FuncDecl:
		// only Path[0] is a variable; `function f()` assigns it, and
		// `function a.b()` / `function a:b()` only read it
		path := s.Name.Path
		if len(path) == 1 && s.Name.Method == nil {
			r.use(path[0], write)
		} else {
			r.use(path[0], read)
		}
		r.funcBody(s.Body, s.Name.Method != nil)
	case *ast.Assign:
		r.exprs(s.Values)
		for _, t := range s.Targets {
			r.target(t, write)
		}
	case *ast.CompoundAssign:
		r.expr(s.Value)
		r.target(s.Target, readWrite)
	case *ast.CallStmt:
		r.expr(s.Call)
	case *ast.Do:
		r.block(s.Body)
	case *ast.While:
		r.expr(s.Cond)
		r.block(s.Body)
	case *ast.Repeat:
		// the condition can see the body's locals, so it's resolved
		// before the body's scope closes
		r.push(nil)
		r.stmts(s.Body)
		r.expr(s.Cond)
		r.pop()
	case *ast.If:
		r.expr(s.Cond)
		r.block(s.Then)
		for _, e := range s.ElseIfs {
			r.expr(e.Cond)
			r.block(e.Body)
		}
		if s.Else != nil {
			r.block(s.Else)
		}
	case *ast.NumericFor:
		// the range is outside the loop; the variable is inside it
		r.expr(s.Start)
		r.expr(s.Stop)
		r.expr(s.Step)
		r.push(nil)
		r.declare(s.Var.Name, ForVar)
		r.stmts(s.Body)
		r.pop()
	case *ast.GenericFor:
		r.exprs(s.Exprs)
		r.push(nil)
		for _, b := range s.Vars {
			r.declare(b.Name, ForVar)
		}
		r.stmts(s.Body)
		r.pop()
	case *ast.Return:
		r.exprs(s.Values)
	case *ast.Break, *ast.Continue:
	}
}

// target resolves an assignment target. Only a bare name is written to; in
// t.x and t[k], t and k are just read.
func (r *resolver) target(e ast.Expr, acc access) {
	if id, ok := e.(*ast.Ident); ok {
		r.use(id, acc)
		return
	}
	r.expr(e)
}

func (r *resolver) exprs(es []ast.Expr) {
	for _, e := range es {
		r.expr(e)
	}
}

func (r *resolver) expr(e ast.Expr) {
	switch e := e.(type) {
	case nil:
	case *ast.Nil, *ast.Bool, *ast.Number, *ast.String:
	case *ast.Vararg:
	case *ast.Ident:
		r.use(e, read)
	case *ast.Field:
		// e.Name is a field name, not a variable
		r.expr(e.X)
	case *ast.Index:
		r.expr(e.X)
		r.expr(e.Key)
	case *ast.Call:
		r.expr(e.Fn)
		r.exprs(e.Args)
	case *ast.MethodCall:
		// e.Method is a method name, not a variable
		r.expr(e.Recv)
		r.exprs(e.Args)
	case *ast.Unary:
		r.expr(e.X)
	case *ast.Binary:
		r.expr(e.L)
		r.expr(e.R)
	case *ast.Paren:
		r.expr(e.X)
	case *ast.Func:
		r.funcBody(e.Body, false)
	case *ast.Table:
		for _, item := range e.Items {
			// item.Name in {name = v} is a key, not a variable
			r.expr(item.Key)
			r.expr(item.Value)
		}
	case *ast.IfExpr:
		r.expr(e.Cond)
		r.expr(e.Then)
		for _, ei := range e.ElseIfs {
			r.expr(ei.Cond)
			r.expr(ei.Then)
		}
		r.expr(e.Else)
	case *ast.Interp:
		r.exprs(e.Exprs)
	}
}

// funcBody resolves a function's params and body in a new function scope.
// method is true for function a:b(), which gets an implicit self.
func (r *resolver) funcBody(f *ast.FuncBody, method bool) {
	if f == nil {
		return
	}
	// params and body share one scope, so a body local shadows a param
	r.push(f)
	if method {
		r.scope.Declare("self", Self, nil)
	}
	for _, p := range f.Params {
		r.declare(p.Name, Param)
	}
	r.stmts(f.Body)
	r.pop()
}
