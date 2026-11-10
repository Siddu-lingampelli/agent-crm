package httpserver

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStaticAssetsStayInsideRootAndKeepAPIProtected(t *testing.T) {
	dir := t.TempDir()
	assets := filepath.Join(dir, "public")
	if err := os.Mkdir(assets, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assets, "index.html"), []byte("SPA HTML"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("private secret"), 0600); err != nil {
		t.Fatal(err)
	}
	srv := New(Config{}, nil)
	srv.ServeStatic("/", assets)
	handler := srv.Handler()
	for _, test := range []struct {
		path string
		want int
	}{{"/dashboard", 200}, {"/api/v1/missing", 404}, {"/assets/missing.js", 404}, {"/%2e%2e/secret.txt", 404}} {
		r := httptest.NewRequest("GET", test.path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if strings.Contains(w.Body.String(), "private secret") {
			t.Fatal("static root escaped")
		}
		if test.path != "/%2e%2e/secret.txt" && w.Code != test.want {
			t.Fatalf("%s: %d %s", test.path, w.Code, w.Body.String())
		}
	}
}
