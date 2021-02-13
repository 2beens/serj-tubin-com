package netlog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestPsqlApi_AddVisit(t *testing.T) {
	// TODO: maybe add some tests against the real DB
	// PostgreSQL containers on GitHub Actions:
	//	https://docs.github.com/en/actions/guides/creating-postgresql-service-containers
}
