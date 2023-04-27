//go:build integration_test || all_tests

package netlog

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/db"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRepoSetup(t *testing.T) (*Repo, func()) {
	t.Helper()

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}
	t.Logf("using postres host: %s", host)

	dbPool, err := db.NewDBPool(timeoutCtx, db.NewDBPoolParams{
		DBHost:         host,
		DBPort:         "5432",
		DBName:         "serj_blogs",
		TracingEnabled: false,
	})
	require.NoError(t, err)

	return NewRepo(dbPool), func() {
		dbPool.Close()
	}
}

func deleteAllVisits(ctx context.Context, repo *Repo) (int64, error) {
	tag, err := repo.db.Exec(ctx, `DELETE FROM netlog.visit`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func cleanupAndAddTestVisits(ctx context.Context, t *testing.T, repo *Repo) []*Visit {
	t.Helper()

	_, err := deleteAllVisits(ctx, repo)
	require.NoError(t, err)

	now := time.Now()
	v1 := &Visit{
		Title:     "title one",
		Source:    "pc",
		Device:    "home-pc",
		URL:       "https://www.one.com/",
		Timestamp: now,
	}
	v2 := &Visit{
		Title:     "title two",
		Source:    "safari",
		Device:    "mb-work",
		URL:       "https://www.two.com/",
		Timestamp: now.Add(-1 * time.Minute),
	}
	v3 := &Visit{
		Title:     "title three",
		Source:    "chrome",
		Device:    "mb-work",
		URL:       "https://www.three.com/",
		Timestamp: now.Add(-2 * time.Minute),
	}
	v4 := &Visit{
		Title:     "title four",
		Source:    "chrome",
		Device:    "mb-serj",
		URL:       "https://www.four.com/",
		Timestamp: now.Add(-3 * time.Minute),
	}
	v4b := &Visit{
		Title:     "title four b",
		Source:    "chrome",
		Device:    "mb-serj",
		URL:       "https://www.four.com/beta",
		Timestamp: now.Add(-4 * time.Minute),
	}

	require.NoError(t, repo.AddVisit(ctx, v1))
	require.NoError(t, repo.AddVisit(ctx, v2))
	require.NoError(t, repo.AddVisit(ctx, v3))
	require.NoError(t, repo.AddVisit(ctx, v4))
	require.NoError(t, repo.AddVisit(ctx, v4b))

	return []*Visit{v1, v2, v3, v4, v4b}
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

func TestRepo_AddVisit(t *testing.T) {
	ctx := context.Background()
	repo, shutdown := testRepoSetup(t)
	defer shutdown()

	count, err := repo.CountAll(ctx)
	require.NoError(t, err)

	err = repo.AddVisit(ctx, &Visit{})
	assert.Equal(t, "visit url or timestamp empty", err.Error())
	err = repo.AddVisit(ctx, &Visit{
		Title:     "test",
		Source:    "pc",
		URL:       "",
		Device:    "mb-serj",
		Timestamp: time.Now(),
	})
	assert.Equal(t, "visit url or timestamp empty", err.Error())

	now := time.Now()
	v1 := &Visit{
		Title:     gofakeit.Name(),
		Source:    "pc",
		Device:    "mb-serj",
		URL:       gofakeit.URL(),
		Timestamp: now,
	}
	v2 := &Visit{
		Title:     gofakeit.Name(),
		Source:    "safari",
		Device:    "mb-serj",
		URL:       gofakeit.URL(),
		Timestamp: now,
	}
	v3 := &Visit{
		Title:     gofakeit.Name(),
		Source:    "chrome",
		Device:    "mb-work1",
		URL:       gofakeit.URL(),
		Timestamp: now,
	}

	require.NoError(t, repo.AddVisit(ctx, v1))
	require.NoError(t, repo.AddVisit(ctx, v2))
	require.NoError(t, repo.AddVisit(ctx, v3))

	assert.NotEqual(t, v1.Id, v2.Id)
	assert.NotEqual(t, v1.Id, v3.Id)
	assert.NotEqual(t, v2.Id, v3.Id)
	assert.True(t, now.Equal(v1.Timestamp), "%v should be before %v", now, v1.Timestamp)
	assert.True(t, now.Equal(v2.Timestamp), "%v should be before %v", now, v2.Timestamp)
	assert.True(t, now.Equal(v2.Timestamp), "%v should be before %v", now, v3.Timestamp)

	countAfter, err := repo.CountAll(ctx)
	require.NoError(t, err)

	assert.Equal(t, 3, countAfter-count)

	allVisits, err := repo.GetAllVisits(ctx, nil)
	require.NoError(t, err)
	require.Len(t, allVisits, countAfter)

	var foundV1 *Visit
	for _, v := range allVisits {
		if v.Id == v1.Id {
			foundV1 = v
			break
		}
	}
	require.NotNil(t, foundV1)
	foundV1.Timestamp = foundV1.Timestamp.Truncate(time.Minute)
	v1.Timestamp = v1.Timestamp.Truncate(time.Minute)
	assert.Equal(t, *foundV1, *v1)
}

func TestRepo_GetAllVisits(t *testing.T) {
	ctx := context.Background()
	repo, shutdown := testRepoSetup(t)
	defer shutdown()

	_, err := deleteAllVisits(ctx, repo)
	require.NoError(t, err)

	allVisits, err := repo.GetAllVisits(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, allVisits)

	now := time.Now()
	addedCount := 10
	for i := 1; i <= 10; i++ {
		repo.AddVisit(ctx, &Visit{
			Title:     gofakeit.Name(),
			Source:    "pc",
			Device:    "mb-serj",
			URL:       gofakeit.URL(),
			Timestamp: now.Add(time.Duration(i) * time.Minute),
		})
	}

	allVisits, err = repo.GetAllVisits(ctx, &now)
	require.NoError(t, err)
	assert.Len(t, allVisits, addedCount)

	after5mins := now.Add(5 * time.Minute)
	after5minutesVisits, err := repo.GetAllVisits(ctx, &after5mins)
	require.NoError(t, err)
	assert.Len(t, after5minutesVisits, 6)
}

func TestRepo_GetVisits_and_Count(t *testing.T) {
	ctx := context.Background()
	repo, shutdown := testRepoSetup(t)
	defer shutdown()

	_ = cleanupAndAddTestVisits(ctx, t, repo)

	allVisits, err := repo.CountAll(ctx)
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
		gottenVisits, err := repo.GetVisits(ctx, tc.keywords, tc.field, tc.source, tc.limit)
		require.NoError(t, err)
		assert.Len(t, gottenVisits, tc.expectedVisitsCount, fmt.Sprintf("retrieved visits invalid for test case: %v", tc))

		gottenCount, err := repo.Count(ctx, tc.keywords, tc.field, tc.source)
		require.NoError(t, err)
		if tc.source != "all" {
			assert.Equal(t, tc.expectedVisitsCount, gottenCount, fmt.Sprintf("count invalid for test case: %+v", tc))
		} else {
			assert.Equal(t, 5, gottenCount, fmt.Sprintf("count invalid for test case: %+v", tc))
		}
	}

	visits, err := repo.GetVisits(ctx, []string{"four"}, "url", "chrome", 10)
	require.NoError(t, err)
	require.Len(t, visits, 2)
	assert.True(t, strings.HasPrefix(visits[0].Title, "title four"))
	assert.True(t, strings.HasPrefix(visits[1].Title, "title four"))
}

func TestRepo_GetVisitsPage(t *testing.T) {
	ctx := context.Background()
	repo, shutdown := testRepoSetup(t)
	defer shutdown()

	addedVisits := cleanupAndAddTestVisits(ctx, t, repo)
	v1, v2, v3, v4, v4b := addedVisits[0], addedVisits[1], addedVisits[2], addedVisits[3], addedVisits[4]

	allVisits, err := repo.CountAll(ctx)
	require.NoError(t, err)
	require.Equal(t, allVisits, 5)

	gottenVisits, err := repo.GetVisitsPage(ctx, []string{"www"}, "url", "all", 3, 2)
	require.NoError(t, err)
	assert.Len(t, gottenVisits, 2)
	assert.Equal(t, v4b.Id, gottenVisits[1].Id)
	assert.Equal(t, v4.Id, gottenVisits[0].Id)
	gottenVisits, err = repo.GetVisitsPage(ctx, []string{"www"}, "url", "all", 2, 2)
	require.NoError(t, err)
	assert.Len(t, gottenVisits, 2)
	assert.Equal(t, v4.Id, gottenVisits[1].Id)
	assert.Equal(t, v3.Id, gottenVisits[0].Id)
	gottenVisits, err = repo.GetVisitsPage(ctx, []string{"www"}, "url", "all", 1, 2)
	require.NoError(t, err)
	assert.Len(t, gottenVisits, 2)
	assert.Equal(t, v2.Id, gottenVisits[1].Id)
	assert.Equal(t, v1.Id, gottenVisits[0].Id)
	gottenVisits, err = repo.GetVisitsPage(ctx, []string{"www"}, "url", "all", 0, 2)
	require.NoError(t, err)
	assert.Len(t, gottenVisits, 0)
	gottenVisits, err = repo.GetVisitsPage(ctx, []string{"www"}, "url", "all", 1, 20)
	require.NoError(t, err)
	assert.Len(t, gottenVisits, 5)
	gottenVisits, err = repo.GetVisitsPage(ctx, []string{"title"}, "title", "all", 1, 20)
	require.NoError(t, err)
	assert.Len(t, gottenVisits, 5)
	gottenVisits, err = repo.GetVisitsPage(ctx, []string{"www"}, "url", "chrome", 1, 2)
	require.NoError(t, err)
	assert.Len(t, gottenVisits, 2)
	assert.Equal(t, v4.Id, gottenVisits[1].Id)
	assert.Equal(t, v3.Id, gottenVisits[0].Id)
}
