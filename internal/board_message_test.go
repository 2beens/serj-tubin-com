package internal

import (
	"testing"

	"time"

	"github.com/2beens/serjtubincom/internal/aerospike"
	"github.com/stretchr/testify/assert"
)

func TestMessageFromBins(t *testing.T) {
	bins := aerospike.AeroBinMap{}

	message := MessageFromBins(bins)
	assert.Equal(t, 0, message.ID)
	assert.Empty(t, message.Message)
	assert.Equal(t, int64(0), message.Timestamp)
	assert.Empty(t, message.Author)

	now := time.Now()
	bins = aerospike.AeroBinMap{
		"id":        1,
		"message":   "test_msg",
		"author":    "test_author",
		"timestamp": now.Unix(),
	}

	message = MessageFromBins(bins)
	assert.Equal(t, 1, message.ID)
	assert.Equal(t, "test_msg", message.Message)
	assert.Equal(t, now.Unix(), message.Timestamp)
	assert.Equal(t, "test_author", message.Author)
}
