package glox

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ModuleState int

const (
	ModuleLoading ModuleState = iota
	ModuleReady
	ModuleExecuting
	ModuleInitialized
	ModuleFailed
)

type Module struct {
	Path    string
	Env     *Environment
	Exports map[string]struct{}
	Closure *Closure
	State   ModuleState
	Native  bool
}

func NewModule(path string) *Module {
	return &Module{
		Path:    path,
		Env:     NewEnvironment(),
		Exports: map[string]struct{}{},
		State:   ModuleLoading,
	}
}

func (m *Module) String() string {
	return "<module " + m.Path + ">"
}

func (m *Module) Export(name string) {
	m.Exports[name] = struct{}{}
}

func (m *Module) IsExported(name string) bool {
	_, ok := m.Exports[name]
	return ok
}

type ModuleLoader struct {
	vm            *VM
	rootDir       string
	searchPaths   []string
	loaded        map[string]*Module
	loadingStack  []string
	nativeModules map[string]*Module
}

func NewModuleLoader(vm *VM, rootDir string, searchPaths []string) *ModuleLoader {
	if rootDir == "" {
		rootDir, _ = os.Getwd()
	}
	absRoot, err := filepath.Abs(rootDir)
	if err == nil {
		rootDir = absRoot
	}
	return &ModuleLoader{
		vm:            vm,
		rootDir:       filepath.Clean(rootDir),
		searchPaths:   append([]string(nil), searchPaths...),
		loaded:        map[string]*Module{},
		nativeModules: map[string]*Module{},
	}
}

func (l *ModuleLoader) RegisterNativeModule(name string, exports map[string]Value) *Module {
	module := NewModule(name)
	module.Native = true
	module.State = ModuleInitialized
	for key, value := range exports {
		module.Env.Define(key, value, true)
		module.Export(key)
	}
	l.nativeModules[name] = module
	l.loaded["native:"+name] = module
	return module
}

func (l *ModuleLoader) Import(request string, caller *Module, line int) (*Module, error) {
	if module, ok := l.nativeModules[request]; ok {
		return module, nil
	}

	resolved, err := l.Resolve(request, caller)
	if err != nil {
		return nil, err
	}

	if module, ok := l.loaded[resolved]; ok {
		switch module.State {
		case ModuleInitialized:
			return module, nil
		case ModuleLoading, ModuleReady, ModuleExecuting:
			return nil, fmt.Errorf("circular import detected: %s", l.cycleMessage(resolved))
		case ModuleFailed:
			return nil, fmt.Errorf("failed to import '%s'", resolved)
		}
	}

	source, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("could not read module '%s': %w", resolved, err)
	}

	module := NewModule(resolved)
	module.State = ModuleLoading
	l.loaded[resolved] = module
	l.loadingStack = append(l.loadingStack, resolved)
	defer func() {
		if len(l.loadingStack) > 0 && l.loadingStack[len(l.loadingStack)-1] == resolved {
			l.loadingStack = l.loadingStack[:len(l.loadingStack)-1]
		}
	}()

	function, err := Compile(string(source), resolved, module, l.vm.Diagnostics)
	if err != nil {
		module.State = ModuleFailed
		return nil, err
	}
	l.vm.disassembleCompiledFunction(function)
	module.Closure = NewClosure(function, module)
	module.State = ModuleReady

	if err := l.vm.executeModule(module); err != nil {
		module.State = ModuleFailed
		return nil, err
	}
	return module, nil
}

func (l *ModuleLoader) Resolve(request string, caller *Module) (string, error) {
	candidates := l.candidates(request, caller)
	for _, candidate := range candidates {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		abs = filepath.Clean(abs)
		if _, err := os.Stat(abs); err == nil {
			if real, err := filepath.EvalSymlinks(abs); err == nil {
				abs = real
			}
			return filepath.Clean(abs), nil
		}
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("empty import path")
	}
	abs, err := filepath.Abs(candidates[0])
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func (l *ModuleLoader) candidates(request string, caller *Module) []string {
	if request == "" {
		return nil
	}
	request = filepath.FromSlash(request)
	if strings.HasPrefix(request, string(filepath.Separator)) ||
		strings.HasPrefix(request, "/") || strings.HasPrefix(request, `\`) {
		trimmed := strings.TrimLeft(request, `/\`)
		return []string{filepath.Join(l.rootDir, trimmed)}
	}

	if filepath.VolumeName(request) != "" || filepath.IsAbs(request) {
		return []string{request}
	}

	if strings.HasPrefix(request, "."+string(filepath.Separator)) ||
		strings.HasPrefix(request, ".."+string(filepath.Separator)) ||
		strings.HasPrefix(request, "./") || strings.HasPrefix(request, "../") {
		base := l.rootDir
		if caller != nil && caller.Path != "" && caller.Path != "repl" && !caller.Native {
			base = filepath.Dir(caller.Path)
		}
		return []string{filepath.Join(base, request)}
	}

	var candidates []string
	if caller != nil && caller.Path != "" && caller.Path != "repl" && !caller.Native {
		candidates = append(candidates, filepath.Join(filepath.Dir(caller.Path), request))
	}
	for _, root := range l.searchPaths {
		if root == "" {
			continue
		}
		candidates = append(candidates, filepath.Join(root, request))
	}
	candidates = append(candidates, filepath.Join(l.rootDir, request))
	return candidates
}

func (l *ModuleLoader) cycleMessage(path string) string {
	start := -1
	for i, item := range l.loadingStack {
		if item == path {
			start = i
			break
		}
	}
	if start < 0 {
		stack := append(append([]string(nil), l.loadingStack...), path)
		return strings.Join(stack, " -> ")
	}
	stack := append(append([]string(nil), l.loadingStack[start:]...), path)
	return strings.Join(stack, " -> ")
}
