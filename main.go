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

	// Create MCP server with dynamic tool discovery capability
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "skipper-mcp",
		Version: "1.0.0",
	}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{
			Tools: &mcp.ToolCapabilities{
				ListChanged: true,
			},
		},
	})

	dynamicHandler := handlers.NewDynamicToolHandler(server)

	// 1. Register Meta-Tools (these are always available)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_catalog",
		Description: "Search the maritime tool catalog to find relevant tools for weather, navigation, safety, or harbor logistics.",
	}, dynamicHandler.SearchCatalog)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "load_tool",
		Description: "Dynamically load a specific tool from the catalog into your active toolset.",
	}, dynamicHandler.LoadTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "unload_tool",
		Description: "Remove a tool from your active toolset to reduce context bloat when a task is finished.",
	}, dynamicHandler.UnloadTool)

	// 2. Register Maritime Tools into the Catalog (NOT on the server yet)
	
	// Weather Tools
	dynamicHandler.RegisterCatalogTool(&mcp.Tool{
		Name:        "get_forecast",
		Description: "Essential for pre-sail checks. Retrieves detailed marine-relevant weather forecasts for a specific GPS location.",
	}, func(s *mcp.Server) {
		mcp.AddTool(s, &mcp.Tool{
			Name:        "get_forecast",
			Description: "Essential for pre-sail checks. Retrieves detailed marine-relevant weather forecasts for a specific GPS location.",
		}, weatherHandler.GetForecast)
	})

	dynamicHandler.RegisterCatalogTool(&mcp.Tool{
		Name:        "get_alerts",
		Description: "Safety critical. Monitors active weather alerts and storm warnings for a US state/area.",
	}, func(s *mcp.Server) {
		mcp.AddTool(s, &mcp.Tool{
			Name:        "get_alerts",
			Description: "Safety critical. Monitors active weather alerts and storm warnings for a US state/area.",
		}, weatherHandler.GetAlerts)
	})

	// Skipper/Navigation Tools
	dynamicHandler.RegisterCatalogTool(&mcp.Tool{
		Name:        "calculate_navigation",
		Description: "Computes course bearing, distance, and estimated voyage time between two nautical coordinates.",
	}, func(s *mcp.Server) {
		mcp.AddTool(s, &mcp.Tool{
			Name:        "calculate_navigation",
			Description: "Computes course bearing, distance, and estimated voyage time between two nautical coordinates.",
		}, skipperHandler.GetNavigation)
	})

	dynamicHandler.RegisterCatalogTool(&mcp.Tool{
		Name:        "book_harbor_slip",
		Description: "Coordinates logistics by requesting berthing/slip availability at specified marinas for overnight stays.",
	}, func(s *mcp.Server) {
		mcp.AddTool(s, &mcp.Tool{
			Name:        "book_harbor_slip",
			Description: "Coordinates logistics by requesting berthing/slip availability at specified marinas for overnight stays.",
		}, skipperHandler.BookHarbor)
	})

	dynamicHandler.RegisterCatalogTool(&mcp.Tool{
		Name:        "plan_fuel_consumption",
		Description: "Optimizes voyage planning by calculating required fuel based on distance, speed, and vessel consumption rates.",
	}, func(s *mcp.Server) {
		mcp.AddTool(s, &mcp.Tool{
			Name:        "plan_fuel_consumption",
			Description: "Optimizes voyage planning by calculating required fuel based on distance, speed, and vessel consumption rates.",
		}, skipperHandler.FuelPlanner)
	})

	dynamicHandler.RegisterCatalogTool(&mcp.Tool{
		Name:        "check_system_maintenance",
		Description: "Diagnostic tool for monitoring the health of critical vessel systems like engines, electrical, and plumbing.",
	}, func(s *mcp.Server) {
		mcp.AddTool(s, &mcp.Tool{
			Name:        "check_system_maintenance",
			Description: "Diagnostic tool for monitoring the health of critical vessel systems like engines, electrical, and plumbing.",
		}, skipperHandler.CheckMaintenance)
	})

	dynamicHandler.RegisterCatalogTool(&mcp.Tool{
		Name:        "get_tide_data",
		Description: "Retrieves localized tide and current information. Critical for navigating narrow channels and harbor entrances.",
	}, func(s *mcp.Server) {
		mcp.AddTool(s, &mcp.Tool{
			Name:        "get_tide_data",
			Description: "Retrieves localized tide and current information. Critical for navigating narrow channels and harbor entrances.",
		}, skipperHandler.GetTideInformation)
	})

	dynamicHandler.RegisterCatalogTool(&mcp.Tool{
		Name:        "calculate_anchor_rode",
		Description: "Safety tool for anchoring. Computes required chain/rope length based on water depth and wind conditions.",
	}, func(s *mcp.Server) {
		mcp.AddTool(s, &mcp.Tool{
			Name:        "calculate_anchor_rode",
			Description: "Safety tool for anchoring. Computes required chain/rope length based on water depth and wind conditions.",
		}, skipperHandler.CalculateAnchorRode)
	})

	dynamicHandler.RegisterCatalogTool(&mcp.Tool{
		Name:        "get_emergency_checklist",
		Description: "CRITICAL SAFETY TOOL. Provides immediate step-by-step procedures for maritime emergencies like Fire, MOB, or Sinking.",
	}, func(s *mcp.Server) {
		mcp.AddTool(s, &mcp.Tool{
			Name:        "get_emergency_checklist",
			Description: "CRITICAL SAFETY TOOL. Provides immediate step-by-step procedures for maritime emergencies like Fire, MOB, or Sinking.",
		}, skipperHandler.GetEmergencyChecklist)
	})

	// Register Resources
	server.AddResource(&mcp.Resource{
		Name:        "Vessel Logs",
		URI:         "vessel://status/logs",
		Description: "Recent maintenance and status logs for the vessel.",
		MIMEType:    "text/plain",
	}, skipperHandler.GetResource)

	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "Marina Regulations",
		URITemplate: "marina://{marina_id}/regulations",
		Description: "Local harbor regulations and docking procedures for a specific marina.",
	}, skipperHandler.GetResource)

	// Register Prompts
	server.AddPrompt(&mcp.Prompt{
		Name:        "arrival-briefing",
		Description: "A guided template for preparing a harbor arrival briefing.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "marina_name",
				Description: "The name of the destination marina.",
				Required:    true,
			},
			{
				Name:        "vessel_name",
				Description: "The name of the vessel.",
				Required:    false,
			},
		},
	}, skipperHandler.GetPrompt)

	// Run server on stdio transport
	log.Println("Starting Skipper Companion MCP Server...")
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
