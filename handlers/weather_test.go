package handlers

import (
	"context"
	"testing"
	"skipperMCP/models"
	"skipperMCP/nws"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestWeatherHandler_Validation(t *testing.T) {
	client := nws.NewClient()
	handler := NewWeatherHandler(client)
	ctx := context.Background()

	t.Run("Invalid Latitude", func(t *testing.T) {
		result, _, _ := handler.GetForecast(ctx, &mcp.CallToolRequest{}, models.ForecastInput{
			Latitude: 100,
		})
		if !result.IsError {
			t.Error("Expected error for latitude 100")
		}
	})

	t.Run("Invalid State Code", func(t *testing.T) {
		result, _, _ := handler.GetAlerts(ctx, &mcp.CallToolRequest{}, models.AlertsInput{
			State: "CAL",
		})
		if !result.IsError {
			t.Error("Expected error for state code CAL")
		}
	})
}

func TestFormatPeriod(t *testing.T) {
	p := models.ForecastPeriod{
		Name:             "Today",
		Temperature:      72,
		TemperatureUnit:  "F",
		WindSpeed:        "10 mph",
		WindDirection:    "NW",
		DetailedForecast: "Sunny and clear.",
	}
	expected := "Today:\nTemperature: 72°F\nWind: 10 mph NW\nForecast: Sunny and clear."
	if got := formatPeriod(p); got != expected {
		t.Errorf("Expected %q, got %q", expected, got)
	}
}
