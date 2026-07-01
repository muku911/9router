package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// SPAHandler serves the embedded Vite build as a single-page app.
// Static assets (JS, CSS, images) are served normally.
// All other requests fall back to index.html so React Router handles routing.
func SPAHandler() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("web: failed to sub dist FS: " + err.Error())
	}
	fsys := http.FS(sub)
	fileServer := http.FileServer(fsys)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Try to serve the file directly (JS/CSS/images)
		if path != "/" && !strings.HasSuffix(path, "/") {
			// Check if the file exists
			f, err := sub.Open(strings.TrimPrefix(path, "/"))
			if err == nil {
				f.Close()
				// Set cache headers for hashed assets
				if strings.HasPrefix(path, "/assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// SPA fallback: serve index.html for all non-file routes
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
