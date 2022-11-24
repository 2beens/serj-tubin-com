package geoip

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"

	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type Api struct {
	mu             sync.Mutex
	ipBaseEndpoint string
	ipBaseAPIKey   string
	httpClient     *http.Client
	redisClient    *redis.Client
}

var (
	devGeoIpInfo = IpInfo{
		Data: IpInfoData{
			IP: "127.0.0.1",
			Location: GeoLocation{
				City: City{
					Name: "Berlin",
				},
				Country: Country{
					Alpha2: "DE",
					Alpha3: "DEU",
					Name:   "Germany",
				},
			},
		},
	}
)

func NewApi(
	ipBaseEndpoint, ipBaseAPIKey string,
	httpClient *http.Client,
	redisClient *redis.Client,
) *Api {
	return &Api{
		ipBaseEndpoint: ipBaseEndpoint,
		ipBaseAPIKey:   ipBaseAPIKey,
		httpClient:     httpClient,
		redisClient:    redisClient,
	}
}

func (gi *Api) GetRequestGeoInfo(ctx context.Context, r *http.Request) (*IpInfo, error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "geoIp.getRequestGeoInfo")
	defer span.End()

	userIp, err := pkg.ReadUserIP(r)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("get user ip: %s", err))
		return nil, fmt.Errorf("get user ip: %w", err)
	}
	span.SetAttributes(attribute.String("user.ip", userIp))

	// used for development
	if userIp == "localhost" {
		log.Debugf("request geo info: returning development localhost / Berlin")
		return &devGeoIpInfo, nil
	}

	// ipbase api free plan contains only 150 calls
	// the frontend client makes 3 concurrent requests upon home page opened: /whereami, weather current,
	// and weather forecast (5 days); all these result in 3 concurrent (and unnecessary) requests to
	// my poor ipbase free plan, thus a mutex is required here to reduce this number, and try getting
	// cached ip info value from redis
	gi.mu.Lock()
	defer gi.mu.Unlock()

	// try to get geo ip info from redis
	userIpKey := fmt.Sprintf("ip-info::%s", userIp)
	cmd := gi.redisClient.Get(ctx, userIpKey)
	if err := cmd.Err(); err != nil {
		log.Errorf("failed to find ip info from redis for [%s]: %s", userIpKey, err)
	}

	geoIpResponse := &IpInfo{}
	if geoIpInfoBytes := cmd.Val(); geoIpInfoBytes != "" {
		span.SetAttributes(attribute.Bool("user.ip.from-cache", true))
		log.Tracef("found geo ip info for [%s] in redis cache", userIp)
		if err = json.Unmarshal([]byte(geoIpInfoBytes), geoIpResponse); err == nil {
			return geoIpResponse, nil
		}

		log.Errorf("failed to unmarshal cached ip info from redis for %s: %s", userIp, err)
		// continue, and try getting it from IP Base API
	} else {
		span.SetAttributes(attribute.Bool("user.ip.from-cache", false))
		log.Debugf("ip info value from redis not found for [%s]", userIp)
	}

	log.Debugf("will ask ip base API for ip info: %s", userIp)

	ipBaseUrl := fmt.Sprintf("%s/v2/info?apikey=%s&ip=%s", gi.ipBaseEndpoint, gi.ipBaseAPIKey, userIp)
	log.Debugf("calling geo ip info: %s", ipBaseUrl)

	req, err := http.NewRequest("GET", ipBaseUrl, nil)
	if err != nil {
		return nil, err
	}

	resp, err := gi.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("error getting freegeoip response: %s", err.Error())
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read geo ip response bytes: %s", err)
	}

	log.Debugf("calling ip base info for ip: %s, response: %s", ipBaseUrl, respBytes)

	err = json.Unmarshal(respBytes, geoIpResponse)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("unmarshal geo ip resp: %s", err))
		return nil, fmt.Errorf("unmarshal geo ip response bytes: %w", err)
	}

	// cache response in redis
	cmdSet := gi.redisClient.Set(ctx, userIpKey, respBytes, 0)
	if err := cmdSet.Err(); err != nil {
		log.Errorf("failed to cache ip info in redis for %s: %s", userIp, err)
	} else {
		log.Debugf("ip info cache set in redis for: %s", userIp)
	}

	return geoIpResponse, nil
}

type IpInfo struct {
	Data IpInfoData `json:"data"`
}

type IpInfoData struct {
	Timezone struct {
		ID               string    `json:"id"`
		CurrentTime      time.Time `json:"current_time"`
		Code             string    `json:"code"`
		IsDaylightSaving bool      `json:"is_daylight_saving"`
		GmtOffset        int       `json:"gmt_offset"`
	} `json:"timezone"`
	IP         string `json:"ip"`
	Type       string `json:"type"`
	Connection struct {
		Asn          int    `json:"asn"`
		Organization string `json:"organization"`
		Isp          string `json:"isp"`
	} `json:"connection"`
	Location GeoLocation `json:"location"`
}

type GeoLocation struct {
	GeonamesID int     `json:"geonames_id"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Zip        string  `json:"zip"`
	Continent  struct {
		Code           string `json:"code"`
		Name           string `json:"name"`
		NameTranslated string `json:"name_translated"`
	} `json:"continent"`
	Country Country `json:"country"`
	City    City    `json:"city"`
	Region  Region  `json:"region"`
}

type Region struct {
	Fips           string `json:"fips"`
	Alpha2         string `json:"alpha2"`
	Name           string `json:"name"`
	NameTranslated string `json:"name_translated"`
}

type City struct {
	Name           string `json:"name"`
	NameTranslated string `json:"name_translated"`
}

type Country struct {
	Alpha2       string     `json:"alpha2"`
	Alpha3       string     `json:"alpha3"`
	CallingCodes []string   `json:"calling_codes"`
	Currencies   []Currency `json:"currencies"`
	Emoji        string     `json:"emoji"`
	Ioc          string     `json:"ioc"`
	Languages    []struct {
		Name       string `json:"name"`
		NameNative string `json:"name_native"`
	} `json:"languages"`
	Name              string   `json:"name"`
	NameTranslated    string   `json:"name_translated"`
	Timezones         []string `json:"timezones"`
	IsInEuropeanUnion bool     `json:"is_in_european_union"`
}

type Currency struct {
	Symbol        string `json:"symbol"`
	Name          string `json:"name"`
	SymbolNative  string `json:"symbol_native"`
	DecimalDigits int    `json:"decimal_digits"`
	Rounding      int    `json:"rounding"`
	Code          string `json:"code"`
	NamePlural    string `json:"name_plural"`
}
