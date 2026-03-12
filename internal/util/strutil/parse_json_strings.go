package strutil

import "encoding/json"

// ParseJSONStringArray parses a JSON array of strings.
// If parsing fails or the result is empty, treats the input as a single
// element and returns []string{s}. This allows backward-compatible reading
// of fields that may store either a plain path or a JSON array of paths.
func ParseJSONStringArray(s string) []string {
	var result []string
	if err := json.Unmarshal([]byte(s), &result); err == nil && len(result) > 0 {
		return result
	}
	return []string{s}
}
