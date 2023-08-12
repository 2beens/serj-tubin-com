package test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/suite"

	"github.com/2beens/serjtubincom/internal"
	"github.com/2beens/serjtubincom/internal/config"
	"github.com/2beens/serjtubincom/internal/db"
)

const (
	serverPort = 9000
	serverHost = "127.0.0.1"
)

var serverEndpoint = fmt.Sprintf("http://%s:%d", serverHost, serverPort)

var (
	testGymStatsIOSAppSecret = "ios-app-secret"
	testUsername             = "testuser"
	testPassword             = "testpass"
	testPasswordHash         = "$2a$14$6Gmhg85si2etd3K9oB8nYu1cxfbrdmhkg6wI6OXsa88IF4L2r/L9i" // testpass
)

type IntegrationTestSuite struct {
	suite.Suite

	dbPool     *pgxpool.Pool
	dockerPool *dockertest.Pool
	server     *internal.Server
	teardown   []func()

	httpClient  *http.Client
	redisClient *redis.Client
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestIntegrationTestSuite(t *testing.T) {
	if ok, _ := strconv.ParseBool(os.Getenv("ST_INT_TESTS")); !ok {
		t.Skip("Skip running integration tests, set `ST_INT_TESTS=1` to run enable.")
		return
	}
	suite.Run(t, new(IntegrationTestSuite))
}

// runs before all tests are executed
func (s *IntegrationTestSuite) SetupSuite() {
	ctx := context.Background()
	fmt.Println("setting up test suite...")

	s.httpClient = &http.Client{
		Timeout: time.Minute,
	}

	s.teardown = make([]func(), 0)

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("test suite panic: %s", r)
			s.TearDownSuite()
			s.T().FailNow()
		}
	}()

	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	var err error
	s.dockerPool, err = dockertest.NewPool("")
	if err != nil {
		log.Fatalf("could not create new dockertest pool: %s", err)
	}
	fmt.Println("dockertest poool created")

	// uses pool to try to connect to Docker
	if err = s.dockerPool.Client.Ping(); err != nil {
		log.Fatalf("could not ping dockertest pool: %s", err)
	}
	fmt.Println("dockertest pool ping successful")

	redisPort, err := s.redisSetup()
	if err != nil {
		s.cleanup()
		log.Fatalf("failed to setup redis: %s", err.Error())
	}
	fmt.Println("redis setup successful")

	pgPort, err := s.postgresSetup(ctx)
	if err != nil {
		s.cleanup()
		log.Fatalf("failed to setup postgres: %s", err)
	}
	fmt.Println("postgres setup successful")

	cfg := getTestConfig(redisPort, pgPort)
	s.server, err = internal.NewServer(
		ctx,
		internal.NewServerParams{
			Config:                  cfg,
			OpenWeatherApiKey:       "test",
			IpInfoAPIKey:            "test",
			GymstatsIOSAppSecret:    testGymStatsIOSAppSecret,
			BrowserRequestsSecret:   "test",
			VersionInfo:             "test-version-info",
			AdminUsername:           testUsername,
			AdminPasswordHash:       testPasswordHash,
			RedisPassword:           "",
			HoneycombTracingEnabled: false,
		},
	)
	if err != nil {
		s.cleanup()
		log.Fatalf("new server: %s", err)
	}
	fmt.Println("server created")

	// initialize the database
	s.dbPool, err = db.NewDBPool(ctx, db.NewDBPoolParams{
		DBHost:         "localhost",
		DBPort:         pgPort,
		DBName:         "serj_blogs",
		TracingEnabled: false,
	})
	if err != nil {
		log.Fatalf("create new db pool: %s", err)
	}

	s.server.Serve(ctx, cfg.Host, cfg.Port)
	fmt.Println("server started")
}

// runs after all tests are executed
func (s *IntegrationTestSuite) TearDownSuite() {
	s.cleanup()
}

func (s *IntegrationTestSuite) cleanup() {
	fmt.Println(" --> test suite cleanup ...")
	if s.server != nil {
		s.server.GracefulShutdown()
	}
	fmt.Println(" --> test suite server shut down")
	for _, teardown := range s.teardown {
		teardown()
	}
	fmt.Println(" --> teardown done...")
	if s.dbPool != nil {
		s.dbPool.Close()
	}
	fmt.Println(" --> test suite cleanup done")
}

func getTestConfig(redisPort, postgresPort string) *config.Config {
	tempDir := os.TempDir()
	return &config.Config{
		Host:                        serverHost,
		Port:                        serverPort,
		QuotesCsvPath:               "../assets/quotes.csv",
		NetlogUnixSocketAddrDir:     tempDir,
		NetlogUnixSocketFileName:    "netlog-test.sock",
		RedisHost:                   "localhost",
		RedisPort:                   redisPort,
		PostgresPort:                postgresPort,
		PostgresHost:                "localhost",
		PostgresDBName:              "serj_blogs",
		LoginRateLimitAllowedPerMin: 10,
	}
}

func (s *IntegrationTestSuite) redisDataCleanup(ctx context.Context) error {
	if err := s.redisClient.FlushAll(ctx).Err(); err != nil {
		return fmt.Errorf("flush redis: %s", err)
	}
	return nil
}

func (s *IntegrationTestSuite) redisSetup() (string, error) {
	redisResource, err := s.dockerPool.RunWithOptions(&dockertest.RunOptions{
		Repository: "redis",
		Name:       "redis",
		Tag:        "6.2",
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
	})
	if err != nil {
		return "", fmt.Errorf("run redis: %s", err)
	}

	redisPort := redisResource.GetPort("6379/tcp")
	s.redisClient = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("localhost:%s", redisPort),
	})

	s.teardown = append(s.teardown, func() {
		if err := redisResource.Close(); err != nil {
			fmt.Printf("redis teardown: %s\n", err)
		}
	})

	return redisPort, nil
}

func (s *IntegrationTestSuite) postgresSetup(ctx context.Context) (string, error) {
	pgResource, err := s.dockerPool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "12",
		Env: []string{
			"TZ=Europe/Berlin",
			"POSTGRES_USER=postgres",
			"POSTGRES_DB=serj_blogs",
			"POSTGRES_HOST_AUTH_METHOD=trust",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		return "", fmt.Errorf("dockerpool run postgres: %s", err)
	}

	s.teardown = append(s.teardown, func() {
		if err := pgResource.Close(); err != nil {
			fmt.Printf("postgres teardown: %s\n", err)
		}
	})

	pgPort := pgResource.GetPort("5432/tcp")
	dsn := fmt.Sprintf(
		"postgres://postgres:admin@localhost:%s/serj_blogs?sslmode=disable",
		pgPort,
	)
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return "", fmt.Errorf("parse db config: %w", err)
	}

	db, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return "", fmt.Errorf("create connection pool: %w", err)
	}

	if err := s.dockerPool.Retry(func() error {
		return db.Ping(ctx)
	}); err != nil {
		panic(fmt.Errorf("connect to db: %s", err))
	}

	// read content of ../sql/db_schema.sql
	initSchemaSQL, err := os.ReadFile("../sql/db_schema.sql")
	if err != nil {
		panic(fmt.Errorf("read db schema sql: %s", err))
	}

	_, err = db.Exec(ctx, string(initSchemaSQL))
	if err != nil {
		return "", fmt.Errorf("run init script: %s", err)
	}

	return pgPort, nil
}
