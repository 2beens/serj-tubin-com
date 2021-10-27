package file_box

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiskApi(t *testing.T) {
	handler, err := NewDiskApi("/var/invaliddir1234")
	assert.Error(t, err)
	assert.Nil(t, handler)

	tempDir := t.TempDir()
	handler, err = NewDiskApi(tempDir)
	require.NoError(t, err)
	assert.NotNil(t, handler)
}
