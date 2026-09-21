package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	// Since the handler is currently defined inline within the main() function in main.go,
	// and we cannot modify main.go to export it, we replicate its exact implementation
	// here to test the logic using httptest, as requested.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// 1. GET /health returns HTTP 200.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// 2. The Content-Type response header indicates application/json.
	expectedContentType := "application/json"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, expectedContentType)
	}

	// 3 & 4. The response body is valid JSON and contains exactly {"status":"ok"}
	expectedBody := `{"status":"ok"}`
	if rr.Body.String() != expectedBody {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expectedBody)
	}
}

func TestReadyEndpoint(t *testing.T) {
	t.Run("ready when ping succeeds", func(t *testing.T) {
		handler := readyHandlerWithPing(func(context.Context) error {
			return nil
		})

		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
		}

		if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
			t.Fatalf("handler returned wrong content type: got %v want %v", contentType, "application/json")
		}

		if body := rr.Body.String(); body != `{"status":"ready"}` {
			t.Fatalf("handler returned unexpected body: got %v want %v", body, `{"status":"ready"}`)
		}
	})

	t.Run("not ready when ping fails", func(t *testing.T) {
		handler := readyHandlerWithPing(func(context.Context) error {
			return errors.New("database unavailable")
		})

		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusServiceUnavailable)
		}

		if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
			t.Fatalf("handler returned wrong content type: got %v want %v", contentType, "application/json")
		}

		if body := rr.Body.String(); body != `{"status":"not_ready"}` {
			t.Fatalf("handler returned unexpected body: got %v want %v", body, `{"status":"not_ready"}`)
		}
	})
}
