package transform

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func init() {
	// String transforms
	Register("uppercase", func(v string) (string, error) {
		return strings.ToUpper(v), nil
	})
	Register("lowercase", func(v string) (string, error) {
		return strings.ToLower(v), nil
	})
	Register("trim", func(v string) (string, error) {
		return strings.TrimSpace(v), nil
	})
	Register("title", func(v string) (string, error) {
		return strings.Title(v), nil
	})

	// Type conversion transforms
	Register("to_int", func(v string) (string, error) {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return "", fmt.Errorf("cannot convert %q to int: %w", v, err)
		}
		return strconv.Itoa(int(f)), nil
	})
	Register("to_float", func(v string) (string, error) {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return "", fmt.Errorf("cannot convert %q to float: %w", v, err)
		}
		return strconv.FormatFloat(f, 'f', -1, 64), nil
	})
	Register("to_bool", func(v string) (string, error) {
		lower := strings.ToLower(strings.TrimSpace(v))
		switch lower {
		case "1", "true", "yes", "on":
			return "true", nil
		case "0", "false", "no", "off", "":
			return "false", nil
		default:
			return "", fmt.Errorf("cannot convert %q to bool", v)
		}
	})

	// Math transforms
	RegisterParameterized("multiply", func(v string, param string) (string, error) {
		val, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return "", err
		}
		factor, err := strconv.ParseFloat(param, 64)
		if err != nil {
			return "", fmt.Errorf("invalid multiply factor: %w", err)
		}
		return strconv.FormatFloat(val*factor, 'f', -1, 64), nil
	})
	RegisterParameterized("divide", func(v string, param string) (string, error) {
		val, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return "", err
		}
		divisor, err := strconv.ParseFloat(param, 64)
		if err != nil {
			return "", fmt.Errorf("invalid divide divisor: %w", err)
		}
		if divisor == 0 {
			return "", fmt.Errorf("division by zero")
		}
		return strconv.FormatFloat(val/divisor, 'f', -1, 64), nil
	})
	RegisterParameterized("add", func(v string, param string) (string, error) {
		val, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return "", err
		}
		addend, err := strconv.ParseFloat(param, 64)
		if err != nil {
			return "", fmt.Errorf("invalid add value: %w", err)
		}
		return strconv.FormatFloat(val+addend, 'f', -1, 64), nil
	})
	RegisterParameterized("round", func(v string, param string) (string, error) {
		val, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return "", err
		}
		places, err := strconv.Atoi(param)
		if err != nil {
			return "", fmt.Errorf("invalid round places: %w", err)
		}
		shift := math.Pow(10, float64(places))
		rounded := math.Round(val*shift) / shift
		return strconv.FormatFloat(rounded, 'f', places, 64), nil
	})

	// String manipulation transforms
	RegisterParameterized("replace", func(v string, param string) (string, error) {
		parts := strings.SplitN(param, "->", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("replace requires 'old->new' format, got %q", param)
		}
		return strings.ReplaceAll(v, parts[0], parts[1]), nil
	})
	RegisterParameterized("prefix", func(v string, param string) (string, error) {
		return param + v, nil
	})
	RegisterParameterized("suffix", func(v string, param string) (string, error) {
		return v + param, nil
	})
	RegisterParameterized("truncate", func(v string, param string) (string, error) {
		maxLen, err := strconv.Atoi(param)
		if err != nil {
			return "", fmt.Errorf("invalid truncate length: %w", err)
		}
		if len(v) > maxLen {
			return v[:maxLen], nil
		}
		return v, nil
	})
	RegisterParameterized("pad_left", func(v string, param string) (string, error) {
		width, err := strconv.Atoi(param)
		if err != nil {
			return "", fmt.Errorf("invalid pad width: %w", err)
		}
		for len(v) < width {
			v = " " + v
		}
		return v, nil
	})
	RegisterParameterized("pad_right", func(v string, param string) (string, error) {
		width, err := strconv.Atoi(param)
		if err != nil {
			return "", fmt.Errorf("invalid pad width: %w", err)
		}
		for len(v) < width {
			v = v + " "
		}
		return v, nil
	})

	// Default value
	RegisterParameterized("default", func(v string, param string) (string, error) {
		if v == "" {
			return param, nil
		}
		return v, nil
	})

	// Date transforms — target layout uses Go reference time format
	RegisterParameterized("date_format", func(v string, param string) (string, error) {
		// Try common input formats
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
			"2006-01-02",
			"01/02/2006",
			"01-02-2006",
			"Jan 2, 2006",
			time.RFC1123,
		}
		for _, format := range formats {
			if t, err := time.Parse(format, v); err == nil {
				return t.Format(param), nil
			}
		}
		return "", fmt.Errorf("could not parse date %q", v)
	})
}
