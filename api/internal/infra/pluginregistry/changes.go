package pluginregistry

import (
	"fmt"

	"github.com/fastagi/fastbuild/api/internal/infra/projectfiles"
)

// stageInsert stages lines before the slot marker and reports whether it added them.
func stageInsert(c *projectfiles.ChangeSet, slot Slot, lines []string) (bool, error) {
	content, err := c.Read(slot.File)
	if err != nil {
		return false, err
	}
	if content == nil {
		return false, fmt.Errorf("%w: %s needs %s", errSlotMissing, slot.Name, slot.File)
	}
	updated, added, err := applyInsert(slot.File, content, slot.Name, lines)
	if err != nil || !added {
		return false, err
	}
	c.Write(slot.File, updated)
	return true, nil
}

// stageRemoval stages the reversal of an applied insert and reports whether the lines were still
// there.
func stageRemoval(c *projectfiles.ChangeSet, applied AppliedInsert) (bool, error) {
	content, err := c.Read(applied.File)
	if err != nil || content == nil {
		return false, err
	}
	updated, removed, err := removeInsert(applied.File, content, applied.Lines)
	if err != nil || !removed {
		return false, err
	}
	c.Write(applied.File, updated)
	return true, nil
}
