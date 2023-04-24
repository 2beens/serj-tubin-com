package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	passwordHash, err := HashPassword("sr")
	require.NoError(t, err)
	assert.NotEmpty(t, passwordHash)
	assert.True(t, CheckPasswordHash("sr", "$2a$14$z8cd4yJpzP40Qh2F2BhiMO.sOm4YAIaf30pmUKLOaISojD9HnXgaG"))
	assert.True(t, CheckPasswordHash("sr", passwordHash))

	passwordHash, err = HashPassword("todo")
	require.NoError(t, err)
	assert.NotEmpty(t, passwordHash)
	assert.True(t, CheckPasswordHash("todo", "$2a$14$H5aVoE1YSTxBF63MLgBfo.u0W7vNcx5JQb7LUix.DicQv3WESnYuq"))
}
