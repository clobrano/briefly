package processor

import (
	"strings"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic cases
		{
			name:     "normal filename unchanged",
			input:    "my-video-title",
			expected: "my-video-title",
		},
		{
			name:     "filename with spaces preserved",
			input:    "my video title",
			expected: "my video title",
		},

		// Pipe character (Syncthing issue)
		{
			name:     "pipe character replaced",
			input:    "video|title",
			expected: "video_title",
		},
		{
			name:     "multiple pipes replaced",
			input:    "a|b|c|d",
			expected: "a_b_c_d",
		},

		// Path separators
		{
			name:     "forward slash replaced",
			input:    "path/to/file",
			expected: "path_to_file",
		},
		{
			name:     "backslash replaced",
			input:    "path\\to\\file",
			expected: "path_to_file",
		},

		// Windows reserved characters
		{
			name:     "less than replaced",
			input:    "file<name",
			expected: "file_name",
		},
		{
			name:     "greater than replaced",
			input:    "file>name",
			expected: "file_name",
		},
		{
			name:     "colon replaced",
			input:    "file:name",
			expected: "file_name",
		},
		{
			name:     "double quote replaced",
			input:    `file"name`,
			expected: "file_name",
		},
		{
			name:     "question mark replaced",
			input:    "file?name",
			expected: "file_name",
		},
		{
			name:     "asterisk replaced",
			input:    "file*name",
			expected: "file_name",
		},

		// Multiple consecutive invalid chars collapsed
		{
			name:     "multiple invalid chars collapsed",
			input:    "file|||name",
			expected: "file_name",
		},
		{
			name:     "mixed invalid chars collapsed",
			input:    "file|/\\name",
			expected: "file_name",
		},

		// Leading/trailing cleanup
		{
			name:     "leading dots removed",
			input:    "...filename",
			expected: "filename",
		},
		{
			name:     "trailing dots removed",
			input:    "filename...",
			expected: "filename",
		},
		{
			name:     "leading spaces removed",
			input:    "   filename",
			expected: "filename",
		},
		{
			name:     "trailing spaces removed",
			input:    "filename   ",
			expected: "filename",
		},
		{
			name:     "leading underscores removed",
			input:    "___filename",
			expected: "filename",
		},

		// Empty and fallback cases
		{
			name:     "empty string returns unnamed",
			input:    "",
			expected: "unnamed",
		},
		{
			name:     "only invalid chars returns unnamed",
			input:    "|||",
			expected: "unnamed",
		},
		{
			name:     "only dots returns unnamed",
			input:    "...",
			expected: "unnamed",
		},
		{
			name:     "only spaces returns unnamed",
			input:    "   ",
			expected: "unnamed",
		},

		// Real-world examples
		{
			name:     "youtube title with pipe",
			input:    "How to Code | Programming Tutorial",
			expected: "How to Code _ Programming Tutorial",
		},
		{
			name:     "filename with quotes",
			input:    `"Best Video Ever"`,
			expected: "Best Video Ever",
		},
		{
			name:     "complex mixed case",
			input:    `My Video: "The Best" | Part 1/2`,
			expected: "My Video_ _The Best_ _ Part 1_2",
		},

		// Unicode handling
		{
			name:     "unicode characters preserved",
			input:    "日本語ファイル名",
			expected: "日本語ファイル名",
		},
		{
			name:     "emoji preserved",
			input:    "video🎬title",
			expected: "video🎬title",
		},
		{
			name:     "accented characters preserved",
			input:    "café-résumé",
			expected: "café-résumé",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFilename_LongFilename(t *testing.T) {
	// Test truncation of very long filenames
	longName := strings.Repeat("a", 300)
	result := SanitizeFilename(longName)

	if len(result) > 200 {
		t.Errorf("SanitizeFilename should truncate to 200 chars, got %d", len(result))
	}

	// Ensure the result is still valid
	if result == "" || result == "unnamed" {
		t.Errorf("SanitizeFilename should return truncated name, not empty/unnamed")
	}
}

func TestSanitizeFilename_ControlCharacters(t *testing.T) {
	// Test removal of control characters
	input := "file\x00name\x1ftest"
	result := SanitizeFilename(input)

	// Should not contain any control characters
	for _, r := range result {
		if r < 32 {
			t.Errorf("SanitizeFilename should remove control characters, found %q", r)
		}
	}
}
