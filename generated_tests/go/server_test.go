package http_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServer_Handle(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		url            string
		handlerPattern string
		handlerFunc    http.HandlerFunc
		wantStatusCode int
		wantBody       string
	}{
		{
			name:           "NormalCase",
			method:         "GET",
			url:            "/test",
			handlerPattern: "/test",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("OK"))
			},
			wantStatusCode: http.StatusOK,
			wantBody:       "OK",
		},
		{
			name:           "NotFound",
			method:         "GET",
			url:            "/notfound",
			handlerPattern: "/test",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantStatusCode: http.StatusNotFound,
			wantBody:       "404 page not found\n",
		},
		{
			name:           "MethodNotAllowed",
			method:         "POST",
			url:            "/test",
			handlerPattern: "/test",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantStatusCode: http.StatusMethodNotAllowed,
			wantBody:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc(tt.handlerPattern, tt.handlerFunc)

			req := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("Expected status code %d, got %d", tt.wantStatusCode, resp.StatusCode)
			}

			if string(body) != tt.wantBody {
				t.Errorf("Expected body %q, got %q", tt.wantBody, string(body))
			}
		})
	}
}

func TestServer_HandleContextCancellation(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/cancel", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		select {
		case <-ctx.Done():
			// Simulate work being cancelled
			http.Error(w, ctx.Err().Error(), http.StatusInternalServerError)
		case <-r.Body.Read(make([]byte, 1)):
			// Simulate completing work successfully
		}
	})

	req := httptest.NewRequest("POST", "/cancel", strings.NewReader("data"))
	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Simulate cancellation before the handler finishes
	go func() {
		cancel()
	}()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, resp.StatusCode)
	}

	if !errors.Is(context.Canceled, errors.New(string(body))) {
		t.Errorf("Expected body to contain context.Canceled error, got %q", string(body))
	}
}