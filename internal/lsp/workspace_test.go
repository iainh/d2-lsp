package lsp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceD2FilesReadsD2Files(t *testing.T) {
	root := t.TempDir()
	diagramPath := filepath.Join(root, "diagram.d2")
	ignoredPath := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(diagramPath, []byte("x -> y\n"), 0644); err != nil {
		t.Fatalf("write diagram: %v", err)
	}
	if err := os.WriteFile(ignoredPath, []byte("not d2"), 0644); err != nil {
		t.Fatalf("write ignored: %v", err)
	}

	files, uriByPath := workspaceD2Files(root)
	if files[diagramPath] != "x -> y\n" {
		t.Fatalf("unexpected files %#v", files)
	}
	if _, ok := files[ignoredPath]; ok {
		t.Fatalf("expected non-d2 file to be ignored, got %#v", files)
	}
	if uriByPath[diagramPath] != uriFromPath(diagramPath) {
		t.Fatalf("unexpected uri map %#v", uriByPath)
	}
}

func TestWorkspaceD2FilesSkipsIgnoredDirectories(t *testing.T) {
	root := t.TempDir()
	ignoredDir := filepath.Join(root, ".git")
	if err := os.Mkdir(ignoredDir, 0755); err != nil {
		t.Fatalf("mkdir ignored: %v", err)
	}
	ignoredPath := filepath.Join(ignoredDir, "ignored.d2")
	if err := os.WriteFile(ignoredPath, []byte("x -> y\n"), 0644); err != nil {
		t.Fatalf("write ignored: %v", err)
	}

	files, _ := workspaceD2Files(root)
	if len(files) != 0 {
		t.Fatalf("expected ignored directory to be skipped, got %#v", files)
	}
}

func TestWorkspacesD2FilesReadsMultipleRoots(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()
	pathA := filepath.Join(rootA, "a.d2")
	pathB := filepath.Join(rootB, "b.d2")
	if err := os.WriteFile(pathA, []byte("a\n"), 0644); err != nil {
		t.Fatalf("write a: %v", err)
	}
	if err := os.WriteFile(pathB, []byte("b\n"), 0644); err != nil {
		t.Fatalf("write b: %v", err)
	}

	files, uris := workspacesD2Files([]string{rootA, rootB})
	if files[pathA] != "a\n" || files[pathB] != "b\n" {
		t.Fatalf("unexpected files %#v", files)
	}
	if uris[pathA] != uriFromPath(pathA) || uris[pathB] != uriFromPath(pathB) {
		t.Fatalf("unexpected uris %#v", uris)
	}
}
