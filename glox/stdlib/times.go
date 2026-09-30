package stdlib

import (
	"fmt"
	gomath "math"
	"time"
)

func Times() Module {
	return Module{
		"ANSIC":       time.ANSIC,
		"DateOnly":    "2006-01-02",
		"DateTime":    "2006-01-02 15:04:05",
		"Kitchen":     time.Kitchen,
		"RFC3339":     time.RFC3339,
		"RFC3339Nano": time.RFC3339Nano,
		"TimeOnly":    "15:04:05",
		"add":         Function("times.add", 2, timesAdd),
		"date":        Function("times.date", 6, timesDate),
		"format":      Function("times.format", 2, timesFormat),
		"formatNow":   Function("times.formatNow", 1, timesFormatNow),
		"now":         Function("times.now", 0, timesNow),
		"parse":       Function("times.parse", 2, timesParse),
		"since":       Function("times.since", 1, timesSince),
		"sleep":       Function("times.sleep", 1, timesSleep),
		"unix":        Function("times.unix", 0, timesUnix),
		"unixMilli":   Function("times.unixMilli", 0, timesUnixMilli),
		"unixNano":    Function("times.unixNano", 0, timesUnixNano),
	}
}

func timesAdd(host Host, args []Value) (Value, error) {
	base, err := timeArg(host, args, 0, "time")
	if err != nil {
		return nil, err
	}
	seconds, err := floatArg(host, args, 1, "seconds")
	if err != nil {
		return nil, err
	}
	return timeValue(base.Add(durationSeconds(seconds))), nil
}

func timesDate(host Host, args []Value) (Value, error) {
	year, err := intArg(host, args, 0, "year")
	if err != nil {
		return nil, err
	}
	month, err := intArg(host, args, 1, "month")
	if err != nil {
		return nil, err
	}
	day, err := intArg(host, args, 2, "day")
	if err != nil {
		return nil, err
	}
	hour, err := intArg(host, args, 3, "hour")
	if err != nil {
		return nil, err
	}
	minute, err := intArg(host, args, 4, "minute")
	if err != nil {
		return nil, err
	}
	second, err := intArg(host, args, 5, "second")
	if err != nil {
		return nil, err
	}
	return timeValue(time.Date(int(year), time.Month(month), int(day), int(hour), int(minute), int(second), 0, time.UTC)), nil
}

func timesFormat(host Host, args []Value) (Value, error) {
	value, err := timeArg(host, args, 0, "time")
	if err != nil {
		return nil, err
	}
	layout, err := stringArg(args, 1, "layout")
	if err != nil {
		return nil, err
	}
	return value.UTC().Format(layout), nil
}

func timesFormatNow(host Host, args []Value) (Value, error) {
	layout, err := stringArg(args, 0, "layout")
	if err != nil {
		return nil, err
	}
	return time.Now().UTC().Format(layout), nil
}

func timesNow(host Host, args []Value) (Value, error) {
	return float64(time.Now().UnixNano()) / 1e9, nil
}

func timesParse(host Host, args []Value) (Value, error) {
	layout, text, err := twoStrings(args, "layout", "value")
	if err != nil {
		return nil, err
	}
	value, err := time.Parse(layout, text)
	if err != nil {
		return nil, err
	}
	return timeValue(value), nil
}

func timesSince(host Host, args []Value) (Value, error) {
	value, err := timeArg(host, args, 0, "time")
	if err != nil {
		return nil, err
	}
	return time.Since(value).Seconds(), nil
}

func timesSleep(host Host, args []Value) (Value, error) {
	seconds, err := floatArg(host, args, 0, "seconds")
	if err != nil {
		return nil, err
	}
	if seconds < 0 {
		return nil, fmt.Errorf("seconds must be non-negative")
	}
	time.Sleep(durationSeconds(seconds))
	return true, nil
}

func timesUnix(host Host, args []Value) (Value, error) {
	return time.Now().Unix(), nil
}

func timesUnixMilli(host Host, args []Value) (Value, error) {
	return time.Now().UnixMilli(), nil
}

func timesUnixNano(host Host, args []Value) (Value, error) {
	return time.Now().UnixNano(), nil
}

func timeArg(host Host, args []Value, index int, name string) (time.Time, error) {
	seconds, ok := host.AsFloat64(args[index])
	if !ok {
		return time.Time{}, fmt.Errorf("%s must be a number", name)
	}
	whole, fraction := gomath.Modf(seconds)
	return time.Unix(int64(whole), int64(fraction*1e9)).UTC(), nil
}

func timeValue(value time.Time) Value {
	if value.Nanosecond() == 0 {
		return value.Unix()
	}
	return float64(value.UnixNano()) / 1e9
}

func durationSeconds(seconds float64) time.Duration {
	return time.Duration(seconds * float64(time.Second))
}
