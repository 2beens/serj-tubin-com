package test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/2beens/serjtubincom/internal"
	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const (
	serverPort = 9000
	serverHost = "localhost"
)

var serverEndpoint = fmt.Sprintf("http://%s:%d", serverHost, serverPort)

var (
	testGymStatsIOSAppSecret = "ios-app-secret"
	testUsername             = "testuser"
	testPassword             = "testpass"
	testPasswordHash         = "$2a$14$6Gmhg85si2etd3K9oB8nYu1cxfbrdmhkg6wI6OXsa88IF4L2r/L9i" // testpass
	testAdmin                = &auth.Admin{
		Username:     testUsername,
		PasswordHash: testPasswordHash,
	}
	testCredentials = auth.Credentials{
		Username: testUsername,
		Password: testPassword,
	}
)

type Suite struct {
	DB         *sql.DB
	dockerPool *dockertest.Pool
	server     *internal.Server
	teardown   []func()
}

func newSuite(ctx context.Context) (_ *Suite) {
	fmt.Println("setting up test suite...")

	var err error
	suite := &Suite{
		teardown: make([]func(), 0),
	}

	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	suite.dockerPool, err = dockertest.NewPool("")
	if err != nil {
		log.Fatalf("could not create new dockertest pool: %s", err)
	}
	fmt.Println("dockertest poool created")

	// uses pool to try to connect to Docker
	if err = suite.dockerPool.Client.Ping(); err != nil {
		log.Fatalf("could not ping dockertest pool: %s", err)
	}
	fmt.Println("dockertest pool ping successful")

	redisPort, err := suite.redisSetup()
	if err != nil {
		suite.cleanup()
		log.Fatalf("failed to setup redis: %s", err.Error())
	}
	fmt.Println("redis setup successful")

	pgPort, err := suite.postgresSetup(ctx)
	if err != nil {
		suite.cleanup()
		log.Fatalf("failed to setup postgres: %s", err)
	}
	fmt.Println("postgres setup successful")

	cfg := getTestConfig(redisPort, pgPort)
	suite.server, err = internal.NewServer(
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
		suite.cleanup()
		log.Fatalf("new server: %s", err)
	}
	fmt.Println("server created")

	suite.server.Serve(ctx, cfg.Host, cfg.Port)
	fmt.Println("server started")

	return suite
}

func (s *Suite) cleanup() {
	fmt.Println(" --> cleaning up test suite...")
	if s.DB != nil {
		s.DB.Close()
	}
	fmt.Println(" --> test suite db closed")
	if s.server != nil {
		s.server.GracefulShutdown()
	}
	fmt.Println(" --> test suite server shut down")
	for _, teardown := range s.teardown {
		teardown()
	}
	fmt.Println(" --> test suite cleanup done")
}

func getTestConfig(redisPort, postgresPort string) *config.Config {
	tempDir := os.TempDir()
	return &config.Config{
		Host:                     serverHost,
		Port:                     serverPort,
		QuotesCsvPath:            "../assets/quotes.csv",
		NetlogUnixSocketAddrDir:  tempDir,
		NetlogUnixSocketFileName: "netlog-test.sock",
		RedisHost:                "localhost",
		RedisPort:                redisPort,
		PostgresPort:             postgresPort,
		PostgresHost:             "localhost",
		PostgresDBName:           "serj_blogs",
	}
}

func (s *Suite) redisSetup() (string, error) {
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

	s.teardown = append(s.teardown, func() {
		redisResource.Close()
	})

	redisPort := redisResource.GetPort("6379/tcp")
	return redisPort, nil
}

func (s *Suite) postgresSetup(ctx context.Context) (string, error) {
	pgResource, err := s.dockerPool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "12",
		Env: []string{
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
		pgResource.Close()
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

	res, err := db.Exec(ctx, initSQL)
	if err != nil {
		return "", fmt.Errorf("run init script: %s", err)
	}

	log.Printf("postgres setup result: %d\n", res.RowsAffected())

	return pgPort, nil
}

const initSQL = `
CREATE TABLE public.blog
(
    id         SERIAL PRIMARY KEY,
    title      VARCHAR NOT NULL,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL,
    content    TEXT    NOT NULL,
    claps      INTEGER NOT NULL DEFAULT 0
);

ALTER TABLE public.blog OWNER TO postgres;
CREATE INDEX ix_blog_created_at ON public.blog USING btree (created_at);

-- NETLOG DB SETUP
CREATE SCHEMA netlog;
CREATE TABLE netlog.visit
(
    id        SERIAL PRIMARY KEY,
    title     VARCHAR,
    source    VARCHAR,
    device    VARCHAR,
    url       VARCHAR     NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL
);

ALTER TABLE netlog.visit OWNER TO postgres;
CREATE INDEX ix_visit_created_at ON netlog.visit USING btree (timestamp);
CREATE INDEX ix_visit_url ON netlog.visit (url);

CREATE TABLE public.note
(
    id         SERIAL PRIMARY KEY,
    title      VARCHAR,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL,
    content    TEXT NOT NULL
);

ALTER TABLE public.note OWNER TO postgres;

CREATE TABLE public.exercise
(
    id           SERIAL PRIMARY KEY,
    exercise_id  VARCHAR NOT NULL,
    muscle_group VARCHAR NOT NULL,
    kilos        INTEGER NOT NULL,
    reps         INTEGER NOT NULL,
    metadata     JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

ALTER TABLE public.exercise OWNER TO postgres;
CREATE INDEX ix_exercise_created_at ON public.exercise (created_at);

CREATE TABLE public.visitor_board_message
(
    id         SERIAL PRIMARY KEY,
    author     VARCHAR,
    message    VARCHAR NOT NULL,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

ALTER TABLE public.visitor_board_message OWNER TO postgres;
CREATE INDEX ix_visitor_board_message_created_at ON public.visitor_board_message (created_at);
`