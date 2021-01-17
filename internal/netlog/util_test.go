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
	assert.Equal(t, "where url like '%word1%' ", queryLike)

	// more keywords
	keywords = []string{"word1", "word2", "word3"}
	queryLike = getQueryLikeCondition(keywords)
	assert.Equal(t, "where url like '%word1%' and url like '%word2%' and url like '%word3%' ", queryLike)
}
