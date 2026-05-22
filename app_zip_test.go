package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestZipDir(t *testing.T) {
	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("hello"), 0644)
	sub := filepath.Join(srcDir, "sub")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(sub, "b.txt"), []byte("world"), 0644)

	zipPath, err := zipDir(srcDir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(zipPath)

	if filepath.Ext(zipPath) != ".zip" {
		t.Fatalf("expected .zip extension, got %s", zipPath)
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	names := map[string]bool{}
	for _, f := range r.File {
		names[f.Name] = true
	}
	if !names["a.txt"] {
		t.Error("missing a.txt in zip")
	}
	if !names["sub/b.txt"] {
		t.Error("missing sub/b.txt in zip")
	}
}

func TestZipDirOverwrites(t *testing.T) {
	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("v1"), 0644)

	zipPath1, _ := zipDir(srcDir)

	os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("v2"), 0644)
	zipPath2, _ := zipDir(srcDir)

	if zipPath1 != zipPath2 {
		t.Fatalf("zip paths should be identical: %s vs %s", zipPath1, zipPath2)
	}
}
