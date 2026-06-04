package main

import (
	"context"
	"log"

	"skipperMCP/handlers"
	"skipperMCP/nws"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// Initialize dependencies
	nwsClient := nws.NewClient()
	weatherHandler := handlers.NewWeatherHandler(nwsClient)
	skipperHandler := handlers.NewSkipperHandler()

	// Create MCP server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "skipper-mcp",
		Version: "1.0.0",
	}, nil)

	// Register tools
	// Weather Tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_forecast",
		Description: "Essential for pre-sail checks. Retrieves detailed marine-relevant weather forecasts for a specific GPS location.",
	}, weatherHandler.GetForecast)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_alerts",
		Description: "Safety critical. Monitors active weather alerts and storm warnings for a US state/area.",
	}, weatherHandler.GetAlerts)

	// Skipper/Navigation Tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "calculate_navigation",
		Description: "Computes course bearing, distance, and estimated voyage time between two nautical coordinates.",
	}, skipperHandler.GetNavigation)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "book_harbor_slip",
		Description: "Coordinates logistics by requesting berthing/slip availability at specified marinas for overnight stays.",
	}, skipperHandler.BookHarbor)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "plan_fuel_consumption",
		Description: "Optimizes voyage planning by calculating required fuel based on distance, speed, and vessel consumption rates.",
	}, skipperHandler.FuelPlanner)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "check_system_maintenance",
		Description: "Diagnostic tool for monitoring the health of critical vessel systems like engines, electrical, and plumbing.",
	}, skipperHandler.CheckMaintenance)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_tide_data",
		Description: "Retrieves localized tide and current information. Critical for navigating narrow channels and harbor entrances.",
	}, skipperHandler.GetTideInformation)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "calculate_anchor_rode",
		Description: "Safety tool for anchoring. Computes required chain/rope length based on water depth and wind conditions.",
	}, skipperHandler.CalculateAnchorRode)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_emergency_checklist",
		Description: "CRITICAL SAFETY TOOL. Provides immediate step-by-step procedures for maritime emergencies like Fire, MOB, or Sinking.",
	}, skipperHandler.GetEmergencyChecklist)

	// Run server on stdio transport
	log.Println("Starting Skipper Companion MCP Server...")
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
