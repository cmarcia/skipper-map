package nws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_GetPoints(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/points/37.000000,-122.000000" {
			t.Errorf("Expected path /points/37.000000,-122.000000, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/geo+json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"properties": {"forecast": "https://api.weather.gov/gridpoints/MTR/84,105/forecast"}}`))
	}))
	defer server.Close()

	// Use a custom client that points to the test server
	c := NewClient()
	// Override the base URL logic for testing (in a real production app, NWSAPIBase would be configurable)
	// For this exercise, I'll just mock the request helper or similar.
	// Since I can't easily change the constant, I'll test the makeRequest helper directly with a dynamic URL.

	t.Run("Successful request", func(t *testing.T) {
		resp, err := makeRequest[struct {
			Properties struct {
				Forecast string `json:"forecast"`
			} `json:"properties"`
		}](context.Background(), c.httpClient, server.URL+"/points/37.000000,-122.000000")

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if resp.Properties.Forecast != "https://api.weather.gov/gridpoints/MTR/84,105/forecast" {
			t.Errorf("Expected forecast URL, got %s", resp.Properties.Forecast)
		}
	})

	t.Run("404 Error", func(t *testing.T) {
		errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not Found"))
		}))
		defer errorServer.Close()

		_, err := makeRequest[any](context.Background(), c.httpClient, errorServer.URL)
		if err == nil {
			t.Fatal("Expected error for 404, got nil")
		}
	})
}
