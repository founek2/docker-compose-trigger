package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetServicePathRejectsTraversal(t *testing.T) {
	originalPath, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "service"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalPath); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	for _, name := range []string{"..", "../outside", "escape", "nested/service", `nested\service`} {
		t.Run(name, func(t *testing.T) {
			if path, err := getServicePath(name); err == nil {
				t.Errorf("getServicePath(%q) = %q, want an error", name, path)
			}
		})
	}

	got, err := getServicePath("service")
	if err != nil {
		t.Fatalf("getServicePath(service): %v", err)
	}
	want, err := filepath.EvalSymlinks(filepath.Join(root, "service"))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("getServicePath(service) = %q, want %q", got, want)
	}
}
