package http_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestServer_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		url            string
		handler        http.HandlerFunc
		wantStatusCode int
		wantBody       string
	}{
		{
			name:   "GET simple path",
			method: "GET",
			url:    "/test",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("test response"))
			},
			wantStatusCode: http.StatusOK,
			wantBody:       "test response",
		},
		{
			name:   "POST with body",
			method: "POST",
			url:    "/post",
			handler: func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				response := "received: " + string(body)
				w.Write([]byte(response))
			},
			wantStatusCode: http.StatusOK,
			wantBody:       "received: post body",
		},
		{
			name:   "404 Not Found",
			method: "GET",
			url:    "/notfound",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.NotFound(w, r)
			},
			wantStatusCode: http.StatusNotFound,
			wantBody:       "404 page not found\n",
		},
		{
			name:   "Method Not Allowed",
			method: "DELETE",
			url:    "/notallowed",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				w.Write([]byte("ok"))
			},
			wantStatusCode: http.StatusMethodNotAllowed,
			wantBody:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup server
			server := httptest.NewServer(http.HandlerFunc(tt.handler))
			defer server.Close()

			// Create request
			client := server.Client()
			req, err := http.NewRequest(tt.method, server.URL+tt.url, bytes.NewBufferString("post body"))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// Execute request
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}
			defer resp.Body.Close()

			// Validate response status code
			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("Expected status code %d, got %d", tt.wantStatusCode, resp.StatusCode)
			}

			// Validate response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}
			if string(body) != tt.wantBody {
				t.Errorf("Expected response body %q, got %q", tt.wantBody, body)
			}
		})
	}
}

func TestServer_TimeoutHandler(t *testing.T) {
	handler := http.TimeoutHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // simulate long process
		w.Write([]byte("ok"))
	}), 1*time.Second, "request timed out")

	server := httptest.NewServer(handler)
	defer server.Close()

	client := server.Client()
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status code %d, got %d", http.StatusServiceUnavailable, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	if string(body) != "request timed out" {
		t.Errorf("Expected response body %q, got %q", "request timed out", body)
	}
}