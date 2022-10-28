//go:build integration_test || all_tests

package netlog

import (
	"context"
	"fmt"
	"os"
	"strings"
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

func TestPsqlApi_GetVisits_and_Count(t *testing.T) {
	ctx := context.Background()
	psqlApi, err := getPsqlApi(t)
	require.NoError(t, err)

	_, err = deleteAllVisits(ctx, psqlApi)
	require.NoError(t, err)

	now := time.Now()
	v1 := &Visit{
		Title:     "title one",
		Source:    "pc",
		URL:       "https://www.one.com/",
		Timestamp: now,
	}
	v2 := &Visit{
		Title:     "title two",
		Source:    "safari",
		URL:       "https://www.two.com/",
		Timestamp: now.Add(-1 * time.Minute),
	}
	v3 := &Visit{
		Title:     "title three",
		Source:    "chrome",
		URL:       "https://www.three.com/",
		Timestamp: now.Add(-2 * time.Minute),
	}
	v4 := &Visit{
		Title:     "title four",
		Source:    "chrome",
		URL:       "https://www.four.com/",
		Timestamp: now.Add(-3 * time.Minute),
	}
	v4b := &Visit{
		Title:     "title four b",
		Source:    "chrome",
		URL:       "https://www.four.com/beta",
		Timestamp: now.Add(-4 * time.Minute),
	}

	require.NoError(t, psqlApi.AddVisit(ctx, v1))
	require.NoError(t, psqlApi.AddVisit(ctx, v2))
	require.NoError(t, psqlApi.AddVisit(ctx, v3))
	require.NoError(t, psqlApi.AddVisit(ctx, v4))
	require.NoError(t, psqlApi.AddVisit(ctx, v4b))

	allVisits, err := psqlApi.CountAll(ctx)
	require.NoError(t, err)
	require.Equal(t, allVisits, 5)

	testCases := []struct {
		keywords            []string
		field               string
		source              string
		limit               int
		expectedVisitsCount int
	}{
		{keywords: nil, field: "url", source: "all", limit: 10, expectedVisitsCount: 5},
		{keywords: nil, field: "url", source: "all", limit: 2, expectedVisitsCount: 2},
		{keywords: nil, field: "title", source: "all", limit: 10, expectedVisitsCount: 5},
		{keywords: nil, field: "title", source: "all", limit: 2, expectedVisitsCount: 2},

		{keywords: []string{"title"}, field: "title", source: "chrome", limit: 10, expectedVisitsCount: 3},
		{keywords: []string{"title"}, field: "title", source: "safari", limit: 10, expectedVisitsCount: 1},
		{keywords: []string{"title"}, field: "title", source: "pc", limit: 10, expectedVisitsCount: 1},
		{keywords: []string{"title"}, field: "title", source: "unknown", limit: 10, expectedVisitsCount: 0},
		{keywords: []string{"title"}, field: "url", source: "unknown", limit: 10, expectedVisitsCount: 0},
		{keywords: []string{"some other title"}, field: "title", source: "chrome", limit: 10, expectedVisitsCount: 0},

		{keywords: []string{"three"}, field: "url", source: "chrome", limit: 10, expectedVisitsCount: 1},
		{keywords: []string{"four"}, field: "url", source: "chrome", limit: 10, expectedVisitsCount: 2},
		{keywords: []string{"www"}, field: "url", source: "chrome", limit: 10, expectedVisitsCount: 3},
		{keywords: []string{"four", "www"}, field: "url", source: "chrome", limit: 10, expectedVisitsCount: 2},
		{keywords: []string{"four", "www", "beta"}, field: "url", source: "chrome", limit: 10, expectedVisitsCount: 1},
		{keywords: []string{"one"}, field: "url", source: "pc", limit: 10, expectedVisitsCount: 1},
		{keywords: []string{"four"}, field: "url", source: "pc", limit: 10, expectedVisitsCount: 0},
	}

	for _, tc := range testCases {
		gottenVisits, err := psqlApi.GetVisits(ctx, tc.keywords, tc.field, tc.source, tc.limit)
		require.NoError(t, err)
		assert.Len(t, gottenVisits, tc.expectedVisitsCount, fmt.Sprintf("retrieved visits invalid for test case: %v", tc))

		gottenCount, err := psqlApi.Count(ctx, tc.keywords, tc.field, tc.source)
		require.NoError(t, err)
		if tc.source != "all" {
			assert.Equal(t, tc.expectedVisitsCount, gottenCount, fmt.Sprintf("count invalid for test case: %+v", tc))
		} else {
			assert.Equal(t, 5, gottenCount, fmt.Sprintf("count invalid for test case: %+v", tc))
		}
	}

	visits, err := psqlApi.GetVisits(ctx, []string{"four"}, "url", "chrome", 10)
	require.NoError(t, err)
	require.Len(t, visits, 2)
	assert.True(t, strings.HasPrefix(visits[0].Title, "title four"))
	assert.True(t, strings.HasPrefix(visits[1].Title, "title four"))
}
