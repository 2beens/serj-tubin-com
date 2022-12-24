package internal

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	// TODO: check how's this used properly

	goleak.VerifyTestMain(m)
}
