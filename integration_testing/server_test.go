package integration_testing

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/2beens/serjtubincom/internal"
	"github.com/2beens/serjtubincom/internal/config"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
)

func getTestConfig(redisPort string) *config.Config {
	tempDir := os.TempDir()
	return &config.Config{
		Host:                     "localhost",
		Port:                     9000,
		QuotesCsvPath:            "../assets/quotes.csv",
		NetlogUnixSocketAddrDir:  tempDir,
		NetlogUnixSocketFileName: "netlog-test.sock",
		RedisHost:                "localhost",
		RedisPort:                redisPort,
	}
}

func redisSetup(pool *dockertest.Pool) (string, func(), error) {
	redisResource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "redis",
		Name:       "redis",
		Tag:        "6.2",
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
	})
	if err != nil {
		return "", nil, fmt.Errorf("run redis: %s", err)
	}

	redisPort := redisResource.GetPort("6379/tcp")
	return redisPort, func() {
		redisResource.Close()
	}, nil
}

func serverSetup(ctx context.Context) (*internal.Server, func(), error) {
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, fmt.Errorf("could not create new dockertest pool: %s", err)
	}

	// uses pool to try to connect to Docker
	if err = pool.Client.Ping(); err != nil {
		return nil, nil, fmt.Errorf("could not ping dockertest pool: %s", err)
	}

	redisPort, redisCleanup, err := redisSetup(pool)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup redis: %s", err.Error())
	}

	cfg := getTestConfig(redisPort)
	server, err := internal.NewServer(
		ctx,
		internal.NewServerParams{
			Config:                  cfg,
			OpenWeatherApiKey:       "test",
			IpInfoAPIKey:            "test",
			GymstatsIOSAppSecret:    "test",
			BrowserRequestsSecret:   "test",
			VersionInfo:             "test-version-info",
			AdminUsername:           "adminUsername",
			AdminPasswordHash:       "adminPasswordHash",
			RedisPassword:           "",
			HoneycombTracingEnabled: false,
		},
	)
	if err != nil {
		return nil, nil, err
	}

	server.Serve(ctx, cfg.Host, cfg.Port)

	return server, func() {
		redisCleanup()
		server.GracefulShutdown()
	}, nil
}

func Test_NewServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, cleanupFunc, err := serverSetup(ctx)
	require.NoError(t, err)
	defer cleanupFunc()

	require.NotNil(t, server)
}
