package bootstrap

import (
	"fmt"

	"github.com/fatih/color"
)

// printBanner prints the FastBuild startup banner.
func printBanner(version string) {
	bannerColor := color.New(color.FgCyan, color.Bold)
	secondaryColor := color.New(color.FgHiBlue)

	bannerColor.Println("FastBuild")
	secondaryColor.Printf("Modular Go API Scaffold %s\n", version)
	fmt.Println()
}
