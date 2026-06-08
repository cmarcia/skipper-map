package models

import "github.com/modelcontextprotocol/go-sdk/mcp"

// PointsResponse represents the response from the /points endpoint.
type PointsResponse struct {
	Properties struct {
		Forecast string `json:"forecast"`
	} `json:"properties"`
}

// ForecastResponse represents the response from the forecast endpoint.
type ForecastResponse struct {
	Properties struct {
		Periods []ForecastPeriod `json:"periods"`
	} `json:"properties"`
}

// ForecastPeriod represents a single forecast period.
type ForecastPeriod struct {
	Name             string `json:"name"`
	Temperature      int    `json:"temperature"`
	TemperatureUnit  string `json:"temperatureUnit"`
	WindSpeed        string `json:"windSpeed"`
	WindDirection    string `json:"windDirection"`
	DetailedForecast string `json:"detailedForecast"`
}

// AlertsResponse represents the response from the /alerts endpoint.
type AlertsResponse struct {
	Features []AlertFeature `json:"features"`
}

// AlertFeature represents a single alert feature.
type AlertFeature struct {
	Properties AlertProperties `json:"properties"`
}

// AlertProperties represents the properties of an alert.
type AlertProperties struct {
	Event       string `json:"event"`
	AreaDesc    string `json:"areaDesc"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Instruction string `json:"instruction"`
}

// ForecastInput represents the input for the get_forecast tool.
type ForecastInput struct {
	Latitude  float64 `json:"latitude" jsonschema:"Latitude of the location"`
	Longitude float64 `json:"longitude" jsonschema:"Longitude of the location"`
}

// AlertsInput represents the input for the get_alerts tool.
type AlertsInput struct {
	State string `json:"state" jsonschema:"Two-letter US state code (e.g. CA, NY)"`
}

// NavigationInput represents the input for course calculation.
type NavigationInput struct {
	StartLat  float64 `json:"start_lat" jsonschema:"Starting latitude"`
	StartLon  float64 `json:"start_lon" jsonschema:"Starting longitude"`
	EndLat    float64 `json:"end_lat" jsonschema:"Destination latitude"`
	EndLon    float64 `json:"end_lon" jsonschema:"Destination longitude"`
	BoatSpeed float64 `json:"boat_speed" jsonschema:"Speed in knots"`
}

// HarborBookingInput represents the input for booking a stay.
type HarborBookingInput struct {
	MarinaName   string `json:"marina_name" jsonschema:"Name of the marina"`
	ArrivalDate  string `json:"arrival_date" jsonschema:"Arrival date (YYYY-MM-DD)"`
	Nights       int    `json:"nights" jsonschema:"Number of nights to stay"`
	VesselLength float64 `json:"vessel_length" jsonschema:"Length of the vessel in feet"`
}

// FuelPlannerInput represents fuel calculation.
type FuelPlannerInput struct {
	DistanceNm      float64 `json:"distance_nm" jsonschema:"Distance in nautical miles"`
	FuelConsumption float64 `json:"fuel_consumption" jsonschema:"Fuel consumption in gallons per hour"`
	AverageSpeed    float64 `json:"average_speed" jsonschema:"Average speed in knots"`
}

// MaintenanceInput represents system status check.
type MaintenanceInput struct {
	System string `json:"system" jsonschema:"System to check (e.g., engine, electrical, plumbing)"`
}

// TideInput represents tide data request.
type TideInput struct {
	StationID string `json:"station_id" jsonschema:"NOAA Station ID for tide data"`
}

// AnchorRodeInput represents anchor calculation.
type AnchorRodeInput struct {
	DepthFeet   float64 `json:"depth_feet" jsonschema:"Water depth in feet"`
	WindSpeed   float64 `json:"wind_speed" jsonschema:"Expected wind speed in knots"`
	IsAllChain  bool    `json:"is_all_chain" jsonschema:"Whether using all-chain rode (true) or rope/chain combo (false)"`
}

// EmergencyInput represents emergency procedure request.
type EmergencyInput struct {
	Situation string `json:"situation" jsonschema:"Emergency type (e.g., Man Overboard, Fire, Engine Failure, Taking Water)"`
}

// SearchCatalogInput represents the input for searching tools.
type SearchCatalogInput struct {
	Query string `json:"query" jsonschema:"The search term or intent to find relevant tools (e.g. 'weather', 'navigation')"`
}

// LoadToolInput represents the input for loading a specific tool.
type LoadToolInput struct {
	ToolName string `json:"tool_name" jsonschema:"The exact name of the tool to load from the catalog"`
}

// ErrorResponse is a helper to return consistent error results to MCP.
func ErrorResponse(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: message},
		},
		IsError: true,
	}
}
