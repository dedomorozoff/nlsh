package executor

import (
	"testing"
)

func TestTranslateToPowerShell(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// rm variants
		{"rm -rf /tmp", "Remove-Item -Recurse -Force /tmp"},
		{"rm -r /tmp", "Remove-Item -Recurse -Force /tmp"},
		{"rm -f file.txt", "Remove-Item -Force file.txt"},
		{"rm file.txt", "Remove-Item file.txt"},
		
		// file operations
		{"mkdir foo", "New-Item -ItemType Directory foo"},
		{"touch file.txt", "New-Item -ItemType File file.txt"},
		{"cat file.txt", "Get-Content file.txt"},
		{"ls", "Get-ChildItem"},
		{"ls -la", "Get-ChildItem -la"},
		{"cp src dst", "Copy-Item src dst"},
		{"mv src dst", "Move-Item src dst"},
		
		// system
		{"pwd", "Get-Location"},
		{"echo hello", "Write-Output hello"},
		
		// network
		{"ping -c 4 ya.ru", "Test-Connection -Count 4 ya.ru"},
		{"curl http://example.com", "Invoke-WebRequest http://example.com"},
		{"wget http://example.com", "Invoke-WebRequest http://example.com"},
		
		// no translation needed
		{"Get-ChildItem", "Get-ChildItem"},
		{"Remove-Item foo", "Remove-Item foo"},
	}

	for _, tc := range tests {
		result := translateToPowerShell(tc.input)
		if result != tc.expected {
			t.Errorf("translate(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}