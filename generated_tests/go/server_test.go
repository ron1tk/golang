package http_test

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type mockResponseWriter struct {
	headerMap http.Header
	body      bytes.Buffer
	status    int
}

func (m *mockResponseWriter) Header() http.Header {
	return m.headerMap
}

func (m *mockResponseWriter) Write(data []byte) (int, error) {
	return m.body.Write(data)
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.status = statusCode
}

type mockHijacker struct {
	mockResponseWriter
}

func (m *mockHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, errors.New("hijack error")
}

func TestResponseWriterWriteHeader(t *testing.T) {
	t.Run("NormalCase", func(t *testing.T) {
		w := &mockResponseWriter{headerMap: make(http.Header)}
		w.WriteHeader(http.StatusOK)

		if w.status != http.StatusOK {
			t.Errorf("expected status %d; got %d", http.StatusOK, w.status)
		}
	})

	t.Run("WriteHeaderAfterWrite", func(t *testing.T) {
		w := &mockResponseWriter{headerMap: make(http.Header)}
		w.Write([]byte("test")) // This should implicitly set the status to http.StatusOK
		w.WriteHeader(http.StatusCreated) // This should have no effect as WriteHeader should only work on the first call

		if w.status != http.StatusOK {
			t.Errorf("expected status %d; got %d", http.StatusOK, w.status)
		}
	})
}

func TestResponseWriterWrite(t *testing.T) {
	t.Run("NormalCase", func(t *testing.T) {
		w := &mockResponseWriter{headerMap: make(http.Header)}
		data := []byte("hello world")
		n, err := w.Write(data)

		if err != nil {
			t.Errorf("expected no error; got %v", err)
		}

		if n != len(data) {
			t.Errorf("expected write length %d; got %d", len(data), n)
		}

		if !bytes.Equal(w.body.Bytes(), data) {
			t.Errorf("expected body %s; got %s", data, w.body.Bytes())
		}
	})

	t.Run("WriteAfterHijack", func(t *testing.T) {
		w := &mockHijacker{mockResponseWriter{headerMap: make(http.Header)}}
		_, err := w.Write([]byte("data"))
		if !errors.Is(err, http.ErrHijacked) {
			t.Errorf("expected hijack error; got %v", err)
		}
	})
}

func TestHijacker(t *testing.T) {
	t.Run("SupportsHijack", func(t *testing.T) {
		w := &mockHijacker{mockResponseWriter{headerMap: make(http.Header)}}
		_, _, err := w.Hijack()

		if !strings.Contains(err.Error(), "hijack error") {
			t.Errorf("expected hijack error; got %v", err)
		}
	})
}

func TestFlusher(t *testing.T) {
	t.Run("FlushData", func(t *testing.T) {
		// Mocking http.ResponseWriter that implements Flusher
		w := struct {
			mockResponseWriter
			http.Flusher
		}{
			mockResponseWriter: mockResponseWriter{headerMap: make(http.Header)},
			Flusher:            mockFlusher{},
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("partial "))
		w.Flush()
		w.Write([]byte("data"))

		expected := "partial data"
		if w.body.String() != expected {
			t.Errorf("expected body %s; got %s", expected, w.body.String())
		}
	})
}

type mockFlusher struct{}

func (f mockFlusher) Flush() {}

func TestCloseNotifier(t *testing.T) {
	t.Run("NotifyOnClose", func(t *testing.T) {
		// Mocking http.ResponseWriter that implements CloseNotifier
		w := struct {
			mockResponseWriter
			http.CloseNotifier
		}{
			mockResponseWriter: mockResponseWriter{headerMap: make(http.Header)},
			CloseNotifier:      mockCloseNotifier{},
		}

		notifyCh := w.CloseNotify()
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-notifyCh:
				// Success
			case <-time.After(1 * time.Second):
				t.Error("expected close notification; timeout reached")
			}
		}()

		// Simulate connection close
		w.CloseNotifier.(mockCloseNotifier).close()
		wg.Wait()
	})
}

type mockCloseNotifier struct {
	ch chan bool
}

func (m mockCloseNotifier) CloseNotify() <-chan bool {
	if m.ch == nil {
		m.ch = make(chan bool, 1)
	}
	return m.ch
}

func (m mockCloseNotifier) close() {
	if m.ch != nil {
		m.ch <- true
		close(m.ch)
	}
}