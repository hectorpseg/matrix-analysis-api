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

// fetchStats sends Q and R to the Node stats service and returns its response.
func fetchStats(Q, R [][]float64) (statsResponse, error) {
	nodeURL := os.Getenv("NODE_API_URL")
	if nodeURL == "" {
		nodeURL = "http://localhost:3001"
	}
	url := strings.TrimRight(nodeURL, "/") + "/api/v1/stats"

	payload := map[string]any{
		"q": Q,
		"r": R,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return statsResponse{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), statsRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return statsResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: statsRequestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return statsResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return statsResponse{}, fmt.Errorf("node returned status %d", resp.StatusCode)
	}

	var stats statsResponse
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return statsResponse{}, err
	}
	return stats, nil
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(newApp().Listen("0.0.0.0:" + port))
}
