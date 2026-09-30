package stdlib

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func String() Module {
	return Module{
		"contains":   Function("string.contains", 2, stringContains),
		"fields":     Function("string.fields", 1, stringFields),
		"hasPrefix":  Function("string.hasPrefix", 2, stringHasPrefix),
		"hasSuffix":  Function("string.hasSuffix", 2, stringHasSuffix),
		"index":      Function("string.index", 2, stringIndex),
		"join":       Function("string.join", 2, stringJoin),
		"lastIndex":  Function("string.lastIndex", 2, stringLastIndex),
		"len":        Function("string.len", 1, stringLen),
		"lower":      Function("string.lower", 1, stringLower),
		"repeat":     Function("string.repeat", 2, stringRepeat),
		"replace":    Function("string.replace", 4, stringReplace),
		"split":      Function("string.split", 2, stringSplit),
		"slice":      Function("string.slice", 3, stringSlice),
		"trim":       Function("string.trim", 2, stringTrim),
		"trimPrefix": Function("string.trimPrefix", 2, stringTrimPrefix),
		"trimSpace":  Function("string.trimSpace", 1, stringTrimSpace),
		"trimSuffix": Function("string.trimSuffix", 2, stringTrimSuffix),
		"upper":      Function("string.upper", 1, stringUpper),
	}
}

func stringContains(host Host, args []Value) (Value, error) {
	text, needle, err := twoStrings(args, "text", "substr")
	if err != nil {
		return nil, err
	}
	return strings.Contains(text, needle), nil
}

func stringFields(host Host, args []Value) (Value, error) {
	text, err := stringArg(args, 0, "text")
	if err != nil {
		return nil, err
	}
	return listFromStrings(host, strings.Fields(text)), nil
}

func stringHasPrefix(host Host, args []Value) (Value, error) {
	text, prefix, err := twoStrings(args, "text", "prefix")
	if err != nil {
		return nil, err
	}
	return strings.HasPrefix(text, prefix), nil
}

func stringHasSuffix(host Host, args []Value) (Value, error) {
	text, suffix, err := twoStrings(args, "text", "suffix")
	if err != nil {
		return nil, err
	}
	return strings.HasSuffix(text, suffix), nil
}

func stringIndex(host Host, args []Value) (Value, error) {
	text, needle, err := twoStrings(args, "text", "substr")
	if err != nil {
		return nil, err
	}
	return int64(runeIndex(text, needle)), nil
}

func stringJoin(host Host, args []Value) (Value, error) {
	items, ok := host.ListItems(args[0])
	if !ok {
		return nil, fmt.Errorf("items must be a list")
	}
	sep, err := stringArg(args, 1, "separator")
	if err != nil {
		return nil, err
	}
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = host.Stringify(item)
	}
	return strings.Join(parts, sep), nil
}

func stringLastIndex(host Host, args []Value) (Value, error) {
	text, needle, err := twoStrings(args, "text", "substr")
	if err != nil {
		return nil, err
	}
	return int64(runeLastIndex(text, needle)), nil
}

func stringLen(host Host, args []Value) (Value, error) {
	text, err := stringArg(args, 0, "text")
	if err != nil {
		return nil, err
	}
	return int64(utf8.RuneCountInString(text)), nil
}

func stringLower(host Host, args []Value) (Value, error) {
	text, err := stringArg(args, 0, "text")
	if err != nil {
		return nil, err
	}
	return strings.ToLower(text), nil
}

func stringRepeat(host Host, args []Value) (Value, error) {
	text, err := stringArg(args, 0, "text")
	if err != nil {
		return nil, err
	}
	count, err := intArg(host, args, 1, "count")
	if err != nil {
		return nil, err
	}
	if count < 0 || int64(int(count)) != count {
		return nil, fmt.Errorf("count out of range")
	}
	return strings.Repeat(text, int(count)), nil
}

func stringReplace(host Host, args []Value) (Value, error) {
	text, old, err := twoStrings(args, "text", "old")
	if err != nil {
		return nil, err
	}
	newValue, err := stringArg(args, 2, "new")
	if err != nil {
		return nil, err
	}
	count, err := intArg(host, args, 3, "count")
	if err != nil {
		return nil, err
	}
	if int64(int(count)) != count {
		return nil, fmt.Errorf("count out of range")
	}
	return strings.Replace(text, old, newValue, int(count)), nil
}

func stringSplit(host Host, args []Value) (Value, error) {
	text, sep, err := twoStrings(args, "text", "separator")
	if err != nil {
		return nil, err
	}
	return listFromStrings(host, strings.Split(text, sep)), nil
}

func stringSlice(host Host, args []Value) (Value, error) {
	text, err := stringArg(args, 0, "text")
	if err != nil {
		return nil, err
	}
	start, err := intArg(host, args, 1, "start")
	if err != nil {
		return nil, err
	}
	end, err := intArg(host, args, 2, "end")
	if err != nil {
		return nil, err
	}
	runes := []rune(text)
	if start < 0 || end < start || end > int64(len(runes)) {
		return nil, fmt.Errorf("slice index out of range")
	}
	return string(runes[start:end]), nil
}

func stringTrim(host Host, args []Value) (Value, error) {
	text, cutset, err := twoStrings(args, "text", "cutset")
	if err != nil {
		return nil, err
	}
	return strings.Trim(text, cutset), nil
}

func stringTrimPrefix(host Host, args []Value) (Value, error) {
	text, prefix, err := twoStrings(args, "text", "prefix")
	if err != nil {
		return nil, err
	}
	return strings.TrimPrefix(text, prefix), nil
}

func stringTrimSpace(host Host, args []Value) (Value, error) {
	text, err := stringArg(args, 0, "text")
	if err != nil {
		return nil, err
	}
	return strings.TrimSpace(text), nil
}

func stringTrimSuffix(host Host, args []Value) (Value, error) {
	text, suffix, err := twoStrings(args, "text", "suffix")
	if err != nil {
		return nil, err
	}
	return strings.TrimSuffix(text, suffix), nil
}

func stringUpper(host Host, args []Value) (Value, error) {
	text, err := stringArg(args, 0, "text")
	if err != nil {
		return nil, err
	}
	return strings.ToUpper(text), nil
}

func twoStrings(args []Value, firstName, secondName string) (string, string, error) {
	first, err := stringArg(args, 0, firstName)
	if err != nil {
		return "", "", err
	}
	second, err := stringArg(args, 1, secondName)
	if err != nil {
		return "", "", err
	}
	return first, second, nil
}

func runeIndex(text, needle string) int {
	byteIndex := strings.Index(text, needle)
	if byteIndex < 0 {
		return -1
	}
	return utf8.RuneCountInString(text[:byteIndex])
}

func runeLastIndex(text, needle string) int {
	byteIndex := strings.LastIndex(text, needle)
	if byteIndex < 0 {
		return -1
	}
	return utf8.RuneCountInString(text[:byteIndex])
}
