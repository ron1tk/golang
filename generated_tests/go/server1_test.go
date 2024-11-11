package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestServer_HandleSuccessCases tests the successful handling of requests by the server.
func TestServer_HandleSuccessCases(t *testing.T) {
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
			name:           "GET_Method_OK",
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc(tc.handlerPattern, tc.handlerFunc)

			req := httptest.NewRequest(tc.method, tc.url, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatusCode {
				t.Errorf("Expected status code %d, got %d", tc.wantStatusCode, resp.StatusCode)
			}

			if string(body) != tc.wantBody {
				t.Errorf("Expected body %q, got %q", tc.wantBody, string(body))
			}
		})
	}
}

// TestServer_HandleFailureCases tests the server's handling of various failure scenarios.
func TestServer_HandleFailureCases(t *testing.T) {
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
			name:           "NotFound",
			method:         "GET",
			url:            "/doesNotExist",
			handlerPattern: "/test",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {},
			wantStatusCode: http.StatusNotFound,
			wantBody:       "404 page not found\n",
		},
		{
			name:           "MethodNotAllowed",
			method:         "POST",
			url:            "/test",
			handlerPattern: "/test",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {},
			wantStatusCode: http.StatusMethodNotAllowed,
			wantBody:       "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc(tc.handlerPattern, tc.handlerFunc)

			req := httptest.NewRequest(tc.method, tc.url, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatusCode {
				t.Errorf("Expected status code %d, got %d", tc.wantStatusCode, resp.StatusCode)
			}

			if string(body) != tc.wantBody {
				t.Errorf("Expected body %q, got %q", tc.wantBody, string(body))
			}
		})
	}
}

// TestServer_HandleContextCancellation tests the server's ability to handle request context cancellation.
func TestServer_HandleContextCancellation(t *testing.T) {
	handlerCalled := false

	mux := http.NewServeMux()
	mux.HandleFunc("/cancel", func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		ctx := r.Context()
		<-ctx.Done()
		http.Error(w, ctx.Err().Error(), http.StatusInternalServerError)
	})

	req := httptest.NewRequest("POST", "/cancel", strings.NewReader("data"))
	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Cancel the context before the request is handled
	cancel()

	mux.ServeHTTP(w, req)

	if !handlerCalled {
		t.Errorf("Expected the handler to be called")
	}

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, resp.StatusCode)
	}

	expectedErrMsg := context.Canceled.Error()
	if !strings.Contains(string(body), expectedErrMsg) {
		t.Errorf("Expected body to contain %q, got %q", expectedErrMsg, string(body))
	}
}