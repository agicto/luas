package deployment

import (
	"github.com/google/wire"
	"github.com/zgiai/zgo/internal/infra/deploycontrol"
)

var ProviderSet = wire.NewSet(
	deploycontrol.NewManager,
	NewService,
	NewHandler,
)
