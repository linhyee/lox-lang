package stdlib

import (
	"fmt"
	"os"
	"path/filepath"
)

func OS() Module {
	return Module{
		"PathSeparator": string(os.PathSeparator),
		"abs":           Function("os.abs", 1, osAbs),
		"args":          Function("os.args", 0, osArgs),
		"base":          Function("os.base", 1, osBase),
		"clean":         Function("os.clean", 1, osClean),
		"cwd":           Function("os.cwd", 0, osCwd),
		"dir":           Function("os.dir", 1, osDir),
		"exists":        Function("os.exists", 1, osExists),
		"ext":           Function("os.ext", 1, osExt),
		"getenv":        Function("os.getenv", 1, osGetenv),
		"isAbs":         Function("os.isAbs", 1, osIsAbs),
		"isDir":         Function("os.isDir", 1, osIsDir),
		"isFile":        Function("os.isFile", 1, osIsFile),
		"join":          Function("os.join", -1, osJoin),
		"mkdirAll":      Function("os.mkdirAll", 1, osMkdirAll),
		"readFile":      Function("os.readFile", 1, osReadFile),
		"remove":        Function("os.remove", 1, osRemove),
		"setenv":        Function("os.setenv", 2, osSetenv),
		"tempDir":       Function("os.tempDir", 0, osTempDir),
		"unsetenv":      Function("os.unsetenv", 1, osUnsetenv),
		"writeFile":     Function("os.writeFile", 2, osWriteFile),
	}
}

func osAbs(host Host, args []Value) (Value, error) {
	path, err := stringArg(args, 0, "path")
	if err != nil {
		return nil, err
	}
	return filepath.Abs(path)
}

func osArgs(host Host, args []Value) (Value, error) {
	values := host.Args()
	items := make([]Value, len(values))
	for i, value := range values {
		items[i] = value
	}
	return host.NewList(items), nil
}

func osBase(host Host, args []Value) (Value, error) {
	path, err := stringArg(args, 0, "path")
	if err != nil {
		return nil, err
	}
	return filepath.Base(path), nil
}

func osClean(host Host, args []Value) (Value, error) {
	path, err := stringArg(args, 0, "path")
	if err != nil {
		return nil, err
	}
	return filepath.Clean(path), nil
}

func osCwd(host Host, args []Value) (Value, error) {
	return os.Getwd()
}

func osDir(host Host, args []Value) (Value, error) {
	path, err := stringArg(args, 0, "path")
	if err != nil {
		return nil, err
	}
	return filepath.Dir(path), nil
}

func osExists(host Host, args []Value) (Value, error) {
	path, err := stringArg(args, 0, "path")
	if err != nil {
		return nil, err
	}
	_, statErr := os.Stat(path)
	if statErr == nil {
		return true, nil
	}
	if os.IsNotExist(statErr) {
		return false, nil
	}
	return nil, statErr
}

func osExt(host Host, args []Value) (Value, error) {
	path, err := stringArg(args, 0, "path")
	if err != nil {
		return nil, err
	}
	return filepath.Ext(path), nil
}

func osGetenv(host Host, args []Value) (Value, error) {
	key, err := stringArg(args, 0, "key")
	if err != nil {
		return nil, err
	}
	return os.Getenv(key), nil
}

func osIsAbs(host Host, args []Value) (Value, error) {
	path, err := stringArg(args, 0, "path")
	if err != nil {
		return nil, err
	}
	return filepath.IsAbs(path), nil
}

func osIsDir(host Host, args []Value) (Value, error) {
	info, err := statArg(args)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return false, nil
	}
	return info.IsDir(), nil
}

func osIsFile(host Host, args []Value) (Value, error) {
	info, err := statArg(args)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return false, nil
	}
	return info.Mode().IsRegular(), nil
}

func osJoin(host Host, args []Value) (Value, error) {
	if len(args) == 0 {
		return "", nil
	}
	parts := make([]string, len(args))
	for i := range args {
		part, err := stringArg(args, i, fmt.Sprintf("path part %d", i+1))
		if err != nil {
			return nil, err
		}
		parts[i] = part
	}
	return filepath.Join(parts...), nil
}

func osMkdirAll(host Host, args []Value) (Value, error) {
	path, err := stringArg(args, 0, "path")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, err
	}
	return true, nil
}

func osReadFile(host Host, args []Value) (Value, error) {
	path, err := stringArg(args, 0, "path")
	if err != nil {
		return nil, err
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}

func osRemove(host Host, args []Value) (Value, error) {
	path, err := stringArg(args, 0, "path")
	if err != nil {
		return nil, err
	}
	if err := os.Remove(path); err != nil {
		return nil, err
	}
	return true, nil
}

func osSetenv(host Host, args []Value) (Value, error) {
	key, err := stringArg(args, 0, "key")
	if err != nil {
		return nil, err
	}
	value, err := stringArg(args, 1, "value")
	if err != nil {
		return nil, err
	}
	if err := os.Setenv(key, value); err != nil {
		return nil, err
	}
	return true, nil
}

func osTempDir(host Host, args []Value) (Value, error) {
	return os.TempDir(), nil
}

func osUnsetenv(host Host, args []Value) (Value, error) {
	key, err := stringArg(args, 0, "key")
	if err != nil {
		return nil, err
	}
	if err := os.Unsetenv(key); err != nil {
		return nil, err
	}
	return true, nil
}

func osWriteFile(host Host, args []Value) (Value, error) {
	path, err := stringArg(args, 0, "path")
	if err != nil {
		return nil, err
	}
	content, err := stringArg(args, 1, "content")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return nil, err
	}
	return true, nil
}

func statArg(args []Value) (os.FileInfo, error) {
	path, err := stringArg(args, 0, "path")
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err == nil {
		return info, nil
	}
	if os.IsNotExist(err) {
		return nil, nil
	}
	return nil, err
}
