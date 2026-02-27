package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 Bytes"},
		{512, "512.00 Bytes"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
	}

	for _, tt := range tests {
		got := FormatBytes(tt.input)
		if got != tt.expected {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCalculateDirectorySize(t *testing.T) {
	dir := t.TempDir()

	// Write two files with known sizes
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), make([]byte, 100), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), make([]byte, 200), 0644); err != nil {
		t.Fatal(err)
	}

	size, err := CalculateDirectorySize(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 300 {
		t.Errorf("CalculateDirectorySize = %d, want 300", size)
	}
}

func TestCalculateDirectorySize_Nested(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "root.txt"), make([]byte, 50), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "nested.txt"), make([]byte, 150), 0644); err != nil {
		t.Fatal(err)
	}

	size, err := CalculateDirectorySize(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 200 {
		t.Errorf("CalculateDirectorySize (nested) = %d, want 200", size)
	}
}
