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

func TestPathExists(t *testing.T) {
	exists, err := PathExists("/invalid/path/some-dir", true)
	assert.NoError(t, err)
	assert.False(t, exists)
	exists, err = PathExists("/invalid/path/some-file", false)
	assert.NoError(t, err)
	assert.False(t, exists)

	tempDir := t.TempDir()
	exists, err = PathExists(tempDir, true)
	assert.NoError(t, err)
	assert.True(t, exists)
	exists, err = PathExists(tempDir, false)
	assert.NoError(t, err)
	assert.False(t, exists)
}
