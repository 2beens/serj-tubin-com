package pkg

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombinedWriter_Write(t *testing.T) {
	sb1 := &strings.Builder{}
	initMessage := "already-here"
	sb1.WriteString(initMessage)
	sb2 := &strings.Builder{}

	cw := NewCombinedWriter(sb1, sb2)
	require.NotNil(t, cw)
	assert.Len(t, cw.Writers, 2)
	assert.NoError(t, cw.Err)

	msg1 := "a message"
	msg2 := "another message here"
	n, err := cw.Write([]byte(msg1))
	require.NoError(t, err)
	assert.Equal(t, len(msg1)*len(cw.Writers), n)
	n, err = cw.Write([]byte(msg2))
	require.NoError(t, err)
	assert.Equal(t, len(msg2)*len(cw.Writers), n)

	assert.Equal(t, initMessage+msg1+msg2, sb1.String())
	assert.Equal(t, msg1+msg2, sb2.String())
}

func TestCombinedWriter_Write_WithError(t *testing.T) {
	fw := &faultyWritter{}
	sb := &strings.Builder{}

	cw := NewCombinedWriter(fw, sb)
	require.NotNil(t, cw)
	assert.Len(t, cw.Writers, 2)
	assert.NoError(t, cw.Err)

	msg1 := "a message"
	n, err := cw.Write([]byte(msg1))
	assert.Error(t, err, "eotikurac")

	// written only to string builder
	assert.Equal(t, len(msg1), n)
	assert.Equal(t, msg1, sb.String())
}

type faultyWritter struct{}

func (fw *faultyWritter) Write(p []byte) (n int, err error) {
	return 0, errors.New("eotikurac")
}
