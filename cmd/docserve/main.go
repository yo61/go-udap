// Command docserve previews docs/site/out as GitHub Pages serves it, under /go-udap.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

// basePath must match the default in docs/site/next.config.mjs.
const (
	basePath = "/go-udap"
	addr     = "localhost:3000"
)

func main() {
	dir := "docs/site/out"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if err := run(dir); err != nil {
		fmt.Fprintln(os.Stderr, "docserve:", err)
		os.Exit(1)
	}
}

func run(dir string) error {
	site := os.DirFS(dir)
	if err := checkSite(site, dir); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "serving %s at http://%s%s/\n", dir, addr, basePath)
	return http.ListenAndServe(addr, newHandler(site))
}

func checkSite(site fs.FS, dir string) error {
	index, err := fs.ReadFile(site, "index.html")
	if err != nil {
		return fmt.Errorf("no exported site in %s (%w): run `pnpm build` in docs/site first", dir, err)
	}
	if !bytes.Contains(index, []byte(basePath+"/_next/")) {
		return fmt.Errorf(
			"%s was not built with the %s basePath: rebuild with BASE_PATH unset (`pnpm build` in docs/site)",
			dir, basePath)
	}
	return nil
}

func newHandler(site fs.FS) http.Handler {
	p := pages{site: site, files: http.FileServerFS(site)}
	mux := http.NewServeMux()
	mux.Handle(basePath+"/", http.StripPrefix(basePath, p))
	mux.HandleFunc(basePath, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, basePath+"/", http.StatusMovedPermanently)
	})
	// Not 301: browsers would cache it and break docs:dev, which serves / on the same port.
	mux.HandleFunc("/{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, basePath+"/", http.StatusFound)
	})
	mux.HandleFunc("/", p.notFound)
	return mux
}

type pages struct {
	site  fs.FS
	files http.Handler
}

func (p pages) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if name == "" {
		name = "."
	}
	info, err := fs.Stat(p.site, name)
	if err == nil && info.IsDir() {
		_, err = fs.Stat(p.site, path.Join(name, "index.html"))
	}
	switch {
	case errors.Is(err, fs.ErrNotExist):
		p.notFound(w, r)
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	default:
		p.files.ServeHTTP(w, r)
	}
}

func (p pages) notFound(w http.ResponseWriter, r *http.Request) {
	body, err := fs.ReadFile(p.site, "404.html")
	switch {
	case errors.Is(err, fs.ErrNotExist):
		http.NotFound(w, r)
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	default:
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(body)
	}
}
