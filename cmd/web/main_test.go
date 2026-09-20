package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveMediaAndList(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "source.mp4"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	path, err := resolveMedia(dir, "source.mp4")
	if err != nil || path != filepath.Join(dir, "source.mp4") {
		t.Fatalf("resolveMedia() = %q, %v", path, err)
	}
	for _, name := range []string{"", "../source.mp4", "ignore.txt", "/tmp/source.mp4"} {
		if _, err := resolveMedia(dir, name); err == nil {
			t.Fatalf("resolveMedia(%q) returned no error", name)
		}
	}
	files, err := mediaFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(files, []string{"source.mp4"}) {
		t.Fatalf("mediaFiles() = %#v", files)
	}
}

func TestProgress(t *testing.T) {
	s := &server{}
	s.rendering.Store(true)
	s.progressDone.Store(2)
	s.progressTotal.Store(5)
	recorder := httptest.NewRecorder()
	s.progress(recorder, httptest.NewRequest("GET", "/progress", nil))

	var got struct {
		Running bool  `json:"running"`
		Done    int64 `json:"done"`
		Total   int64 `json:"total"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if !got.Running || got.Done != 2 || got.Total != 5 {
		t.Fatalf("progress = %+v", got)
	}
}
