package xfile

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/jerr3n/rox/ast"
)

type Require struct {
	Call *ast.Call
	// Target is the instance an instance require like require(script.X)
	// resolved to through the sourcemap. It is nil for string requires.
	Target *Instance
	// Path is the file the require loads, or "" if it couldn't be resolved.
	Path string
}

// Requires finds every require(x) in file, which lives at path.
func (sm *Sourcemap) Requires(file *ast.Block, path string) []Require {
	from := sm.File(path)
	var out []Require
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := ast.CallTo(n, "require")
		if !ok || len(call.Args) != 1 {
			return true
		}
		r := Require{Call: call}
		if s, ok := call.Args[0].(*ast.String); ok {
			r.Path = ResolvePath(s.Value, path)
		} else if r.Target = sm.Resolve(call.Args[0], from); r.Target != nil {
			if src := r.Target.Source(); src != "" {
				r.Path = joinPath(sm.dir, src)
			}
		}
		out = append(out, r)
		return true
	})
	return out
}

// ResolvePath resolves a string require like require("./foo") in the file at
// from, following Luau's require-by-string rules:
//
//   - "./x" and "../x" are relative to the directory holding the module. For
//     pkg/init.luau the module is pkg itself, so "./x" is a sibling of pkg.
//   - "@self/x" is inside the module: pkg/x for pkg/init.luau, and a/x for
//     a.luau.
//   - "x" names x.luau, x.lua, x/init.luau or x/init.lua. Exactly one must
//     exist.
//
// Other aliases ("@pkg/x") come from .luaurc, which isn't read yet, so they
// return "".
func ResolvePath(req, from string) string {
	module := strings.TrimSuffix(from, filepath.Ext(from))
	if filepath.Base(module) == "init" {
		module = filepath.Dir(module)
	}
	var p string
	switch {
	case req == "@self" || strings.HasPrefix(req, "@self/"):
		p = filepath.Join(module, strings.TrimPrefix(req, "@self"))
	case strings.HasPrefix(req, "./"), strings.HasPrefix(req, "../"):
		p = filepath.Join(filepath.Dir(module), req)
	default:
		return ""
	}
	found := ""
	for _, c := range []string{p + ".luau", p + ".lua", filepath.Join(p, "init.luau"), filepath.Join(p, "init.lua")} {
		if info, err := os.Stat(c); err != nil || info.IsDir() {
			continue
		}
		if found != "" {
			return ""
		}
		found = c
	}
	return found
}
func (sm *Sourcemap) Resolve(e ast.Expr, from *Instance) *Instance {
	switch e := e.(type) {
	case *ast.Paren:
		return sm.Resolve(e.X, from)
	case *ast.Ident:
		switch e.Name {
		case "script":
			return from
		case "game":
			return sm.Root
		}
	case *ast.Field:
		return member(sm.Resolve(e.X, from), e.Name.Name)
	case *ast.Index:
		key, ok := e.Key.(*ast.String)
		if !ok {
			return nil
		}
		return member(sm.Resolve(e.X, from), key.Value)
	case *ast.MethodCall:
		return sm.resolveMethod(e, from)
	}
	return nil
}

func member(inst *Instance, name string) *Instance {
	if inst == nil {
		return nil
	}
	if name == "Parent" {
		return inst.Parent
	}
	return inst.Child(name)
}

func (sm *Sourcemap) resolveMethod(m *ast.MethodCall, from *Instance) *Instance {
	recv := sm.Resolve(m.Recv, from)
	if recv == nil || len(m.Args) == 0 {
		return nil
	}
	name, ok := m.Args[0].(*ast.String)
	if !ok {
		return nil
	}
	switch m.Method.Name {
	case "WaitForChild":
		return recv.Child(name.Value)
	case "FindFirstChild":
		if len(m.Args) > 1 && !isFalse(m.Args[1]) {
			return nil
		}
		return recv.Child(name.Value)
	case "GetService":
		if recv != sm.Root {
			return nil
		}
		return recv.Child(name.Value)
	}
	return nil
}

func isFalse(e ast.Expr) bool {
	b, ok := e.(*ast.Bool)
	return ok && !b.Value
}
