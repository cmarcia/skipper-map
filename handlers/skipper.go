package handlers

import (
	"context"
	"fmt"
	"math"
	"strings"

	"skipperMCP/models"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SkipperHandler manages boat-specific tools.
type SkipperHandler struct{}

func NewSkipperHandler() *SkipperHandler {
	return &SkipperHandler{}
}

// GetNavigation provides basic routing instructions.
func (h *SkipperHandler) GetNavigation(ctx context.Context, req *mcp.CallToolRequest, input models.NavigationInput) (*mcp.CallToolResult, any, error) {
	// Simple Great Circle distance approximation
	dist := haversine(input.StartLat, input.StartLon, input.EndLat, input.EndLon)
	time := dist / input.BoatSpeed

	result := fmt.Sprintf("Course Analysis:\nDistance: %.2f nautical miles\nEstimated Time at En-route: %.1f hours\nInitial Bearing: %.0f°",
		dist, time, calculateBearing(input.StartLat, input.StartLon, input.EndLat, input.EndLon))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, nil, nil
}

// BookHarbor simulates a marina reservation.
func (h *SkipperHandler) BookHarbor(ctx context.Context, req *mcp.CallToolRequest, input models.HarborBookingInput) (*mcp.CallToolResult, any, error) {
	// In a real app, this would call a booking API
	confirmation := fmt.Sprintf("Harbor Reservation Pending:\nMarina: %s\nArrival: %s\nDuration: %d nights\nSlip Type: %.0fft Berthing\nStatus: Request Sent to Dockmaster",
		input.MarinaName, input.ArrivalDate, input.Nights, input.VesselLength)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: confirmation},
		},
	}, nil, nil
}

// FuelPlanner calculates consumption.
func (h *SkipperHandler) FuelPlanner(ctx context.Context, req *mcp.CallToolRequest, input models.FuelPlannerInput) (*mcp.CallToolResult, any, error) {
	if input.AverageSpeed <= 0 {
		return models.ErrorResponse("Average speed must be greater than 0."), nil, nil
	}
	hours := input.DistanceNm / input.AverageSpeed
	totalFuel := hours * input.FuelConsumption

	result := fmt.Sprintf("Fuel Plan:\nEstimated Voyage Time: %.1f hours\nTotal Fuel Required: %.1f gallons\nRecommended Reserve (20%%): %.1f gallons",
		hours, totalFuel, totalFuel*1.2)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, nil, nil
}

// CheckMaintenance provides status of boat systems.
func (h *SkipperHandler) CheckMaintenance(ctx context.Context, req *mcp.CallToolRequest, input models.MaintenanceInput) (*mcp.CallToolResult, any, error) {
	system := strings.ToLower(input.System)
	var status string

	switch system {
	case "engine":
		status = "Oil pressure normal. Next service in 45 hours. Coolant levels OK."
	case "electrical":
		status = "Battery bank at 12.8V (95%). Solar input: 4.2A. All circuits active."
	case "plumbing":
		status = "Fresh water: 60%. Bilge pumps: Auto/Standby. Grey water: Empty."
	default:
		status = fmt.Sprintf("System '%s' check complete. No anomalies detected.", input.System)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: status},
		},
	}, nil, nil
}

// GetTideInformation provides simulated tide/current data.
func (h *SkipperHandler) GetTideInformation(ctx context.Context, req *mcp.CallToolRequest, input models.TideInput) (*mcp.CallToolResult, any, error) {
	result := fmt.Sprintf("Tide Data for Station %s:\nHigh Tide: 14:22 (+4.2ft)\nLow Tide: 20:45 (-0.8ft)\nCurrent: 1.2kts Ebb\nStatus: Safe for harbor entrance", input.StationID)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, nil, nil
}

// CalculateAnchorRode provides recommended anchor chain/rope length.
func (h *SkipperHandler) CalculateAnchorRode(ctx context.Context, req *mcp.CallToolRequest, input models.AnchorRodeInput) (*mcp.CallToolResult, any, error) {
	scope := 5.0 // Default scope
	if input.WindSpeed > 25 {
		scope = 7.0
	}
	if !input.IsAllChain {
		scope += 2.0
	}

	totalRode := input.DepthFeet * scope
	result := fmt.Sprintf("Anchoring Analysis:\nDepth: %.1fft\nRecommended Scope: %.1f:1\nTotal Rode to Deploy: %.1fft\nNotes: Ensure anchor is set before engine shutdown.",
		input.DepthFeet, scope, totalRode)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, nil, nil
}

// GetEmergencyChecklist provides immediate action steps for maritime emergencies.
func (h *SkipperHandler) GetEmergencyChecklist(ctx context.Context, req *mcp.CallToolRequest, input models.EmergencyInput) (*mcp.CallToolResult, any, error) {
	situation := strings.ToLower(input.Situation)
	var checklist string

	switch {
	case strings.Contains(situation, "man overboard") || strings.Contains(situation, "mob"):
		checklist = "1. Shout 'Man Overboard!'\n2. Throw flotation device immediately.\n3. Keep eyes on person at all times.\n4. Press MOB button on GPS.\n5. Execute Williamson Turn or Anderson Turn."
	case strings.Contains(situation, "fire"):
		checklist = "1. Sound alarm.\n2. Cut engines and fuel supply.\n3. Position boat so wind blows fire AWAY from vessel.\n4. Use appropriate extinguisher at base of flames.\n5. Prepare life raft."
	case strings.Contains(situation, "taking water") || strings.Contains(situation, "sink"):
		checklist = "1. Start all bilge pumps.\n2. Locate source of ingress.\n3. Head for shallow water/beach if possible.\n4. Issue Pan-Pan or Mayday on VHF Ch 16.\n5. Don life jackets."
	default:
		checklist = "1. Maintain calm.\n2. Ensure all souls are wearing life jackets.\n3. Assess situation and stop engines if safe.\n4. Radio for assistance on VHF Ch 16 if required."
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("EMERGENCY CHECKLIST: %s\n\n%s", strings.ToUpper(input.Situation), checklist)},
		},
	}, nil, nil
}

// Helper functions for navigation math
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 3440.065 // Earth radius in nautical miles
	phi1 := lat1 * math.Pi / 180
	phi2 := lat2 * math.Pi / 180
	deltaPhi := (lat2 - lat1) * math.Pi / 180
	deltaLambda := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaPhi/2)*math.Sin(deltaPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*
			math.Sin(deltaLambda/2)*math.Sin(deltaLambda/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func calculateBearing(lat1, lon1, lat2, lon2 float64) float64 {
	y := math.Sin((lon2-lon1)*math.Pi/180) * math.Cos(lat2*math.Pi/180)
	x := math.Cos(lat1*math.Pi/180)*math.Sin(lat2*math.Pi/180) -
		math.Sin(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*math.Cos((lon2-lon1)*math.Pi/180)
	theta := math.Atan2(y, x)
	bearing := math.Mod((theta*180/math.Pi)+360, 360)
	return bearing
}
