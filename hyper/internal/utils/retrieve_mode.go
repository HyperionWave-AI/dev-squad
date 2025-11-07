package utils

// ParseRetrieveMode parses the retrieve mode parameter and returns the retrieve type, chunk size in lines, and t-shirt size
// Maps: chunk-s→50/s, chunk-m→100/m, chunk-l→200/l, chunk-xl→400/xl, chunk→200/l (backward compat), full→0/empty
func ParseRetrieveMode(mode string) (retrieveType string, chunkLines int, tshirtSize string) {
	switch mode {
	case "chunk-s":
		return "chunk", 50, "s"
	case "chunk-m":
		return "chunk", 100, "m"
	case "chunk-l":
		return "chunk", 200, "l"
	case "chunk-xl":
		return "chunk", 400, "xl"
	case "chunk":
		// Backward compatibility: chunk defaults to chunk-l (200 lines)
		return "chunk", 200, "l"
	case "full":
		return "full", 0, ""
	default:
		// Default to chunk-l for unknown values
		return "chunk", 200, "l"
	}
}

// IsValidRetrieveMode checks if the given retrieve mode is valid
func IsValidRetrieveMode(mode string) bool {
	switch mode {
	case "chunk-s", "chunk-m", "chunk-l", "chunk-xl", "chunk", "full":
		return true
	default:
		return false
	}
}
