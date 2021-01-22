package netlog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUtil_getQueryLikeCondition(t *testing.T) {
	// no keywords
	queryLike := getQueryLikeCondition([]string{})
	assert.Empty(t, queryLike)

	// only one keyword
	keywords := []string{"word1"}
	queryLike = getQueryLikeCondition(keywords)
	assert.Equal(t, "WHERE url LIKE '%word1%' ", queryLike)

	// more keywords
	keywords = []string{"word1", "word2", "word3"}
	queryLike = getQueryLikeCondition(keywords)
	assert.Equal(t, "WHERE url LIKE '%word1%' AND url LIKE '%word2%' AND url LIKE '%word3%' ", queryLike)
}
