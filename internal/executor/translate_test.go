package executor

import "testing"

func TestTranslateToWindows(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Should translate
		{"rm -rf /tmp", "rmdir /s /q /tmp"},
		{"mkdir foo", "mkdir foo"},
		{"ls", "dir"},
		{"cat file.txt", "type file.txt"},
		{"cp src dst", "copy src dst"},
		{"clear", "cls"},

		// Should NOT translate (Windows/PS commands)
		{"git status", "git status"},
		{"sqlite3 db.sqlite", "sqlite3 db.sqlite"},
		{"python script.py", "python script.py"},
		{"Get-ChildItem", "Get-ChildItem"},
		{"Invoke-WebRequest https://example.com", "Invoke-WebRequest https://example.com"},
	}

	for _, tc := range tests {
		result := translateToWindows(tc.input)
		if result != tc.expected {
			t.Errorf("translate(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}