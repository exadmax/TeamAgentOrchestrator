package fsx

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A reader holding the target open must not make the write fail (it did on Windows).
func TestReplaceWhileTargetIsOpen(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "state.json")
	tmp := filepath.Join(dir, "state.json.tmp")
	if err := os.WriteFile(dst, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmp, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	reader, err := os.Open(dst)
	if err != nil {
		t.Fatal(err)
	}
	time.AfterFunc(100*time.Millisecond, func() { reader.Close() })
	if err := Replace(tmp, dst); err != nil {
		t.Fatalf("Replace: %v", err)
	}
	if got, _ := os.ReadFile(dst); string(got) != "new" {
		t.Fatalf("dst = %q", got)
	}
}
