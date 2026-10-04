package starter

import (
	"fmt"
	"strings"

	"github.com/zgiai/luas/api/internal/starter/assembly"
)

// MarkdownCatalog renders every default and optional starter as a Markdown table from the same
// manifests the runtime assembles, so documentation cannot drift from code.
func MarkdownCatalog() string {
	var builder strings.Builder
	builder.WriteString("| Starter | Mode | Requires | Migrations | Summary |\n")
	builder.WriteString("|---|---|---|---:|---|\n")
	write := func(mode string, manifests []assembly.StarterManifest) {
		for _, manifest := range manifests {
			requires := "-"
			if dependencies := manifest.Dependencies(); len(dependencies) > 0 {
				requires = "`" + strings.Join(dependencies, "`, `") + "`"
			}
			fmt.Fprintf(&builder, "| `%s` | %s | %s | %d | %s |\n",
				manifest.Name(), mode, requires, len(manifest.MigrationNames()), assembly.SummaryOf(manifest))
		}
	}
	write("default", DefaultManifests(nil))
	write("optional", OptionalManifests(nil))
	return builder.String()
}
