package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
)

// Request and response types for POST /api/v1/qr.

type qrRequest struct {
	Matrix [][]float64 `json:"matrix"`
}

type globalStats struct {
	Max     float64 `json:"max"`
	Min     float64 `json:"min"`
	Average float64 `json:"average"`
	Sum     float64 `json:"sum"`
}

type diagonalStat struct {
	IsDiagonal bool `json:"isDiagonal"`
}

type statsResponse struct {
	Global globalStats  `json:"global"`
	Q      diagonalStat `json:"q"`
	R      diagonalStat `json:"r"`
}

type qrResponse struct {
	Q     [][]float64   `json:"q"`
	R     [][]float64   `json:"r"`
	Stats statsResponse `json:"stats"`
}

type errorResponse struct {
	Error string `json:"error"`
}

//go:embed web/index.html
var indexHTML []byte

// statsRequestTimeout is the maximum time allowed for the Go -> Node request.
// It is package-level so tests can override it without adding public config.
var statsRequestTimeout = 10 * time.Second

// statsRetryDeadline is the maximum time allowed for all retries of a single
// stats request. It is package-level so tests can override it.
var statsRetryDeadline = 40 * time.Second

// nodeWarmupTimeout is the maximum time allowed for the background Node
// health check triggered from the root endpoint.
var nodeWarmupTimeout = 3 * time.Second

// nodeBaseURL returns the configured Node API URL with no trailing slash.
func nodeBaseURL() string {
	nodeURL := os.Getenv("NODE_API_URL")
	if nodeURL == "" {
		nodeURL = "http://localhost:3001"
	}
	return strings.TrimRight(nodeURL, "/")
}

func newApp() *fiber.App {
	app := fiber.New()
	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.Get("/", handleIndex)
	app.Post("/api/v1/qr", handleQR)
	return app
}

func handleIndex(c fiber.Ctx) error {
	// Trigger a background warm-up of the Node stats service so that cold
	// Render Free deployments have a chance to wake before the QR endpoint
	// needs them. The response is not blocked on the warm-up result.
	go warmupNode(context.Background(), &http.Client{Timeout: nodeWarmupTimeout}, nodeBaseURL())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.Send(indexHTML)
}

func handleQR(c fiber.Ctx) error {
	var req qrRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "invalid matrix"})
	}

	if !isValidMatrix(req.Matrix) {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "invalid matrix"})
	}

	Q, R := QR(req.Matrix)
	if Q == nil || R == nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "invalid matrix"})
	}

	if !isFiniteMatrix(Q) || !isFiniteMatrix(R) {
		return c.Status(http.StatusInternalServerError).JSON(errorResponse{Error: "internal server error"})
	}

	stats, err := fetchStats(Q, R)
	if err != nil {
		return c.Status(http.StatusBadGateway).JSON(errorResponse{Error: "stats service unavailable"})
	}

	return c.JSON(qrResponse{
		Q:     Q,
		R:     R,
		Stats: stats,
	})
}

// isValidMatrix checks the input constraints required before calling QR.
// Shape validation mirrors QR's behaviour; this layer adds the finite-value check.
func isValidMatrix(A [][]float64) bool {
	if len(A) == 0 || len(A[0]) == 0 {
		return false
	}
	width := len(A[0])
	for _, row := range A {
		if len(row) != width {
			return false
		}
		for _, v := range row {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return false
			}
		}
	}
	return true
}

// isFiniteMatrix verifies that every entry is a normal finite number.
func isFiniteMatrix(A [][]float64) bool {
	for _, row := range A {
		for _, v := range row {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return false
			}
		}
	}
	return true
}

// warmupNode sends a best-effort health request to the Node service.
// Errors and timeouts are ignored; the QR endpoint retries its own stats call.
func warmupNode(ctx context.Context, client *http.Client, baseURL string) {
	ctx, cancel := context.WithTimeout(ctx, nodeWarmupTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/health", nil)
	if err != nil {
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

// retryDelay returns the wait before retry attempt n (1-based).
// Delays follow 0, 1s, 2s, 4s, ... capped at 8s.
func retryDelay(attempt int) time.Duration {
	if attempt <= 1 {
		return 0
	}
	d := time.Duration(1<<(attempt-2)) * time.Second
	const maxDelay = 8 * time.Second
	if d > maxDelay {
		return maxDelay
	}
	return d
}

// fetchStats sends Q and R to the Node stats service and returns its response.
// It retries transient failures (network errors, timeouts, HTTP 5xx) with
// exponential backoff until statsRetryDeadline. HTTP 4xx responses are not
// retried because they indicate request/application errors.
func fetchStats(Q, R [][]float64) (statsResponse, error) {
	url := nodeBaseURL() + "/api/v1/stats"

	payload := map[string]any{
		"q": Q,
		"r": R,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return statsResponse{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), statsRetryDeadline)
	defer cancel()

	client := &http.Client{Timeout: statsRequestTimeout}

	for attempt := 1; ; attempt++ {
		stats, status, err := doStatsRequest(ctx, client, url, body)
		if err == nil {
			return stats, nil
		}
		// Do not retry client errors; they mean the request itself is bad.
		if status >= http.StatusBadRequest && status < http.StatusInternalServerError {
			return statsResponse{}, err
		}

		select {
		case <-ctx.Done():
			return statsResponse{}, ctx.Err()
		case <-time.After(retryDelay(attempt)):
		}
	}
}

// doStatsRequest performs a single stats call. It returns the HTTP status so
// callers can decide whether the failure is retryable.
func doStatsRequest(ctx context.Context, client *http.Client, url string, body []byte) (statsResponse, int, error) {
	reqCtx, cancel := context.WithTimeout(ctx, statsRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return statsResponse{}, 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return statsResponse{}, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return statsResponse{}, resp.StatusCode, fmt.Errorf("node returned status %d", resp.StatusCode)
	}

	var stats statsResponse
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return statsResponse{}, resp.StatusCode, err
	}
	return stats, resp.StatusCode, nil
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(newApp().Listen("0.0.0.0:" + port))
}
