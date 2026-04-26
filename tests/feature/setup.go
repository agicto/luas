package feature

import (
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zgiai/zgo/internal/bootstrap"
	"github.com/zgiai/zgo/internal/infra/config"
	"github.com/zgiai/zgo/internal/infra/database"
	"github.com/zgiai/zgo/internal/infra/events"
	"github.com/zgiai/zgo/internal/infra/jwt"
	test_platform "github.com/zgiai/zgo/internal/infra/testing"
	"github.com/zgiai/zgo/internal/modules/apikey"
	"github.com/zgiai/zgo/internal/modules/audit"
	"github.com/zgiai/zgo/internal/modules/user"
	"github.com/zgiai/zgo/internal/starter"
	"github.com/zgiai/zgo/routes"
)

// SetupApp initializes the application for feature testing.
// Uses manual DI instead of Wire for test flexibility.
func SetupApp() *gin.Engine {
	// 1. Create Test Config
	cfg := &config.Config{}
	cfg.Server.Mode = "test"
	cfg.Database.Enabled = true
	cfg.Database.Driver = "sqlite"
	cfg.Database.Memory = true
	cfg.Database.MaxIdleConns = 1
	cfg.Database.MaxOpenConns = 1
	cfg.JWT.Secret = "testing-secret"
	cfg.JWT.Expire = time.Hour

	// 2. Initialize Database (In-Memory SQLite)
	db, err := database.NewDB(cfg)
	if err != nil {
		fmt.Printf("NewDB Error: %v\n", err)
		panic("failed to init test db: " + err.Error())
	}

	// 3. Run Migrations
	if err := bootstrap.RunMigrations(db); err != nil {
		fmt.Printf("RunMigrations Error: %v\n", err)
		panic("failed to run migrations for test db: " + err.Error())
	}

	// 4. Create Services via DI
	jwtService := jwt.NewService(cfg)
	eventBus := events.NewEventBus()

	// 5. Create Repositories
	auditRepo := audit.NewRepository(db)
	apiKeyRepo := apikey.NewRepository(db)
	userRepo := user.NewRepository(db)

	// 6. Create Services
	auditService := audit.NewService(auditRepo)
	apiKeyService := apikey.NewService(apiKeyRepo)
	userService := user.NewService(userRepo, jwtService, eventBus)

	// 7. Create Starter Registry
	auditHandler := audit.NewHandler(auditService)
	apiKeyHandler := apikey.NewHandler(apiKeyService)
	userHandler := user.NewHandler(userService, userService, userService, jwtService)
	starters, err := starter.NewDefaultRegistry(auditHandler, apiKeyHandler, userHandler)
	if err != nil {
		panic("failed to build starter registry: " + err.Error())
	}

	// 8. Build Application Routes
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	routes.Setup(r, starters)

	return r
}

// NewTestCase is a shortcut to create a test case with the setup app
func NewTestCase(t *testing.T) *test_platform.TestCase {
	engine := SetupApp()
	return test_platform.NewTestCase(t, engine)
}
