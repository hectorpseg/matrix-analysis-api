package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHealth(t *testing.T) {
	resp, err := newApp().Test(httptest.NewRequest(http.MethodGet, "/health", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Fatalf("body = %v, want status ok", body)
	}
}

func TestQRHandlerValidMatrix(t *testing.T) {
	expectedStats := statsResponse{
		Global: globalStats{Max: 5, Min: 1, Average: 3, Sum: 12},
		Q:      diagonalStat{IsDiagonal: false},
		R:      diagonalStat{IsDiagonal: true},
	}

	nodeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/stats" {
			t.Errorf("node path = %s, want /api/v1/stats", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content-type = %s, want application/json", r.Header.Get("Content-Type"))
		}

		var payload struct {
			Q [][]float64 `json:"q"`
			R [][]float64 `json:"r"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("failed to decode node request: %v", err)
		}
		if len(payload.Q) == 0 || len(payload.R) == 0 {
			t.Errorf("node request missing q or r")
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(expectedStats); err != nil {
			t.Errorf("failed to encode node response: %v", err)
		}
	}))
	defer nodeServer.Close()

	t.Setenv("NODE_API_URL", nodeServer.URL)

	app := newApp()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/qr", strings.NewReader(`{"matrix":[[1,2],[3,4]]}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body qrResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	if len(body.Q) == 0 || len(body.R) == 0 {
		t.Fatal("response missing q or r")
	}
	if body.Stats != expectedStats {
		t.Fatalf("stats = %+v, want %+v", body.Stats, expectedStats)
	}

	matricesApproxEqual(t, "reconstruction", multiply(body.Q, body.R), [][]float64{{1, 2}, {3, 4}}, 1e-10)
}

func TestQRHandlerInvalidMatrix(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"empty matrix", `{"matrix":[]}`},
		{"empty row", `{"matrix":[[]]}`},
		{"ragged rows", `{"matrix":[[1,2],[3]]}`},
		{"missing matrix", `{}`},
		{"non-finite value", `{"matrix":[[1,2],[3,1e309]]}`},
		{"matrix is string", `{"matrix":"not an array"}`},
		{"nested string", `{"matrix":[[1,2],[3,"x"]]}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := newApp()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/qr", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", resp.StatusCode)
			}

			var body errorResponse
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.Error != "invalid matrix" {
				t.Fatalf("error = %q, want invalid matrix", body.Error)
			}
		})
	}
}

func TestQRHandlerNodeReturns500(t *testing.T) {
	originalDeadline := statsRetryDeadline
	statsRetryDeadline = 200 * time.Millisecond
	defer func() { statsRetryDeadline = originalDeadline }()

	nodeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte(`{"error":"node boom"}`)); err != nil {
			t.Errorf("failed to write node response: %v", err)
		}
	}))
	defer nodeServer.Close()

	t.Setenv("NODE_API_URL", nodeServer.URL)

	app := newApp()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/qr", strings.NewReader(`{"matrix":[[1,2],[3,4]]}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	assert502(t, resp)
}

func TestQRHandlerNodeUnavailable(t *testing.T) {
	originalDeadline := statsRetryDeadline
	statsRetryDeadline = 200 * time.Millisecond
	defer func() { statsRetryDeadline = originalDeadline }()

	t.Setenv("NODE_API_URL", "http://127.0.0.1:1")

	app := newApp()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/qr", strings.NewReader(`{"matrix":[[1,2],[3,4]]}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	assert502(t, resp)
}

func TestQRHandlerNodeTimeout(t *testing.T) {
	original := statsRequestTimeout
	statsRequestTimeout = 50 * time.Millisecond
	defer func() { statsRequestTimeout = original }()

	originalDeadline := statsRetryDeadline
	statsRetryDeadline = 200 * time.Millisecond
	defer func() { statsRetryDeadline = originalDeadline }()

	nodeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer nodeServer.Close()

	t.Setenv("NODE_API_URL", nodeServer.URL)

	app := newApp()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/qr", strings.NewReader(`{"matrix":[[1,2],[3,4]]}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	assert502(t, resp)
}

func assert502(t *testing.T, resp *http.Response) {
	t.Helper()
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", resp.StatusCode)
	}
	var body errorResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error != "stats service unavailable" {
		t.Fatalf("error = %q, want stats service unavailable", body.Error)
	}
}

func TestFetchStatsRetriesTransientFailureThenSucceeds(t *testing.T) {
	expectedStats := statsResponse{
		Global: globalStats{Max: 5, Min: 1, Average: 3, Sum: 12},
		Q:      diagonalStat{IsDiagonal: false},
		R:      diagonalStat{IsDiagonal: true},
	}

	attempts := 0
	nodeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(expectedStats); err != nil {
			t.Errorf("failed to encode node response: %v", err)
		}
	}))
	defer nodeServer.Close()

	t.Setenv("NODE_API_URL", nodeServer.URL)

	originalDeadline := statsRetryDeadline
	statsRetryDeadline = 5 * time.Second
	defer func() { statsRetryDeadline = originalDeadline }()

	stats, err := fetchStats([][]float64{{1, 0}, {0, 1}}, [][]float64{{1, 0}, {0, 1}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	if stats != expectedStats {
		t.Fatalf("stats = %+v, want %+v", stats, expectedStats)
	}
}

func TestFetchStatsPersistentFailureReturnsError(t *testing.T) {
	attempts := 0
	nodeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer nodeServer.Close()

	t.Setenv("NODE_API_URL", nodeServer.URL)

	original := statsRequestTimeout
	statsRequestTimeout = 50 * time.Millisecond
	defer func() { statsRequestTimeout = original }()

	originalDeadline := statsRetryDeadline
	statsRetryDeadline = 1500 * time.Millisecond
	defer func() { statsRetryDeadline = originalDeadline }()

	_, err := fetchStats([][]float64{{1, 0}, {0, 1}}, [][]float64{{1, 0}, {0, 1}})
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts < 2 {
		t.Fatalf("attempts = %d, want at least 2", attempts)
	}
}

func TestFetchStats4xxNotRetried(t *testing.T) {
	attempts := 0
	nodeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer nodeServer.Close()

	t.Setenv("NODE_API_URL", nodeServer.URL)

	_, err := fetchStats([][]float64{{1, 0}, {0, 1}}, [][]float64{{1, 0}, {0, 1}})
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestHandleIndexDoesNotBlockOnWarmup(t *testing.T) {
	healthCalled := make(chan struct{})
	handlerReleased := make(chan struct{})
	nodeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			close(healthCalled)
			<-handlerReleased
		}
	}))
	defer nodeServer.Close()
	defer close(handlerReleased)

	t.Setenv("NODE_API_URL", nodeServer.URL)

	app := newApp()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	select {
	case <-healthCalled:
		// warm-up was triggered in the background.
	case <-time.After(2 * time.Second):
		t.Fatal("warm-up request was not triggered")
	}
}

func TestWarmupNodeSucceedsAndIgnoresErrors(t *testing.T) {
	called := 0
	nodeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			called++
			if called == 1 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer nodeServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// A failed warm-up must not panic or return an error.
	warmupNode(ctx, nodeServer.Client(), nodeServer.URL)
	if called != 1 {
		t.Fatalf("warmup calls = %d, want 1", called)
	}

	// A successful warm-up increments the call counter.
	warmupNode(context.Background(), nodeServer.Client(), nodeServer.URL)
	if called != 2 {
		t.Fatalf("warmup calls = %d, want 2", called)
	}
}
