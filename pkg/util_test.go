package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestGenerateRandomString(t *testing.T) {
	s, err := GenerateRandomString(0)
	require.Error(t, err)
	assert.Empty(t, s)

	for i := 1; i <= 8; i++ {
		s, err := GenerateRandomString(i * 5)
		require.NoError(t, err)
		assert.Len(t, s, i*5)
	}
}

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
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not a directory")
	assert.False(t, exists)
}
