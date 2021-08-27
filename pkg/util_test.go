package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBytesToString(t *testing.T) {
	want := "test"
	stringBytes := []byte(want)
	got := BytesToString(stringBytes)
	assert.Equal(t, want, got)
}
