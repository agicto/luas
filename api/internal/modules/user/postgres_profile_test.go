package user

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/zgiai/luas/api/internal/domain"
)

const postgresProfileDSNEnv = "LUAS_TEST_POSTGRES_DSN"

type statementCounterContextKey struct{}

type statementCountingLogger struct{}

func (statementCountingLogger) LogMode(logger.LogLevel) logger.Interface {
	return statementCountingLogger{}
}

func (statementCountingLogger) Info(context.Context, string, ...interface{})  {}
func (statementCountingLogger) Warn(context.Context, string, ...interface{})  {}
func (statementCountingLogger) Error(context.Context, string, ...interface{}) {}

func (statementCountingLogger) Trace(
	ctx context.Context,
	_ time.Time,
	_ func() (string, int64),
	_ error,
) {
	if counter, ok := ctx.Value(statementCounterContextKey{}).(*atomic.Int64); ok {
		counter.Add(1)
	}
}

type postgresUserProfile struct {
	db         *gorm.DB
	repository *repository
	sequence   atomic.Uint64
}

func newPostgresUserProfile(tb testing.TB) *postgresUserProfile {
	tb.Helper()
	rawDSN := os.Getenv(postgresProfileDSNEnv)
	if rawDSN == "" {
		tb.Skip(postgresProfileDSNEnv + " is not set")
	}
	parsed, err := url.Parse(rawDSN)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		tb.Fatalf("%s must be a PostgreSQL connection URI", postgresProfileDSNEnv)
	}

	admin := openPostgresProfileDB(tb, parsed.String())
	schema := fmt.Sprintf("luas_user_profile_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		closeProfileDB(tb, admin)
		tb.Fatalf("create isolated profile schema: %v", err)
	}
	var db *gorm.DB
	tb.Cleanup(func() {
		closeProfileDB(tb, db)
		if dropErr := admin.Exec("DROP SCHEMA " + schema + " CASCADE").Error; dropErr != nil {
			tb.Errorf("drop isolated profile schema: %v", dropErr)
		}
		closeProfileDB(tb, admin)
	})

	scoped := *parsed
	parameters := scoped.Query()
	parameters.Set("application_name", "luas-user-profile")
	parameters.Set("search_path", schema)
	scoped.RawQuery = parameters.Encode()
	db = openPostgresProfileDB(tb, scoped.String())

	if err := db.AutoMigrate(&UserPO{}); err != nil {
		tb.Fatalf("migrate isolated user profile schema: %v", err)
	}
	seed := make([]UserPO, 128)
	for index := range seed {
		seed[index] = UserPO{
			Username: fmt.Sprintf("profile-seed-%03d", index),
			Email:    fmt.Sprintf("profile-seed-%03d@example.com", index),
			Password: "profile-password-hash",
			Nickname: "Profile Seed",
			Status:   1,
		}
	}
	if err := db.Session(&gorm.Session{SkipDefaultTransaction: true}).CreateInBatches(seed, 64).Error; err != nil {
		tb.Fatalf("seed isolated user profile schema: %v", err)
	}

	return &postgresUserProfile{db: db, repository: NewRepository(db)}
}

func openPostgresProfileDB(tb testing.TB, dsn string) *gorm.DB {
	tb.Helper()
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger:               statementCountingLogger{},
		DisableAutomaticPing: true,
	})
	if err != nil {
		tb.Fatalf("open PostgreSQL profile database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		tb.Fatalf("resolve PostgreSQL profile pool: %v", err)
	}
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(4)
	sqlDB.SetConnMaxIdleTime(time.Minute)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		tb.Fatalf("ping PostgreSQL profile database: %v", err)
	}
	return db
}

func closeProfileDB(tb testing.TB, db *gorm.DB) {
	tb.Helper()
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		tb.Errorf("resolve PostgreSQL profile pool during cleanup: %v", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		tb.Errorf("close PostgreSQL profile pool: %v", err)
	}
}

func (p *postgresUserProfile) nextUser() *domain.User {
	sequence := p.sequence.Add(1)
	return &domain.User{
		Username: fmt.Sprintf("profile-write-%d", sequence),
		Email:    fmt.Sprintf("profile-write-%d@example.com", sequence),
		Password: "profile-password-hash",
		Nickname: "Profile Write",
		Status:   1,
	}
}

func TestPostgresUserRepositoryQueryShape(t *testing.T) {
	profile := newPostgresUserProfile(t)

	listCtx, listStatements := contextWithStatementCounter(context.Background())
	users, total, err := profile.repository.FindAll(listCtx, 1, 25)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 25 || total != 128 {
		t.Fatalf("FindAll returned %d users and total %d", len(users), total)
	}
	for _, listedUser := range users {
		if listedUser.Password != "" {
			t.Fatal("FindAll loaded password hashes into the list result")
		}
	}
	if got := listStatements.Load(); got != 1 {
		t.Fatalf("FindAll application SQL statements = %d, want 1", got)
	}

	createCtx, createStatements := contextWithStatementCounter(context.Background())
	created := profile.nextUser()
	if err := profile.repository.Create(createCtx, created); err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 {
		t.Fatal("Create did not return the generated ID")
	}
	if got := createStatements.Load(); got != 1 {
		t.Fatalf("Create application SQL statements = %d, want 1", got)
	}
}

func TestPostgresUserRepositoryPerformanceProfile(t *testing.T) {
	profile := newPostgresUserProfile(t)
	const samples = 200

	if _, _, err := profile.repository.FindAll(context.Background(), 1, 25); err != nil {
		t.Fatal(err)
	}
	listDurations := sampleDurations(t, samples, func() error {
		_, _, err := profile.repository.FindAll(context.Background(), 1, 25)
		return err
	})
	createDurations := sampleDurations(t, samples, func() error {
		return profile.repository.Create(context.Background(), profile.nextUser())
	})

	listAllocs := testing.AllocsPerRun(25, func() {
		if _, _, err := profile.repository.FindAll(context.Background(), 1, 25); err != nil {
			t.Fatal(err)
		}
	})
	createAllocs := testing.AllocsPerRun(25, func() {
		if err := profile.repository.Create(context.Background(), profile.nextUser()); err != nil {
			t.Fatal(err)
		}
	})

	t.Logf(
		"PostgreSQL user repository profile (%d samples): FindAll p95=%s allocs/op=%.0f statements=1; Create p95=%s allocs/op=%.0f statements=1",
		samples,
		percentile95(listDurations),
		listAllocs,
		percentile95(createDurations),
		createAllocs,
	)
}

func BenchmarkPostgresUserRepositoryList(b *testing.B) {
	profile := newPostgresUserProfile(b)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, _, err := profile.repository.FindAll(context.Background(), 1, 25); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPostgresUserRepositoryCreate(b *testing.B) {
	profile := newPostgresUserProfile(b)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := profile.repository.Create(context.Background(), profile.nextUser()); err != nil {
			b.Fatal(err)
		}
	}
}

func contextWithStatementCounter(ctx context.Context) (context.Context, *atomic.Int64) {
	counter := &atomic.Int64{}
	return context.WithValue(ctx, statementCounterContextKey{}, counter), counter
}

func sampleDurations(t *testing.T, count int, operation func() error) []time.Duration {
	t.Helper()
	durations := make([]time.Duration, count)
	for index := range durations {
		started := time.Now()
		if err := operation(); err != nil {
			t.Fatal(err)
		}
		durations[index] = time.Since(started)
	}
	return durations
}

func percentile95(durations []time.Duration) time.Duration {
	sorted := append([]time.Duration(nil), durations...)
	sort.Slice(sorted, func(left, right int) bool { return sorted[left] < sorted[right] })
	index := (len(sorted)*95 + 99) / 100
	if index == 0 {
		return 0
	}
	return sorted[index-1]
}
