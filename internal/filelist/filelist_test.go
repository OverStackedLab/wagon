package filelist

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFormatSizeShowsKnownDirectorySize(t *testing.T) {
	item := Item{
		Name:      "Photos",
		IsDir:     true,
		Size:      5 * 1024 * 1024,
		SizeKnown: true,
	}

	if got := FormatSize(item); got != "5.0 MB" {
		t.Fatalf("FormatSize() = %q, want %q", got, "5.0 MB")
	}
}

func TestFormatSizeShowsUnknownDirectorySize(t *testing.T) {
	item := Item{Name: "Photos", IsDir: true}

	if got := FormatSize(item); got != "?" {
		t.Fatalf("FormatSize() = %q, want %q", got, "?")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{name: "bytes", bytes: 42, want: "42 B"},
		{name: "megabytes", bytes: 2 * 1024 * 1024, want: "2.0 MB"},
		{name: "unknown", bytes: -1, want: "?"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := FormatBytes(test.bytes); got != test.want {
				t.Fatalf("FormatBytes() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSizeLocalSumsRegularFiles(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "one.txt"), 3)
	mustWriteFile(t, filepath.Join(root, "nested", "two.txt"), 5)

	got, err := SizeLocal(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if got != 8 {
		t.Fatalf("SizeLocal() = %d, want 8", got)
	}
}

func mustWriteFile(t *testing.T, path string, size int) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}
