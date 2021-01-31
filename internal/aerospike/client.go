package aerospike

type Client interface {
	Close()
	GetMessageIdCounter() (int, error)
	IncrementMessageIdCounter(increment int) (int, error)

	Put(key string, binMap AeroBinMap) error
	Delete(key string) (bool, error)
	QueryByRange(index string, from, to int64) ([]AeroBinMap, error)
	ScanAll() ([]AeroBinMap, error)
	CountAll() (int, error)
}
