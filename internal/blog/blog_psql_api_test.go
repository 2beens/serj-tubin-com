//go:build integration_test || all_tests

package blog

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBlogPsqlApi(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	host, port, dbName := "localhost", "5432", "serj_blogs"

	psqlApi, err := NewBlogPsqlApi(ctx, host, port, dbName)
	require.NoError(t, err)
	require.NotNil(t, psqlApi)
	assert.NotNil(t, psqlApi.db)
}
