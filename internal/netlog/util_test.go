package netlog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUtil_getQueryLikeCondition(t *testing.T) {
	// no keywords
	queryLike := getQueryLikeCondition("url", []string{})
	assert.Empty(t, queryLike)

	// only one keyword
	keywords := []string{"word1"}
	queryLike = getQueryLikeCondition("url", keywords)
	assert.Equal(t, "WHERE url LIKE '%word1%' ", queryLike)

	keywords = []string{"word1"}
	queryLike = getQueryLikeCondition("title", keywords)
	assert.Equal(t, "WHERE title LIKE '%word1%' ", queryLike)

	// more keywords
	keywords = []string{"word1", "word2", "word3"}
	queryLike = getQueryLikeCondition("url", keywords)
	assert.Equal(t, "WHERE url LIKE '%word1%' AND url LIKE '%word2%' AND url LIKE '%word3%' ", queryLike)

	keywords = []string{"word1", "word2", "word3"}
	queryLike = getQueryLikeCondition("title", keywords)
	assert.Equal(t, "WHERE title LIKE '%word1%' AND title LIKE '%word2%' AND title LIKE '%word3%' ", queryLike)
}
