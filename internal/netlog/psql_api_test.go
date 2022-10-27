//go:build integration_test || all_tests

package netlog

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getPsqlApi(t *testing.T) (*PsqlApi, error) {
	t.Helper()

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}
	t.Logf("using postres host: %s", host)

	return NewNetlogPsqlApi(timeoutCtx, host, "5432", "serj_blogs")
}

func deleteAllVisits(ctx context.Context, psqlApi *PsqlApi) (int64, error) {
	tag, err := psqlApi.db.Exec(ctx, `DELETE FROM netlog.visit`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func TestUtil_getQueryLikeCondition(t *testing.T) {
	// no keywords
	queryLike := getQueryWhereCondition("url", "chrome", []string{})
	assert.Equal(t, "WHERE source = 'chrome'", queryLike)

	// only one keyword
	keywords := []string{"word1"}
	queryLike = getQueryWhereCondition("url", "chrome", keywords)
	assert.Equal(t, "WHERE url LIKE '%word1%' AND source = 'chrome'", queryLike)

	keywords = []string{"word1"}
	queryLike = getQueryWhereCondition("title", "safari", keywords)
	assert.Equal(t, "WHERE title LIKE '%word1%' AND source = 'safari'", queryLike)

	// more keywords
	keywords = []string{"word1", "word2", "word3"}
	queryLike = getQueryWhereCondition("url", "pc", keywords)
	assert.Equal(t, "WHERE url LIKE '%word1%' AND url LIKE '%word2%' AND url LIKE '%word3%' AND source = 'pc'", queryLike)

	keywords = []string{"word1", "word2", "word3"}
	queryLike = getQueryWhereCondition("title", "pc", keywords)
	assert.Equal(t, "WHERE title LIKE '%word1%' AND title LIKE '%word2%' AND title LIKE '%word3%' AND source = 'pc'", queryLike)
}

func TestNewNetlogPsqlApi(t *testing.T) {
	psqlApi, err := getPsqlApi(t)
	require.NoError(t, err)
	require.NotNil(t, psqlApi)
	assert.NotNil(t, psqlApi.db)
}

func TestPsqlApi_AddVisit(t *testing.T) {
	ctx := context.Background()
	psqlApi, err := getPsqlApi(t)
	require.NoError(t, err)

	count, err := psqlApi.CountAll(ctx)
	require.NoError(t, err)

	err = psqlApi.AddVisit(ctx, &Visit{})
	assert.Equal(t, "visit url or timestamp empty", err.Error())
	err = psqlApi.AddVisit(ctx, &Visit{
		Title:     "test",
		Source:    "pc",
		URL:       "",
		Timestamp: time.Now(),
	})
	assert.Equal(t, "visit url or timestamp empty", err.Error())

	now := time.Now()
	v1 := &Visit{
		Title:     gofakeit.Name(),
		Source:    "pc",
		URL:       gofakeit.URL(),
		Timestamp: now,
	}
	v2 := &Visit{
		Title:     gofakeit.Name(),
		Source:    "safari",
		URL:       gofakeit.URL(),
		Timestamp: now,
	}
	v3 := &Visit{
		Title:     gofakeit.Name(),
		Source:    "chrome",
		URL:       gofakeit.URL(),
		Timestamp: now,
	}

	require.NoError(t, psqlApi.AddVisit(ctx, v1))
	require.NoError(t, psqlApi.AddVisit(ctx, v2))
	require.NoError(t, psqlApi.AddVisit(ctx, v3))

	assert.NotEqual(t, v1.Id, v2.Id)
	assert.NotEqual(t, v1.Id, v3.Id)
	assert.NotEqual(t, v2.Id, v3.Id)
	assert.True(t, now.Equal(v1.Timestamp), "%v should be before %v", now, v1.Timestamp)
	assert.True(t, now.Equal(v2.Timestamp), "%v should be before %v", now, v2.Timestamp)
	assert.True(t, now.Equal(v2.Timestamp), "%v should be before %v", now, v3.Timestamp)

	countAfter, err := psqlApi.CountAll(ctx)
	require.NoError(t, err)

	assert.Equal(t, 3, countAfter-count)
}

func TestPsqlApi_GetAllVisits(t *testing.T) {
	ctx := context.Background()
	psqlApi, err := getPsqlApi(t)
	require.NoError(t, err)

	_, err = deleteAllVisits(ctx, psqlApi)
	require.NoError(t, err)

	allVisits, err := psqlApi.GetAllVisits(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, allVisits)

	now := time.Now()
	addedCount := 10
	for i := 1; i <= 10; i++ {
		psqlApi.AddVisit(ctx, &Visit{
			Title:     gofakeit.Name(),
			Source:    "pc",
			URL:       gofakeit.URL(),
			Timestamp: now.Add(time.Duration(i) * time.Minute),
		})
	}

	allVisits, err = psqlApi.GetAllVisits(ctx, &now)
	require.NoError(t, err)
	assert.Len(t, allVisits, addedCount)

	after5mins := now.Add(5 * time.Minute)
	after5minutesVisits, err := psqlApi.GetAllVisits(ctx, &after5mins)
	require.NoError(t, err)
	assert.Len(t, after5minutesVisits, 6)
}
