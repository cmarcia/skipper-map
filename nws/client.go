package nws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"skipperMCP/models"
)

const (
	NWSAPIBase = "https://api.weather.gov"
	UserAgent  = "weather-app/1.0"
	MaxRetries = 3
)

// Client handles communication with the National Weather Service API.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new NWS client with production-standard timeouts and connection pooling.
func NewClient() *Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
	}
}

// GetPoints fetches the forecast URL for a given latitude and longitude.
func (c *Client) GetPoints(ctx context.Context, lat, lon float64) (*models.PointsResponse, error) {
	url := fmt.Sprintf("%s/points/%f,%f", NWSAPIBase, lat, lon)
	return makeRequest[models.PointsResponse](ctx, c.httpClient, url)
}

// GetForecast fetches the detailed forecast from a forecast URL.
func (c *Client) GetForecast(ctx context.Context, url string) (*models.ForecastResponse, error) {
	return makeRequest[models.ForecastResponse](ctx, c.httpClient, url)
}

// GetAlerts fetches active alerts for a specific US state.
func (c *Client) GetAlerts(ctx context.Context, state string) (*models.AlertsResponse, error) {
	url := fmt.Sprintf("%s/alerts/active/area/%s", NWSAPIBase, state)
	return makeRequest[models.AlertsResponse](ctx, c.httpClient, url)
}

// makeRequest is a generic helper that handles HTTP requests, retries, and decoding.
func makeRequest[T any](ctx context.Context, client *http.Client, url string) (*T, error) {
	var lastErr error

	for i := 0; i < MaxRetries; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("User-Agent", UserAgent)
		req.Header.Set("Accept", "application/geo+json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond) // Basic backoff
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var result T
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			return &result, nil
		}

		// Handle transient errors with retry
		if resp.StatusCode >= 500 {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
			continue
		}

		// For 4xx errors, don't retry
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("after %d attempts, request failed: %w", MaxRetries, lastErr)
}
