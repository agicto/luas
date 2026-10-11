package pluginregistry

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConflictMarker starts the local side of an unresolved merge written by update.
const ConflictMarker = "<<<<<<< local"

// Problem is one unfinished integration step and the command or edit that finishes it.
type Problem struct {
	Plugin  string `json:"plugin"`
	Problem string `json:"problem"`
	Fix     string `json:"fix"`
}

// Doctor checks that every installed plugin is complete: its files exist without unresolved
// merges, its slot edits are in place, and its contract is merged. Local edits are not problems.
func Doctor(root string, lock *Lock) []Problem {
	var problems []Problem
	contract, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ContractFile)))
	if err != nil {
		contract = nil // every merged path is then reported missing
	}
	for _, name := range sortedFiles(lock.Plugins) {
		entry := lock.Plugins[name]
		report := func(problem, fix string) {
			problems = append(problems, Problem{Plugin: name, Problem: problem, Fix: fix})
		}
		for _, file := range sortedFiles(entry.Files) {
			content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
			if err != nil {
				report(file+" is missing", "restore it with git, or run plugin:remove "+name+" --force and plugin:add "+name)
				continue
			}
			if bytes.Contains(content, []byte(ConflictMarker)) {
				report(file+" has unresolved update conflicts", "resolve the conflict markers, keeping the project's intent")
			}
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(file+IncomingSuffix))); err == nil {
				report(file+IncomingSuffix+" waits to be merged", "merge it into "+file+", then delete it")
			}
			if strings.HasPrefix(file, "contracts/fragments/") {
				report(file+" is not merged into the contract",
					"cd contracts && corepack pnpm merge-fragment ../"+file+" && corepack pnpm generate")
			}
		}
		for _, applied := range entry.Wiring {
			content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(applied.File)))
			if err != nil || findLines(strings.Split(string(content), "\n"), applied.Lines) < 0 {
				report(fmt.Sprintf("slot %s in %s lacks: %s", applied.Slot, applied.File, strings.Join(applied.Lines, " ")),
					"restore the line before the `"+SlotMarker+applied.Slot+"` marker")
			}
		}
		if entry.Contract != nil {
			for _, path := range entry.Contract.Paths {
				if !bytes.Contains(contract, []byte("\n  "+path+":\n")) {
					report(ContractFile+" lacks "+path, "run plugin:update "+name+" or restore the path from the plugin's fragment")
				}
			}
		}
	}
	return problems
}
