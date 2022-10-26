//go:build integration_test || all_tests

package blog

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBlogPsqlApi(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}
	t.Logf("using postres host: %s", host)

	psqlApi, err := NewBlogPsqlApi(ctx, host, "5432", "serj_blogs")
	require.NoError(t, err)
	require.NotNil(t, psqlApi)
	assert.NotNil(t, psqlApi.db)
}
