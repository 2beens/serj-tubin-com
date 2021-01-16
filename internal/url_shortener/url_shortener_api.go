package url_shortener

type UrlShortenerApi struct {
	aeroClient *UrlShortenerAeroClient
}

func NewUrlShortenerApi(aeroClient *UrlShortenerAeroClient) *UrlShortenerApi {
	return &UrlShortenerApi{
		aeroClient: aeroClient,
	}
}

func (api *UrlShortenerApi) NewUrl() (string, error) {
	panic("")
}
