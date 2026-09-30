package stdlib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
)

func JSON() Module {
	return Module{
		"compact": Function("json.compact", 1, jsonCompact),
		"decode":  Function("json.decode", 1, jsonDecode),
		"encode":  Function("json.encode", 1, jsonEncode),
		"indent":  Function("json.indent", 3, jsonIndent),
		"valid":   Function("json.valid", 1, jsonValid),
	}
}

func jsonCompact(host Host, args []Value) (Value, error) {
	text, err := stringArg(args, 0, "json")
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := json.Compact(&out, []byte(text)); err != nil {
		return nil, err
	}
	return out.String(), nil
}

func jsonDecode(host Host, args []Value) (Value, error) {
	text, err := stringArg(args, 0, "json")
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	var raw Value
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}
	var trailing Value
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("invalid trailing JSON data")
		}
		return nil, err
	}
	return fromJSON(host, raw)
}

func jsonEncode(host Host, args []Value) (Value, error) {
	raw, err := toJSON(host, args[0])
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

func jsonIndent(host Host, args []Value) (Value, error) {
	text, err := stringArg(args, 0, "json")
	if err != nil {
		return nil, err
	}
	prefix, err := stringArg(args, 1, "prefix")
	if err != nil {
		return nil, err
	}
	indent, err := stringArg(args, 2, "indent")
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := json.Indent(&out, []byte(text), prefix, indent); err != nil {
		return nil, err
	}
	return out.String(), nil
}

func jsonValid(host Host, args []Value) (Value, error) {
	text, err := stringArg(args, 0, "json")
	if err != nil {
		return nil, err
	}
	return json.Valid([]byte(text)), nil
}

func toJSON(host Host, value Value) (Value, error) {
	switch v := value.(type) {
	case nil, bool, string, int64:
		return v, nil
	case float64:
		if math.IsInf(v, 0) || math.IsNaN(v) {
			return nil, fmt.Errorf("cannot encode non-finite number")
		}
		return v, nil
	}
	if items, ok := host.ListItems(value); ok {
		out := make([]Value, len(items))
		for i, item := range items {
			converted, err := toJSON(host, item)
			if err != nil {
				return nil, err
			}
			out[i] = converted
		}
		return out, nil
	}
	if items, ok := host.MapItems(value); ok {
		return mapToJSON(host, items)
	}
	if items, ok := host.ModuleExports(value); ok {
		return mapToJSON(host, items)
	}
	return nil, fmt.Errorf("cannot encode %s as JSON", host.TypeName(value))
}

func mapToJSON(host Host, items map[string]Value) (Value, error) {
	out := make(map[string]Value, len(items))
	for key, item := range items {
		converted, err := toJSON(host, item)
		if err != nil {
			return nil, err
		}
		out[key] = converted
	}
	return out, nil
}

func fromJSON(host Host, value Value) (Value, error) {
	switch v := value.(type) {
	case nil, bool, string, float64:
		return v, nil
	case json.Number:
		return jsonNumber(v)
	case []Value:
		items := make([]Value, len(v))
		for i, item := range v {
			converted, err := fromJSON(host, item)
			if err != nil {
				return nil, err
			}
			items[i] = converted
		}
		return host.NewList(items), nil
	case map[string]Value:
		items := make(map[string]Value, len(v))
		for key, item := range v {
			converted, err := fromJSON(host, item)
			if err != nil {
				return nil, err
			}
			items[key] = converted
		}
		return host.NewMap(items), nil
	default:
		return nil, fmt.Errorf("unsupported JSON value %T", value)
	}
}

func jsonNumber(number json.Number) (Value, error) {
	text := number.String()
	if !strings.ContainsAny(text, ".eE") {
		if value, err := number.Int64(); err == nil {
			return value, nil
		}
	}
	value, err := number.Float64()
	if err != nil {
		return nil, err
	}
	return value, nil
}
