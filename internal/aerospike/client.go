package aerospike

import "errors"

var (
	ErrAeroClientNil          = errors.New("aero client is nil")
	ErrAeroClientNotConnected = errors.New("aero client is not connected")
	ErrEmptyNamespace         = errors.New("namespace cannot be empty")
)

type AeroBinMap map[string]interface{}

type Client interface {
	Put(key string, binMap AeroBinMap) error
	Delete(key string) (bool, error)
	QueryByRange(index string, from, to int64) ([]AeroBinMap, error)
	ScanAll() ([]AeroBinMap, error)
	CountAll() (int, error)

	IsConnected() bool
	Close()
}
