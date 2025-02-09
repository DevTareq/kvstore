package utils

import "strings"

// GetKeyFromPath extracts the key from a URL path like "/kv/{key}"
func GetKeyFromPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return ""
	}
	return parts[2]
}
