// Package web serves the browser interface embedded in the Go executable.
package web

import (
	"embed"
	"net/http"
)

//go:embed index.html styles.css app.js photo.js engine.js session.js solver-worker.js
var assets embed.FS

func Handler() http.Handler { return http.FileServer(http.FS(assets)) }
