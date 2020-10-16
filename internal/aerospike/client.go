package aerospike

type AeroBinMap map[string]interface{}

type Client interface {
	Put(set, key string, binMap AeroBinMap) error
	QueryByRange(set string, index string, from, to int64) ([]AeroBinMap, error)
	ScanAll(set string) ([]AeroBinMap, error)
	CountAll(set string) (int, error)

	IsConnected() bool
	Close()
}
