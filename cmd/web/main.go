package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"content-factory/internal/factory"
)

//go:embed index.html
var pageHTML string

type server struct {
	incomingDir   string
	outputPath    string
	page          *template.Template
	renderMu      sync.Mutex
	rendering     atomic.Bool
	progressDone  atomic.Int64
	progressTotal atomic.Int64
}

type pageData struct {
	Files      []string
	Source     string
	Banner     string
	Parts      int
	Result     *factory.Result
	Outputs    []outputView
	ResultJSON string
	Error      string
}

type outputView struct {
	factory.Output
	URL string
}

func main() {
	s := &server{
		incomingDir: "storage/incoming",
		outputPath:  "storage/output/clip-001.mp4",
		page:        template.Must(template.New("index.html").Parse(pageHTML)),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.index)
	mux.HandleFunc("POST /render", s.render)
	mux.HandleFunc("GET /progress", s.progress)
	mux.HandleFunc("GET /output/{name}", s.output)

	addr := os.Getenv("CONTENT_FACTORY_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	log.Printf("Content Factory debug UI: http://%s", addr)
	log.Fatal((&http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}).ListenAndServe())
}

func (s *server) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.renderPage(w, http.StatusOK, s.defaults())
}

func (s *server) render(w http.ResponseWriter, r *http.Request) {
	data := s.defaults()
	if err := r.ParseForm(); err != nil {
		data.Error = err.Error()
		s.renderPage(w, http.StatusBadRequest, data)
		return
	}
	data.Source = r.FormValue("source")
	data.Banner = r.FormValue("banner")
	parts, err := strconv.Atoi(r.FormValue("parts"))
	if err != nil || parts < 1 || parts > 1000 {
		data.Error = "Количество частей должно быть целым числом от 1 до 1000."
		s.renderPage(w, http.StatusBadRequest, data)
		return
	}
	data.Parts = parts
	sourcePath, err := resolveMedia(s.incomingDir, data.Source)
	if err != nil {
		data.Error = fmt.Sprintf("Source: %v", err)
		s.renderPage(w, http.StatusBadRequest, data)
		return
	}
	bannerPath, err := resolveMedia(s.incomingDir, data.Banner)
	if err != nil {
		data.Error = fmt.Sprintf("Banner: %v", err)
		s.renderPage(w, http.StatusBadRequest, data)
		return
	}

	s.renderMu.Lock()
	s.rendering.Store(true)
	s.progressDone.Store(0)
	s.progressTotal.Store(int64(parts))
	result, err := factory.RunWithProgress(r.Context(), sourcePath, bannerPath, s.outputPath, parts, func(done, total int) {
		s.progressDone.Store(int64(done))
		s.progressTotal.Store(int64(total))
	})
	s.rendering.Store(false)
	s.renderMu.Unlock()
	if err != nil {
		data.Error = err.Error()
		s.renderPage(w, http.StatusInternalServerError, data)
		return
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		data.Error = err.Error()
		s.renderPage(w, http.StatusInternalServerError, data)
		return
	}
	data.Result = &result
	data.ResultJSON = string(encoded)
	stamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	for _, output := range result.Outputs {
		data.Outputs = append(data.Outputs, outputView{
			Output: output,
			URL:    "/output/" + url.PathEscape(filepath.Base(output.Path)) + "?v=" + stamp,
		})
	}
	s.renderPage(w, http.StatusOK, data)
}

func (s *server) progress(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"running": s.rendering.Load(),
		"done":    s.progressDone.Load(),
		"total":   s.progressTotal.Load(),
	})
}

func (s *server) output(w http.ResponseWriter, r *http.Request) {
	path, err := resolveMedia(filepath.Dir(s.outputPath), r.PathValue("name"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, path)
}

func (s *server) defaults() pageData {
	files, err := mediaFiles(s.incomingDir)
	data := pageData{Files: files, Source: "source.mp4", Banner: "banner.mp4", Parts: 10}
	if err != nil {
		data.Error = err.Error()
	}
	return data
}

func (s *server) renderPage(w http.ResponseWriter, status int, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := s.page.Execute(w, data); err != nil {
		log.Printf("render page: %v", err)
	}
}

func mediaFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if _, err := resolveMedia(dir, entry.Name()); err == nil {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}

func resolveMedia(dir, name string) (string, error) {
	if name == "" || filepath.Base(name) != name || !strings.EqualFold(filepath.Ext(name), ".mp4") {
		return "", fmt.Errorf("invalid MP4 filename %q", name)
	}
	path := filepath.Join(dir, name)
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%q is not a regular file", name)
	}
	return path, nil
}
