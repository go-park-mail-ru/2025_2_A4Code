package validation

import "strings"

func HasDangerousCharacters(input string) bool {
	dangerous := []string{
		"<", ">", "script", "javascript:", "onload", "onerror",
		"--", "/*", "*/", "'", "\"", "&", ";", "|", "\n", "\r",
		"../", "file://", "http://", "https://",
	}

	lowerInput := strings.ToLower(input)
	for _, char := range dangerous {
		if strings.Contains(lowerInput, char) {
			return true
		}
	}
	return false
}
