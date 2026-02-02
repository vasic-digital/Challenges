package assertion

import "strings"

// ParseAssertionString parses a compact assertion string of the
// form "type:value" into its components. If no colon is present
// the entire string is treated as the type and value is nil.
//
// Examples:
//
//	"contains:func"  -> ("contains", "func")
//	"not_empty"      -> ("not_empty", nil)
//	"min_length:100" -> ("min_length", "100")
func ParseAssertionString(
	s string,
) (assertionType string, value any) {
	parts := strings.SplitN(s, ":", 2)
	assertionType = parts[0]

	if len(parts) > 1 {
		value = parts[1]
	}

	return
}
