package http_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestServer_Handle_SuccessCases tests the successful handling of different HTTP methods and patterns.
func TestServer_Handle_SuccessCases(t *testing.T) {
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
			name:           "GetRequest_Success",
			method:         "GET",
			url:            "/success",
			handlerPattern: "/success",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("Success"))
			},
			wantStatusCode: http.StatusOK,
			wantBody:       "Success",
		},
		{
			name:           "PostRequest_Success",
			method:         "POST",
			url:            "/create",
			handlerPattern: "/create",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte("Created"))
			},
			wantStatusCode: http.StatusCreated,
			wantBody:       "Created",
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
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("Expected status code %d, got %d", tt.wantStatusCode, resp.StatusCode)
			}

			if string(body) != tt.wantBody {
				t.Errorf("Expected body %q, got %q", tt.wantBody, string(body))
			}
		})
	}
}

// TestServer_Handle_ErrorCases tests the handling of errors such as not found or method not allowed.
func TestServer_Handle_ErrorCases(t *testing.T) {
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
			name:           "PageNotFound",
			method:         "GET",
			url:            "/does-not-exist",
			handlerPattern: "/exists",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantStatusCode: http.StatusNotFound,
			wantBody:       "404 page not found\n",
		},
		{
			name:           "MethodNotAllowed",
			method:         "DELETE",
			url:            "/create",
			handlerPattern: "/create",
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
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("Expected status code %d, got %d", tt.wantStatusCode, resp.StatusCode)
			}

			if string(body) != tt.wantBody {
				t.Errorf("Expected body %q, got %q", tt.wantBody, string(body))
			}
		})
	}
}

// TestServer_HandleContextCancellation tests the server's ability to handle request context cancellation.
func TestServer_HandleContextCancellation(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/cancel", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		select {
		case <-ctx.Done():
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(ctx.Err().Error()))
		case <-time.After(100 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
		}
	})

	req := httptest.NewRequest("POST", "/cancel", strings.NewReader("data"))
	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Cancel the context before the handler has a chance to finish
	cancel()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, resp.StatusCode)
	}

	expectedError := context.Canceled.Error()
	if !strings.Contains(string(body), expectedError) {
		t.Errorf("Expected body to contain %q error, got %q", expectedError, string(body))
	}
}