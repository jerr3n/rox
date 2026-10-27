package xfile

import (
	"fmt"
	"strings"

	"github.com/jerr3n/rox/ast"
)

// ---------------------------------------------------------------------------
// Dumper
//
// Prints instances as "ClassName Name", one per line, with children indented
// under their parent and the backing Luau file (if any) on the right:
//
//	DataModel demo
//	  ReplicatedStorage ReplicatedStorage
//	    Folder Shared
//	      ModuleScript Signal  src/shared/Signal.luau
//
// Requires print what they resolved to, then the argument's tree from
// ast.Dump, so an unresolved one shows the shape Resolve gave up on:
//
//	Require → ReplicatedStorage.Shared.Signal  src/shared/Signal.luau
//	  Field .Signal
//	    ...
// ---------------------------------------------------------------------------

type dumper struct {
	b strings.Builder
}

// Dump prints a *Sourcemap (its whole tree), an *Instance and everything
// under it, one Require, a []Require, one *File, or a []*File from Follow.
func Dump(v any) string {
	var p dumper
	switch v := v.(type) {
	case *Sourcemap:
		if v == nil {
			p.line(0, "<nil sourcemap>")
			break
		}
		p.instance(v.Root, 0)
	case *Instance:
		p.instance(v, 0)
	case Require:
		p.require(v, 0)
	case []Require:
		if len(v) == 0 {
			p.line(0, "<no requires>")
		}
		for _, r := range v {
			p.require(r, 0)
		}
	case *File:
		p.file(v, 0)
	case []*File:
		for _, f := range v {
			p.file(f, 0)
		}
	case nil:
		p.line(0, "<nil>")
	default:
		p.line(0, "<can't dump %T>", v)
	}
	return p.b.String()
}

func (p *dumper) line(depth int, format string, args ...any) {
	p.b.WriteString(strings.Repeat("  ", depth))
	fmt.Fprintf(&p.b, format, args...)
	p.b.WriteByte('\n')
}

// sourceSuffix is "  path" for an instance backed by a Luau file, else "".
func sourceSuffix(inst *Instance) string {
	if src := inst.Source(); src != "" {
		return "  " + src
	}
	return ""
}

func (p *dumper) instance(inst *Instance, depth int) {
	if inst == nil {
		p.line(depth, "<nil instance>")
		return
	}
	p.line(depth, "%s %s%s", inst.ClassName, inst.Name, sourceSuffix(inst))
	for _, c := range inst.Children {
		p.instance(c, depth+1)
	}
}

func (p *dumper) file(f *File, depth int) {
	if f == nil {
		p.line(depth, "<nil file>")
		return
	}
	p.line(depth, "%s", f.Path)
	if f.Err != nil {
		p.line(depth+1, "error: %v", f.Err)
		return
	}
	if len(f.Requires) == 0 {
		p.line(depth+1, "<no requires>")
	}
	for _, r := range f.Requires {
		p.require(r, depth+1)
	}
}

func (p *dumper) require(r Require, depth int) {
	var to []string
	if r.Target != nil {
		to = append(to, r.Target.FullName())
	}
	if r.Path != "" {
		to = append(to, r.Path)
	}
	if len(to) == 0 {
		p.line(depth, "Require ✗ unresolved")
	} else {
		p.line(depth, "Require → %s", strings.Join(to, "  "))
	}
	if r.Call == nil || len(r.Call.Args) == 0 {
		return
	}
	// ast.Dump starts at column 0, so shift its lines under this one.
	for _, l := range strings.Split(strings.TrimSuffix(ast.Dump(r.Call.Args[0]), "\n"), "\n") {
		p.line(depth+1, "%s", l)
	}
}
