package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEmbeddedAssets(t *testing.T) {
	for _, tt := range []struct{ path, contentType, content string }{
		{"/", "text/html", "id=\"board\""},
		{"/styles.css", "text/css", ".board"},
		{"/app.js", "javascript", "/api/solve"},
		{"/photo.js", "javascript", "tessedit_char_whitelist"},
	} {
		w := httptest.NewRecorder()
		Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Header().Get("Content-Type"), tt.contentType) || !strings.Contains(w.Body.String(), tt.content) {
			t.Fatalf("asset %s failed: %d", tt.path, w.Code)
		}
	}
	for _, path := range []string{"/assets.go", "/missing.js"} {
		w := httptest.NewRecorder()
		Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("unexpected file served: %s", path)
		}
	}
}
