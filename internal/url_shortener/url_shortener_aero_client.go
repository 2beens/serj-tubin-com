package url_shortener

import (
	"github.com/2beens/serjtubincom/internal/aerospike"
	as "github.com/aerospike/aerospike-client-go"
)

type UrlShortenerAeroClient struct {
	namespace  string
	set        string
	aeroClient *as.Client
}

func NewUrlShortenerAeroClient(aeroClient *as.Client, namespace, set string) (*UrlShortenerAeroClient, error) {
	if aeroClient == nil {
		return nil, aerospike.ErrAeroClientNil
	}
	if namespace == "" {
		return nil, aerospike.ErrEmptyNamespace
	}

	return &UrlShortenerAeroClient{
		namespace:  namespace,
		set:        set,
		aeroClient: aeroClient,
	}, nil
}

func (u *UrlShortenerAeroClient) Put(key string, binMap aerospike.AeroBinMap) error {
	panic("implement me")
}

func (u *UrlShortenerAeroClient) Delete(key string) (bool, error) {
	panic("implement me")
}

func (u *UrlShortenerAeroClient) Close() {
	if u.aeroClient == nil {
		return
	}
	u.aeroClient.Close()
}
