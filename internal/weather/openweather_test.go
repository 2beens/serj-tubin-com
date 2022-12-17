package weather

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadCitiesData(t *testing.T) {
	cities, err := LoadCitiesData()
	require.NoError(t, err)
	assert.Len(t, cities, 209579)
}
