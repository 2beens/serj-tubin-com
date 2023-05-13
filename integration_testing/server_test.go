package integration_testing

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/2beens/serjtubincom/internal"
	"github.com/2beens/serjtubincom/internal/config"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func getTestConfig() *config.Config {
	tempDir := os.TempDir()
	return &config.Config{
		Host:                     "localhost",
		Port:                     9000,
		QuotesCsvPath:            "../assets/quotes.csv",
		NetlogUnixSocketAddrDir:  tempDir,
		NetlogUnixSocketFileName: "netlog-test.sock",
	}
}

func redisSetup(ctx context.Context) (func(), error) {
	req := testcontainers.ContainerRequest{
		Image:        "redis:latest",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}
	redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	endpoint, err := redisC.Endpoint(ctx, "")
	if err != nil {
		return nil, err
	}
	fmt.Println("redis endpoint: ", endpoint)

	return func() {
		if redisC == nil {
			return
		}
		fmt.Println("terminating redis container...")
		if err := redisC.Terminate(ctx); err != nil {
			fmt.Printf("failed to terminate redis container: %s", err.Error())
		}
	}, nil
}

func serverSetup(ctx context.Context) (*internal.Server, func(), error) {
	redisCleanup, err := redisSetup(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup redis: %s", err.Error())
	}

	cfg := getTestConfig()
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
