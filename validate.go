package endee

import "regexp"

// NameRegex is the compiled regex for validating index names.
var NameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// validSpaceTypes contains the set of supported distance metric types.
var validSpaceTypes = map[string]bool{
	"cosine": true,
	"l2":     true,
	"ip":     true,
}

// isValidIndexName validates that the index name is alphanumeric with underscores
// and fewer than MaxIndexNameLenAllowed characters.
func isValidIndexName(name string) bool {
	if len(name) == 0 || len(name) >= MaxIndexNameLenAllowed {
		return false
	}

	return NameRegex.MatchString(name)
}
