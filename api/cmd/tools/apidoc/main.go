// Package main provides API documentation generator for Luas.
// Usage: go run cmd/tools/apidoc/main.go
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zgiai/luas/api/cmd/tools/apidoc/parser"
)

func main() {
	// Command line flags
	outputDir := flag.String("output", "docs/api", "Output directory for generated docs")
	routesFile := flag.String("routes", "routes/v1/register.go", "Routes file to parse")
	appDir := flag.String("app", "app", "Application modules directory")
	format := flag.String("format", "markdown", "Output format: markdown or json")

	module := flag.String("module", "", "Generate docs for specific module only (e.g., user, apikey)")
	diff := flag.Bool("diff", false, "Show changes since last generation")
	flag.Parse()

	fmt.Println("🔍 Luas API Documentation Generator")
	fmt.Println("====================================")

	// Parse routes
	fmt.Printf("📂 Parsing routes from: %s\n", *routesFile)
	routes, err := parser.ParseRoutes(*routesFile)
	if err != nil {
		fmt.Printf("❌ Error parsing routes: %v\n", err)
		os.Exit(1)
	}

	// Filter by module if specified
	if *module != "" {
		routes = filterRoutesByModule(routes, *module)
		fmt.Printf("🔍 Filtered to module: %s (%d routes)\n", *module, len(routes))
	}
	fmt.Printf("✅ Found %d routes\n", len(routes))

	// Parse DTOs from app directory
	fmt.Printf("📂 Parsing DTOs from: %s\n", *appDir)
	dtos, err := parser.ParseDTOs(*appDir)
	if err != nil {
		fmt.Printf("❌ Error parsing DTOs: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Found %d DTOs\n", len(dtos))

	// Match routes with DTOs
	endpoints := parser.MatchRoutesWithDTOs(routes, dtos)

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Printf("❌ Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Check for changes if diff mode
	if *diff {
		changes := parser.DetectChanges(*outputDir, endpoints)
		if len(changes) == 0 {
			fmt.Println("✅ No changes detected")
		} else {
			fmt.Println("\n📝 Changes detected:")
			for _, change := range changes {
				fmt.Printf("   %s\n", change)
			}
			fmt.Println()
		}
	}

	// Generate documentation
	switch *format {
	case "markdown":
		var outputFile string
		if *module != "" {
			outputFile = filepath.Join(*outputDir, *module+".md")
			if err := parser.GenerateSingleModuleDoc(endpoints, outputFile, *module); err != nil {
				fmt.Printf("❌ Error generating markdown: %v\n", err)
				os.Exit(1)
			}
		} else {
			outputFile = filepath.Join(*outputDir, "api.md")
			if err := parser.GenerateMarkdown(endpoints, outputFile); err != nil {
				fmt.Printf("❌ Error generating markdown: %v\n", err)
				os.Exit(1)
			}
			// Also generate per-module docs
			if err := parser.GenerateModuleDocs(endpoints, *outputDir); err != nil {
				fmt.Printf("❌ Error generating module docs: %v\n", err)
				os.Exit(1)
			}
		}
		fmt.Printf("✅ Generated: %s\n", outputFile)

	case "json":
		outputFile := filepath.Join(*outputDir, "api.json")
		if err := parser.GenerateJSON(endpoints, outputFile); err != nil {
			fmt.Printf("❌ Error generating JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Generated: %s\n", outputFile)

	default:
		fmt.Printf("❌ Unknown format: %s\n", *format)
		os.Exit(1)
	}

	fmt.Println("\n🎉 Documentation generated successfully!")
}

// filterRoutesByModule filters routes by module name
func filterRoutesByModule(routes []parser.Route, module string) []parser.Route {
	var filtered []parser.Route
	module = strings.ToLower(module)
	for _, r := range routes {
		if strings.ToLower(r.Module) == module {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
