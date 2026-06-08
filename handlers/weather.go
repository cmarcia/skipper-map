package handlers

import (
	"context"
	"fmt"
	"strings"

	"skipperMCP/models"
	"skipperMCP/nws"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// WeatherHandler manages the MCP tool logic.
type WeatherHandler struct {
	nwsClient *nws.Client
}

// NewWeatherHandler creates a new handler with the provided NWS client.
func NewWeatherHandler(client *nws.Client) *WeatherHandler {
	return &WeatherHandler{
		nwsClient: client,
	}
}

// GetForecast handles the get_forecast MCP tool.
func (h *WeatherHandler) GetForecast(ctx context.Context, req *mcp.CallToolRequest, input models.ForecastInput) (*mcp.CallToolResult, any, error) {
	// Validation
	if input.Latitude < -90 || input.Latitude > 90 {
		return models.ErrorResponse("Invalid latitude. Must be between -90 and 90."), nil, nil
	}
	if input.Longitude < -180 || input.Longitude > 180 {
		return models.ErrorResponse("Invalid longitude. Must be between -180 and 180."), nil, nil
	}

	// Get points data
	pointsData, err := h.nwsClient.GetPoints(ctx, input.Latitude, input.Longitude)
	if err != nil {
		return models.ErrorResponse(fmt.Sprintf("Failed to fetch location data: %v", err)), nil, nil
	}

	// Get forecast data
	forecastURL := pointsData.Properties.Forecast
	if forecastURL == "" {
		return models.ErrorResponse("Forecast data is not available for this location."), nil, nil
	}

	forecastData, err := h.nwsClient.GetForecast(ctx, forecastURL)
	if err != nil {
		return models.ErrorResponse(fmt.Sprintf("Failed to fetch detailed forecast: %v", err)), nil, nil
	}

	// Format results
	periods := forecastData.Properties.Periods
	if len(periods) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "No forecast periods available."},
			},
		}, nil, nil
	}

	var forecasts []string
	for i := 0; i < min(5, len(periods)); i++ {
		forecasts = append(forecasts, formatPeriod(periods[i]))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: strings.Join(forecasts, "\n---\n")},
		},
	}, nil, nil
}

// GetAlerts handles the get_alerts MCP tool.
func (h *WeatherHandler) GetAlerts(ctx context.Context, req *mcp.CallToolRequest, input models.AlertsInput) (*mcp.CallToolResult, any, error) {
	stateCode := strings.ToUpper(strings.TrimSpace(input.State))
	if len(stateCode) != 2 {
		return models.ErrorResponse("Invalid state code. Please provide a two-letter US state code (e.g., CA, NY)."), nil, nil
	}

	alertsData, err := h.nwsClient.GetAlerts(ctx, stateCode)
	if err != nil {
		return models.ErrorResponse(fmt.Sprintf("Failed to fetch alerts: %v", err)), nil, nil
	}

	if len(alertsData.Features) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("No active alerts for %s.", stateCode)},
			},
		}, nil, nil
	}

	var alerts []string
	for _, feature := range alertsData.Features {
		alerts = append(alerts, formatAlert(feature))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: strings.Join(alerts, "\n---\n")},
		},
	}, nil, nil
}

// formatPeriod creates a readable string for a forecast period.
func formatPeriod(p models.ForecastPeriod) string {
	return fmt.Sprintf("%s:\nTemperature: %d°%s\nWind: %s %s\nForecast: %s",
		p.Name, p.Temperature, p.TemperatureUnit, p.WindSpeed, p.WindDirection, p.DetailedForecast)
}

// formatAlert creates a readable string for a weather alert.
func formatAlert(a models.AlertFeature) string {
	p := a.Properties
	return fmt.Sprintf("Event: %s\nArea: %s\nSeverity: %s\nDescription: %s\nInstructions: %s",
		p.Event, p.AreaDesc, p.Severity, p.Description, p.Instruction)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
