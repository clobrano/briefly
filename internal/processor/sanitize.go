package processor

import (
	"regexp"
	"strings"
	"unicode"
)

// Characters that are problematic for cross-platform file sync (Windows, Linux, macOS, Syncthing)
// Windows reserved: < > : " / \ | ? *
// Unix reserved: / (null byte)
// Syncthing and other sync tools may have issues with these across different filesystems
var invalidCharsRegexp = regexp.MustCompile(`[<>:"/\\|?*]`)

// Control characters (0x00-0x1F) should be removed
var controlCharsRegexp = regexp.MustCompile(`[\x00-\x1f]`)

// Multiple consecutive replacement chars should be collapsed
var multipleUnderscoresRegexp = regexp.MustCompile(`_+`)

// SanitizeFilename cleans a filename to ensure cross-platform compatibility
// with file synchronization tools like Syncthing.
//
// It handles:
// - Reserved characters on Windows: < > : " / \ | ? *
// - Path separators: / and \
// - Control characters (0x00-0x1F)
// - Leading/trailing dots and spaces (Windows ignores them)
// - Empty filenames (returns fallback)
// - Very long filenames (truncates to 200 chars)
func SanitizeFilename(filename string) string {
	if filename == "" {
		return "unnamed"
	}

	// Remove control characters
	result := controlCharsRegexp.ReplaceAllString(filename, "")

	// Replace invalid characters with underscore
	result = invalidCharsRegexp.ReplaceAllString(result, "_")

	// Replace any remaining non-printable or problematic Unicode characters
	var builder strings.Builder
	for _, r := range result {
		if unicode.IsPrint(r) && r != unicode.ReplacementChar {
			builder.WriteRune(r)
		} else {
			builder.WriteRune('_')
		}
	}
	result = builder.String()

	// Collapse multiple consecutive underscores
	result = multipleUnderscoresRegexp.ReplaceAllString(result, "_")

	// Trim leading/trailing spaces, dots, and underscores
	result = strings.Trim(result, " ._")

	// Handle empty result after sanitization
	if result == "" {
		return "unnamed"
	}

	// Truncate very long filenames (keep reasonable length for all filesystems)
	// 200 chars leaves room for path and extension
	const maxLength = 200
	if len(result) > maxLength {
		result = result[:maxLength]
		// Clean up any trailing underscore from truncation
		result = strings.TrimRight(result, "_")
	}

	return result
}
