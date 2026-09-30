package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// validName limits plugin names to lowercase words so a name can never become a path.
var validName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)

// ValidName reports whether name can identify a plugin binary named luas-<name>.
func ValidName(name string) bool {
	return validName.MatchString(name)
}

// Plugin represents a plugin interface
type Plugin interface {
	Name() string
	Version() string
	Description() string
}

// Discover finds installed plugins: executables named luas-<name> in PATH directories.
// Discovery runs each plugin with --version, so only PATH is searched; the current working directory
// is never included, because a checked-out repository could otherwise plant an executable.
//
// To author a plugin, ship an executable named `luas-<plugin>` on the user's PATH and respond to
// `--version` for discovery.
func Discover() []PluginInfo {
	var plugins []PluginInfo

	// Get PATH environment variable
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return plugins
	}

	// Split PATH into directories. Empty entries mean the current directory to the shell, so skip them.
	paths := strings.Split(pathEnv, string(os.PathListSeparator))

	// Track discovered plugins to avoid duplicates
	discovered := make(map[string]bool)

	for _, dir := range paths {
		if dir == "" || !filepath.IsAbs(dir) {
			continue
		}
		// Find all luas-* executables
		pattern := filepath.Join(dir, "luas-*")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, match := range matches {
			// Check if file is executable
			if !isExecutable(match) {
				continue
			}

			// Extract plugin name (remove luas- prefix)
			name := strings.TrimPrefix(filepath.Base(match), "luas-")
			if !ValidName(name) {
				continue
			}

			// Skip if already discovered
			if discovered[name] {
				continue
			}

			discovered[name] = true

			// Get plugin info
			info := PluginInfo{
				Name:   name,
				Binary: match,
			}

			// Try to get version and description
			if version := getPluginVersion(match); version != "" {
				info.Version = version
			}

			plugins = append(plugins, info)
		}
	}

	return plugins
}

// PluginInfo contains information about a discovered plugin
type PluginInfo struct {
	Name        string
	Binary      string
	Version     string
	Description string
}

// isExecutable checks if a file is executable
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// Check if it's a regular file and has execute permission
	mode := info.Mode()
	return mode.IsRegular() && (mode.Perm()&0111 != 0)
}

// getPluginVersion attempts to get plugin version
func getPluginVersion(binary string) string {
	cmd := exec.Command(binary, "--version")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// Execute runs a plugin command. The binary is resolved from PATH only.
func Execute(pluginName string, args []string) error {
	if !ValidName(pluginName) {
		return fmt.Errorf("invalid plugin name %q", pluginName)
	}
	binary, err := exec.LookPath("luas-" + pluginName)
	if err != nil {
		return err
	}
	cmd := exec.Command(binary, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// IsInstalled checks if a plugin is installed
func IsInstalled(pluginName string) bool {
	if !ValidName(pluginName) {
		return false
	}
	binary := "luas-" + pluginName
	_, err := exec.LookPath(binary)
	return err == nil
}
