package deploycontrol

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWatchCancelIsIdempotentForFinishedDeployment(t *testing.T) {
	storageRoot := t.TempDir()
	t.Setenv("DEPLOY_STORAGE_ROOT", storageRoot)
	t.Setenv("DEPLOY_TARGETS_PATH", storageRoot+"/targets.json")

	manager := NewManager()
	now := time.Now().UTC()
	deployment := &Deployment{
		ID:         "finished-deployment",
		Target:     "test",
		TargetName: "Test",
		Status:     StatusSucceeded,
		CreatedAt:  now,
		FinishedAt: &now,
	}

	require.NoError(t, manager.saveDeployment(deployment))

	_, cancel, err := manager.Watch(deployment.ID)
	require.NoError(t, err)

	require.NotPanics(t, cancel)
	require.NotPanics(t, cancel)
}
