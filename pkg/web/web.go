package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var staticFS embed.FS

func Handler() http.Handler {
	fsys, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(fsys))
}
