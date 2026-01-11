package web

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

//go:embed dist/*
var distFS embed.FS

func Handler() http.Handler {
	distRoot, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}

	// Use a custom handler for SPA support
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Try to open the file
		f, err := distRoot.Open(path)
		if err == nil {
			_ = f.Close() // Ignore close error
			// File exists, serve it
			// Manually set content type to be safe, as http.FileServer might guess wrong on embedded sometimes
			ctype := mime.TypeByExtension(filepath.Ext(path))
			if ctype == "" {
				// Fallback manual checks
				if strings.HasSuffix(path, ".js") {
					ctype = "application/javascript"
				} else if strings.HasSuffix(path, ".css") {
					ctype = "text/css"
				} else if strings.HasSuffix(path, ".html") {
					ctype = "text/html"
				} else if strings.HasSuffix(path, ".svg") {
					ctype = "image/svg+xml"
				}
			}
			if ctype != "" {
				w.Header().Set("Content-Type", ctype)
			}

			content, err := fs.ReadFile(distRoot, path)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			_, _ = w.Write(content) // Ignore write error (already committed response)
			return
		}

		// Fallback to index.html
		content, err := fs.ReadFile(distRoot, "index.html")
		if err != nil {
			http.Error(w, "Internal Server Error: index.html not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write(content) // Ignore write error (already committed response)
	})
}
