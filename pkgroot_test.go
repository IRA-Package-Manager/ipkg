package ipkg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMakeRoot(t *testing.T) {
	path := "./test"
	root, err := CreateRoot(path)
	if err != nil {
		t.Fatalf("creating package root: %v", err)
	}
	if root == nil {
		t.Fatal("package root wasn't returned")
	}
	if root.path != path {
		t.Errorf("root has wrong path: got %q, expepected %q", root.path, path)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("package root wasn't created")
	}
	if _, err := os.Stat(filepath.Join(path, "db.sqlite3")); os.IsNotExist(err) {
		t.Fatal("database wasn't created")
	}
}
