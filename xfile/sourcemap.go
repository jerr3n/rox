package xfile

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Instance struct {
	Name      string      `json:"name"`
	ClassName string      `json:"className"`
	FilePaths []string    `json:"filePaths"`
	Children  []*Instance `json:"children"`

	Parent *Instance `json:"-"`
}

func (inst *Instance) Child(name string) *Instance {
	var found *Instance
	for _, c := range inst.Children {
		if c.Name != name {
			continue
		}
		if found != nil {
			return nil
		}
		found = c
	}
	return found
}

func (inst *Instance) Source() string {
	for _, p := range inst.FilePaths {
		if isLuau(p) {
			return p
		}
	}
	return ""
}

func (inst *Instance) FullName() string {
	if inst.Parent == nil {
		return ""
	}
	if inst.Parent.Parent == nil {
		return inst.Name
	}
	return inst.Parent.FullName() + "." + inst.Name
}

type Sourcemap struct {
	Root *Instance

	files map[string]*Instance
	dir   string // the filePaths in the JSON are relative to this
}

func LoadSourcemap(path string) (*Sourcemap, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseSourcemap(f, filepath.Dir(path))
}

func ParseSourcemap(r io.Reader, dir string) (*Sourcemap, error) {
	var root Instance
	if err := json.NewDecoder(r).Decode(&root); err != nil {
		return nil, fmt.Errorf("sourcemap: %w", err)
	}
	sm := &Sourcemap{Root: &root, files: map[string]*Instance{}, dir: dir}
	sm.index(&root, nil, dir)
	return sm, nil
}

func (sm *Sourcemap) index(inst, parent *Instance, dir string) {
	inst.Parent = parent
	if src := inst.Source(); src != "" {
		sm.files[joinPath(dir, src)] = inst
	}
	for _, c := range inst.Children {
		sm.index(c, inst, dir)
	}
}

func (sm *Sourcemap) File(path string) *Instance {
	return sm.files[filepath.Clean(path)]
}

func (sm *Sourcemap) Files() []string {
	out := make([]string, 0, len(sm.files))
	for p := range sm.files {
		out = append(out, p)
	}
	return out
}

func joinPath(dir, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(dir, p)
}

func isLuau(p string) bool {
	return strings.HasSuffix(p, ".luau") || strings.HasSuffix(p, ".lua")
}
