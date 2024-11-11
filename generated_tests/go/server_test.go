package http_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServeMux_HandleFunc_Pattern(t *testing.T) {
	mux := http.NewServeMux()
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "testing")
	}

	tests := []struct {
		pattern string
		url     string
		want    int // expected HTTP status code
	}{
		{"/path", "/path", http.StatusOK},
		{"/path", "/path/", http.StatusNotFound},
		{"/path/", "/path", http.StatusMovedPermanently},
		{"/path/", "/path/", http.StatusOK},
	}

	for _, tt := range tests {
		mux.HandleFunc(tt.pattern, handler)
		req := httptest.NewRequest("GET", tt.url, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != tt.want {
			t.Errorf("Pattern %q, URL %q: got status %d, want %d", tt.pattern, tt.url, resp.StatusCode, tt.want)
		}
	}
}

func TestServeMux_HandleFunc_MethodNotAllowed(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/path", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		io.WriteString(w, "testing")
	})

	tests := []struct {
		method string
		want   int
	}{
		{"GET", http.StatusOK},
		{"POST", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, "/path", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != tt.want {
			t.Errorf("Method %q: got status %d, want %d", tt.method, resp.StatusCode, tt.want)
		}
	}
}

func TestServeMux_HandleFunc_QueryParameters(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/path", func(w http.ResponseWriter, r *http.Request) {
		v := r.URL.Query().Get("param")
		io.WriteString(w, v)
	})

	tests := []struct {
		query string
		want  string
	}{
		{"param=value", "value"},
		{"other=stuff&param=value", "value"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/path?"+tt.query, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		resp := w.Result()
		body := new(bytes.Buffer)
		body.ReadFrom(resp.Body)
		if body.String() != tt.want {
			t.Errorf("Query %q: got body %q, want %q", tt.query, body.String(), tt.want)
		}
	}
}

func TestServeMux_HandleFunc_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/path", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "testing")
	})

	req := httptest.NewRequest("GET", "/notfound", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Got status %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestServeMux_HandleFunc_RedirectTrailingSlash(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/path/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "testing")
	})

	req := httptest.NewRequest("GET", "/path", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMovedPermanently {
		t.Errorf("Got status %d, want %d", resp.StatusCode, http.StatusMovedPermanently)
	}
}

func TestServeMux_HandleFunc_RedirectCleanPath(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/path/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "testing")
	})

	req := httptest.NewRequest("GET", "/path//", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMovedPermanently {
		t.Errorf("Got status %d, want %d", resp.StatusCode, http.StatusMovedPermanently)
	}
}

func TestServeMux_HandleFunc_ErrorHandling(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	})

	req := httptest.NewRequest("GET", "/error", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Got status %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

func TestServeMux_HandleFunc_MockError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/mock", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		io.WriteString(w, "mock error")
	})

	req := httptest.NewRequest("GET", "/mock", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("Got status %d, want %d", resp.StatusCode, http.StatusBadGateway)
	}

	body := new(bytes.Buffer)
	body.ReadFrom(resp.Body)
	if body.String() != "mock error" {
		t.Errorf("Got body %q, want %q", body.String(), "mock error")
	}
}

func TestServeMux_HandleFunc_CustomNotFoundHandler(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/path", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "testing")
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		io.WriteString(w, "root")
	})

	tests := []struct {
		path string
		want string
	}{
		{"/", "root"},
		{"/notfound", "404 page not found\n"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		resp := w.Result()
		body := new(bytes.Buffer)
		body.ReadFrom(resp.Body)
		if body.String() != tt.want {
			t.Errorf("Path %q: got body %q, want %q", tt.path, body.String(), tt.want)
		}
	}
}