package aerospike

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
