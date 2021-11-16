package internal

import (
	"testing"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/file_box"
	"github.com/stretchr/testify/assert"
)

func TestNewFileHandler(t *testing.T) {
	api := file_box.NewDiskTestApi()
	loginChecker := auth.NewLoginTestChecker()
	fileHandler := NewFileHandler(api, loginChecker)
	assert.NotNil(t, fileHandler)
}
