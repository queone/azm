package utl

import (
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/exp/rand"
)

// Case insensitive substring check
func SubString(large, small string) bool {
	return strings.Contains(strings.ToLower(large), strings.ToLower(small))
}

// Split the string and return last element

// Returns the element after the last dot in the string. Returns empty if there's no dot.
func LastElemByDot(s string) string {
	if pos := strings.LastIndex(s, "."); pos != -1 && pos < len(s)-1 {
		return s[pos+1:]
	}
	return ""
}

// Safely casts obj to a slice ([]interface{}); returns nil if not possible.
func Slice(obj any) []any {
	if objSlice, ok := obj.([]any); ok {
		return objSlice
	}
	return nil
}

// Safely casts obj to a map (map[string]interface{}); returns nil if not possible.
func Map(obj any) map[string]any {
	if objMap, ok := obj.(map[string]any); ok {
		return objMap
	}
	return nil
}

// Safely casts an object to a string only if it is already a string; else returns "".
func Str(obj any) string {
	// Useful when you only want to accept values that are already strings and
	// reject others (e.g., parsing JSON where a field must be a string).
	if objString, ok := obj.(string); ok {
		return objString
	}
	return ""
}

// Safely casts obj to a boolean (bool); returns false if not possible.
func Bool(obj any) bool {
	switch v := obj.(type) {
	case bool:
		return v
	case string:
		// Returns true if the string is "1", "t", "T", "true", "TRUE",
		// or "True", and false otherwise
		b, err := strconv.ParseBool(v)
		return err == nil && b
	case int:
		return v != 0
	default:
		return false
	}
}

// Safely casts obj to an int64 (int64); returns 0 if not possible.
func Int64(obj any) int64 {
	switch v := obj.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			return i
		}
		return 0
	default:
		return 0
	}
}

// Returns string value of unknown variable, wrapped in single quotes. This is a special
// function to lookout for leading '*', which YAML does not allow and must be single-quoted
func StrSingleQuote(x any) string {
	s := Str(x)
	if strings.HasPrefix(s, "*") {
		return "'" + s + "'"
	}
	return s
}

// Converts any value to its string representation using default formatting.
func ToStr(value any) string {
	// Examples:
	//   ToStr(42)          // "42"
	//   ToStr(3.14)        // "3.14"
	//   ToStr(true)        // "true"
	//   ToStr([]int{1, 2}) // "[1 2]"
	//   ToStr(nil)         // "<nil>"
	return fmt.Sprintf("%v", value)
}

// ItemInList checks if a given string (arg) is present in a list of strings (argList).
// Returns true if found, false otherwise. Consider using utl.NewStringSet() type instead.

// Return string of spaces for padded printing. Needed when printing terminal colors.
// Colorize output uses % sequences that conflict with Printf's own formatting with %
func PadSpaces(targetWidth, stringWidth int) string {
	padding := targetWidth - stringWidth
	if padding > 0 {
		return fmt.Sprintf("%*s", padding, " ")
	} else {
		return ""
	}
}

// Return value as a string, padded with leading spaces, totalling width size wide. This is
// needed when printing terminal colors, because they conflict with Printf's own '%' formatting
func PreSpc(value any, width int) string {
	str := ToStr(value)
	padding := width - len(str)
	if padding > 0 {
		return fmt.Sprintf("%*s%s", padding, " ", str)
	} else {
		return str
	}
}

// Return value as a string, padded with trailing spaces, totalling width size wide. This is
// needed when printing terminal colors, because they conflict with Printf's own '%' formatting
func PostSpc(value any, width int) string {
	str := ToStr(value)
	padding := width - len(str)
	if padding > 0 {
		return fmt.Sprintf("%s%*s", str, padding, " ")
	} else {
		return str
	}
}

// Old version of commafy
func Int2StrWithCommas(num any) string {
	return Commafy(num)
}

// Converts given generic type value to a string with commas as thousand separators
func Commafy(num any) string {
	var numStr string

	switch v := num.(type) {
	case int:
		numStr = strconv.Itoa(v)
	case int64:
		numStr = strconv.FormatInt(v, 10)
	case uint64:
		numStr = strconv.FormatUint(v, 10)
	case float64:
		numStr = strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		numStr = strconv.FormatFloat(float64(v), 'f', -1, 32)
	case *big.Int:
		numStr = v.String()
	default:
		panic("Unsupported number type for Commafy")
	}

	// Split into integer and fractional parts if there's a decimal point
	parts := strings.Split(numStr, ".")
	intPart := parts[0]

	n := len(intPart)
	if n <= 3 {
		// Rejoin with fractional part if it exists
		if len(parts) > 1 {
			return intPart + "." + parts[1]
		}
		return intPart
	}

	var out strings.Builder
	for i, c := range intPart {
		if i > 0 && (n-i)%3 == 0 {
			out.WriteRune(',')
		}
		out.WriteRune(c)
	}

	// Append decimal part if present
	if len(parts) > 1 {
		out.WriteRune('.')
		out.WriteString(parts[1])
	}

	return out.String()
}

// Generates a random string of the given length
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	rand.Seed(uint64(time.Now().UnixNano())) // Convert int64 to uint64
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// Returns true if given string is a valid UUID number. False otherwise.
func ValidUuid(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// Return the map string object's keys sorted
func SortMapStringKeys(obj map[string]string) (sortedKeys []string) {
	sortedKeys = make([]string, 0, len(obj))
	for k := range obj {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)
	return sortedKeys
}
