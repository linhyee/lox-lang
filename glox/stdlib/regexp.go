package stdlib

import (
	"fmt"
	"regexp"
)

func Regexp() Module {
	return Module{
		"find":         Function("regexp.find", 2, regexpFind),
		"findAll":      Function("regexp.findAll", 3, regexpFindAll),
		"findSubmatch": Function("regexp.findSubmatch", 2, regexpFindSubmatch),
		"match":        Function("regexp.match", 2, regexpMatch),
		"quoteMeta":    Function("regexp.quoteMeta", 1, regexpQuoteMeta),
		"replace":      Function("regexp.replace", 3, regexpReplace),
		"split":        Function("regexp.split", 3, regexpSplit),
	}
}

func regexpFind(host Host, args []Value) (Value, error) {
	re, text, err := regexpArgs(args)
	if err != nil {
		return nil, err
	}
	match := re.FindString(text)
	if match == "" && re.FindStringIndex(text) == nil {
		return nil, nil
	}
	return match, nil
}

func regexpFindAll(host Host, args []Value) (Value, error) {
	re, text, err := regexpArgs(args)
	if err != nil {
		return nil, err
	}
	limit, err := intArg(host, args, 2, "limit")
	if err != nil {
		return nil, err
	}
	if int64(int(limit)) != limit {
		return nil, fmt.Errorf("limit out of range")
	}
	return listFromStrings(host, re.FindAllString(text, int(limit))), nil
}

func regexpFindSubmatch(host Host, args []Value) (Value, error) {
	re, text, err := regexpArgs(args)
	if err != nil {
		return nil, err
	}
	matches := re.FindStringSubmatch(text)
	if matches == nil {
		return nil, nil
	}
	return listFromStrings(host, matches), nil
}

func regexpMatch(host Host, args []Value) (Value, error) {
	re, text, err := regexpArgs(args)
	if err != nil {
		return nil, err
	}
	return re.MatchString(text), nil
}

func regexpQuoteMeta(host Host, args []Value) (Value, error) {
	text, err := stringArg(args, 0, "text")
	if err != nil {
		return nil, err
	}
	return regexp.QuoteMeta(text), nil
}

func regexpReplace(host Host, args []Value) (Value, error) {
	re, text, err := regexpArgs(args)
	if err != nil {
		return nil, err
	}
	replacement, err := stringArg(args, 2, "replacement")
	if err != nil {
		return nil, err
	}
	return re.ReplaceAllString(text, replacement), nil
}

func regexpSplit(host Host, args []Value) (Value, error) {
	re, text, err := regexpArgs(args)
	if err != nil {
		return nil, err
	}
	limit, err := intArg(host, args, 2, "limit")
	if err != nil {
		return nil, err
	}
	if int64(int(limit)) != limit {
		return nil, fmt.Errorf("limit out of range")
	}
	return listFromStrings(host, re.Split(text, int(limit))), nil
}

func regexpArgs(args []Value) (*regexp.Regexp, string, error) {
	pattern, err := stringArg(args, 0, "pattern")
	if err != nil {
		return nil, "", err
	}
	text, err := stringArg(args, 1, "text")
	if err != nil {
		return nil, "", err
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, "", err
	}
	return re, text, nil
}
