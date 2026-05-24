package support

import (
	"fmt"

	"github.com/fatih/color"
)

// PrintBanner prints the ZGO startup banner to console.
func PrintBanner(version string) {
	bannerColor := color.New(color.FgCyan, color.Bold)
	secondaryColor := color.New(color.FgHiBlue)

	bannerColor.Println("ZGO")
	secondaryColor.Printf("Modular Go API Scaffold %s\n", version)
	fmt.Println()
}
