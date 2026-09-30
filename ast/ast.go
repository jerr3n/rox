package ast

// ---------------------------------------------------------------------------
// Positions
// ---------------------------------------------------------------------------

// Pos is a byte offset into the source. Resolve it to line:col lazily using
// the line-start table your lexer builds. (If you go with go/token instead,
// swap this for `type Pos = token.Pos`.)
type Pos int

// Span is embedded in every node to give it Pos() and End().
type Span struct {
	Start Pos // first byte of the node
	Stop  Pos // one past the last byte
}

func (s Span) Pos() Pos { return s.Start }
func (s Span) End() Pos { return s.Stop }

// ---------------------------------------------------------------------------
// Node categories
// ---------------------------------------------------------------------------

// Node is anything in the tree.
type Node interface {
	Pos() Pos
	End() Pos
}

// Expr is anything that produces a value.
type Expr interface {
	Node
	exprNode()
}

// Stmt is anything that can appear in a Block.
type Stmt interface {
	Node
	stmtNode()
}

// ---------------------------------------------------------------------------
// Operators
//
// These are separate from the lexer's token kinds on purpose: the AST doesn't
// depend on the lexer, and the parser maps tokens to ops in one place.
// ---------------------------------------------------------------------------

type UnaryOp uint8

const (
	OpNeg UnaryOp = iota // -x
	OpNot                // not x
	OpLen                // #x
)

type BinaryOp uint8

const (
	OpAdd      BinaryOp = iota // +
	OpSub                      // -
	OpMul                      // *
	OpDiv                      // /
	OpFloorDiv                 // //
	OpMod                      // %
	OpPow                      // ^
	OpConcat                   // ..
	OpEq                       // ==
	OpNe                       // ~=
	OpLt                       // <
	OpLe                       // <=
	OpGt                       // >
	OpGe                       // >=
	OpAnd                      // and
	OpOr                       // or
)

// ---------------------------------------------------------------------------
// Expressions
// ---------------------------------------------------------------------------

type (
	// nil
	Nil struct{ Span }

	// true / false
	Bool struct {
		Span
		Value bool
	}

	// 42, 0xFF, 1_000, .5, 1e10
	Number struct {
		Span
		Raw string // source text; parse to float64 when you need the value
	}

	// "hi", 'hi', [[hi]]
	String struct {
		Span
		Value string // escapes already decoded
	}

	// ...
	Vararg struct{ Span }

	// foo
	Ident struct {
		Span
		Name string
		// Sym *Symbol // filled in later by the scope resolver
	}

	// x.name
	Field struct {
		Span
		X    Expr
		Name *Ident
	}

	// x[key]
	Index struct {
		Span
		X   Expr
		Key Expr
	}

	// f(a, b), f"str", f{...}  (sugar forms produce a one-element Args)
	Call struct {
		Span
		Fn   Expr
		Args []Expr
	}

	// obj:method(a, b)
	MethodCall struct {
		Span
		Recv   Expr
		Method *Ident
		Args   []Expr
	}

	// -x, not x, #x
	Unary struct {
		Span
		Op UnaryOp
		X  Expr
	}

	// a + b, a and b, ...
	Binary struct {
		Span
		Op BinaryOp
		L  Expr
		R  Expr
	}

	// (x) — kept because parens truncate multiple returns to one value
	Paren struct {
		Span
		X Expr
	}

	// function(a, b) ... end
	Func struct {
		Span
		Body *FuncBody
	}

	// {1, x = 2, [k] = 3}
	Table struct {
		Span
		Items []*TableItem
	}

	// if a then b elseif c then d else e
	IfExpr struct {
		Span
		Cond    Expr
		Then    Expr
		ElseIfs []*ElseIfExpr
		Else    Expr // always present in expression form
	}

	// `hello {name}!`
	// Parts has exactly len(Exprs)+1 entries: Parts[0], Exprs[0], Parts[1], ...
	Interp struct {
		Span
		Parts []string
		Exprs []Expr
	}
)

// ElseIfExpr is one `elseif c then v` arm of an IfExpr.
type ElseIfExpr struct {
	Span
	Cond Expr
	Then Expr
}

type TableItemKind uint8

const (
	ItemPositional TableItemKind = iota // value
	ItemNamed                           // name = value
	ItemKeyed                           // [key] = value
)

// TableItem is one entry in a table constructor.
type TableItem struct {
	Span
	Kind  TableItemKind
	Name  *Ident // ItemNamed only
	Key   Expr   // ItemKeyed only
	Value Expr
}

func (*Nil) exprNode()        {}
func (*Bool) exprNode()       {}
func (*Number) exprNode()     {}
func (*String) exprNode()     {}
func (*Vararg) exprNode()     {}
func (*Ident) exprNode()      {}
func (*Field) exprNode()      {}
func (*Index) exprNode()      {}
func (*Call) exprNode()       {}
func (*MethodCall) exprNode() {}
func (*Unary) exprNode()      {}
func (*Binary) exprNode()     {}
func (*Paren) exprNode()      {}
func (*Func) exprNode()       {}
func (*Table) exprNode()      {}
func (*IfExpr) exprNode()     {}
func (*Interp) exprNode()     {}

// ---------------------------------------------------------------------------
// Statements
// ---------------------------------------------------------------------------

type (
	// local a: number, b = 1, 2
	Local struct {
		Span
		Names  []*Binding
		Values []Expr // may be empty
	}

	// local function f() end
	LocalFunc struct {
		Span
		Name *Ident
		Body *FuncBody
	}

	// function a.b.c:d() end
	FuncDecl struct {
		Span
		Name *FuncName
		Body *FuncBody
	}

	// a, t.x, t[k] = 1, 2, 3
	// Targets must be *Ident, *Field, or *Index (the parser checks this).
	Assign struct {
		Span
		Targets []Expr
		Values  []Expr
	}

	// x += 1  (Op is one of Add..Concat)
	CompoundAssign struct {
		Span
		Op     BinaryOp
		Target Expr
		Value  Expr
	}

	// print("hi") — a *Call or *MethodCall used as a statement
	CallStmt struct {
		Span
		Call Expr
	}

	// do ... end
	Do struct {
		Span
		Body *Block
	}

	// while cond do ... end
	While struct {
		Span
		Cond Expr
		Body *Block
	}

	// repeat ... until cond
	// Note: Cond is inside Body's scope (it can see Body's locals).
	Repeat struct {
		Span
		Body *Block
		Cond Expr
	}

	// if a then ... elseif b then ... else ... end
	If struct {
		Span
		Cond    Expr
		Then    *Block
		ElseIfs []*ElseIf
		Else    *Block // nil if there's no else
	}

	// for i = start, stop, step do ... end
	NumericFor struct {
		Span
		Var   *Binding
		Start Expr
		Stop  Expr
		Step  Expr // nil if omitted
		Body  *Block
	}

	// for k, v in pairs(t) do ... end
	GenericFor struct {
		Span
		Vars  []*Binding
		Exprs []Expr
		Body  *Block
	}

	// return a, b
	Return struct {
		Span
		Values []Expr
	}

	// break
	Break struct{ Span }

	// continue
	Continue struct{ Span }
)

// ElseIf is one `elseif c then ...` arm of an If statement.
type ElseIf struct {
	Span
	Cond Expr
	Body *Block
}

func (*Local) stmtNode()          {}
func (*LocalFunc) stmtNode()      {}
func (*FuncDecl) stmtNode()       {}
func (*Assign) stmtNode()         {}
func (*CompoundAssign) stmtNode() {}
func (*CallStmt) stmtNode()       {}
func (*Do) stmtNode()             {}
func (*While) stmtNode()          {}
func (*Repeat) stmtNode()         {}
func (*If) stmtNode()             {}
func (*NumericFor) stmtNode()     {}
func (*GenericFor) stmtNode()     {}
func (*Return) stmtNode()         {}
func (*Break) stmtNode()          {}
func (*Continue) stmtNode()       {}

// ---------------------------------------------------------------------------
// Support structs
// ---------------------------------------------------------------------------

// Block is a list of statements. A whole file is a Block, as is every
// function body, loop body, and do/if arm. Each Block starts a new scope.
type Block struct {
	Span
	Stmts []Stmt
}

// Binding is a name being declared (local, for variable, parameter),
// as opposed to an Ident being used. Type annotations are parsed and discarded.
type Binding struct {
	Span
	Name *Ident
}

// FuncBody is everything after the `function` keyword (plus any attributes
// before it). Shared by Func, LocalFunc, and FuncDecl.
type FuncBody struct {
	Span
	Attributes []*Attribute // @native, @checked, ...
	Params     []*Binding
	Vararg     bool // true if the parameter list ends in `...`
	Body       *Block
}

// FuncName is the restricted name in `function a.b.c:d()`.
type FuncName struct {
	Span
	Path   []*Ident // a, b, c (always at least one)
	Method *Ident   // d, or nil if there's no `:`
}

// Attribute is a function attribute like @native.
type Attribute struct {
	Span
	Name string
}
