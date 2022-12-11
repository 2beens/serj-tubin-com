package geoip

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/go-redis/redis/v8"
	"github.com/ipinfo/go/v2/ipinfo"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var (
	devGeoIpInfo = &ipinfo.Core{
		City:    "Berlin",
		Region:  "Berlin",
		Country: "DE",
		IsEU:    true,
		Postal:  "12099",
	}
)

const (
	DefaultIpInfoBaseURL = "https://ipinfo.io/"
)

type Api struct {
	ipInfoBaseURL string
	ipInfoAPIKey  string
	httpClient    *http.Client
	redisClient   *redis.Client
}

func NewApi(
	ipInfoBaseURL string,
	ipInfoAPIKey string,
	httpClient *http.Client,
	redisClient *redis.Client,
) *Api {
	return &Api{
		ipInfoBaseURL: ipInfoBaseURL,
		ipInfoAPIKey:  ipInfoAPIKey,
		httpClient:    httpClient,
		redisClient:   redisClient,
	}
}

func (api *Api) GetIPGeoInfo(ctx context.Context, userIp string) (*ipinfo.Core, error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "geoIp.getRequestGeoInfo")
	defer span.End()

	span.SetAttributes(attribute.String("user.ip", userIp))

	// used for development
	if userIp == "localhost" {
		log.Debugf("request geo info: returning development localhost / Berlin")
		return devGeoIpInfo, nil
	}

	// try to get geo ip info from redis
	userIpKey := fmt.Sprintf("ip-info::%s", userIp)
	cmd := api.redisClient.Get(ctx, userIpKey)
	if err := cmd.Err(); err != nil {
		log.Errorf("failed to find ip info from redis for [%s]: %s", userIpKey, err)
	}

	var err error
	ipInfo := &ipinfo.Core{}
	if ipInfoBytes := cmd.Val(); ipInfoBytes != "" {
		span.SetAttributes(attribute.Bool("user.ip.from-cache", true))
		log.Tracef("found geo ip info for [%s] in redis cache", userIp)
		if err = json.Unmarshal([]byte(ipInfoBytes), ipInfo); err == nil {
			if ipInfo.City != "" && ipInfo.Country != "" {
				return ipInfo, nil
			}
		}

		log.Errorf("failed to unmarshal cached ip info from redis for %s: %s", userIp, err)
		// continue, and try getting it from IP Info API
	} else {
		span.SetAttributes(attribute.Bool("user.ip.from-cache", false))
		log.Debugf("ip info value from redis not found for [%s]", userIp)
	}

	log.Debugf("will ask ip info API for ip info: %s", userIp)

	ipInfo, err = api.getIPInfo(ctx, userIp)
	if err != nil {
		return nil, fmt.Errorf("call ip info api: %w", err)
	}

	ipInfoJson, err := json.Marshal(ipInfo)
	if err != nil {
		return nil, fmt.Errorf("marshal ip info api received object: %w", err)
	}

	// cache response object in redis
	cmdSet := api.redisClient.Set(ctx, userIpKey, ipInfoJson, 0)
	if err := cmdSet.Err(); err != nil {
		log.Errorf("failed to cache ip info in redis for %s: %s", userIp, err)
	} else {
		log.Debugf("ip info cache set in redis for: %s", userIp)
	}

	return ipInfo, nil
}

func (api *Api) getIPInfo(ctx context.Context, ip string) (*ipinfo.Core, error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "geoIp.getIPInfo")
	defer span.End()

	req, err := api.newRequest(ctx, "GET", ip, nil)
	if err != nil {
		return nil, err
	}

	ipInfoVal := new(ipinfo.Core)
	if _, err := api.do(ctx, req, ipInfoVal); err != nil {
		return nil, err
	}

	// format
	if ipInfoVal.Country != "" {
		ipInfoVal.CountryName = ipinfo.GetCountryName(ipInfoVal.Country)
		ipInfoVal.IsEU = ipinfo.IsEU(ipInfoVal.Country)
		ipInfoVal.CountryFlag.Emoji = ipinfo.GetCountryFlagEmoji(ipInfoVal.Country)
		ipInfoVal.CountryFlag.Unicode = ipinfo.GetCountryFlagUnicode(ipInfoVal.Country)
		ipInfoVal.CountryCurrency.Code = ipinfo.GetCountryCurrencyCode(ipInfoVal.Country)
		ipInfoVal.CountryCurrency.Symbol = ipinfo.GetCountryCurrencySymbol(ipInfoVal.Country)
		ipInfoVal.Continent.Code = ipinfo.GetContinentCode(ipInfoVal.Country)
		ipInfoVal.Continent.Name = ipinfo.GetContinentName(ipInfoVal.Country)
	}
	if ipInfoVal.Abuse != nil && ipInfoVal.Abuse.Country != "" {
		ipInfoVal.Abuse.CountryName = ipinfo.GetCountryName(ipInfoVal.Abuse.Country)
	}

	return ipInfoVal, nil
}

// `newRequest` creates an API request. A relative URL can be provided in
// urlStr, in which case it is resolved relative to the BaseURL of the Client.
// Relative URLs should always be specified without a preceding slash.
func (api *Api) newRequest(
	ctx context.Context,
	method string,
	urlStr string,
	body io.Reader,
) (*http.Request, error) {
	u := new(url.URL)

	baseURL, err := url.Parse(api.ipInfoBaseURL)
	if err != nil {
		return nil, fmt.Errorf("get base url: %w", err)
	}

	// get final URL path.
	if rel, err := url.Parse(urlStr); err == nil {
		u = baseURL.ResolveReference(rel)
	} else if strings.ContainsRune(urlStr, ':') {
		// IPv6 strings fail to parse as URLs, so let's add it as a URL Path.
		*u = *baseURL
		u.Path += urlStr
	} else {
		return nil, err
	}

	// get `http` package request object.
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}

	// set common headers.
	req.Header.Set("Accept", "application/json")
	//if api.UserAgent != "" {
	//	req.Header.Set("User-Agent", api.UserAgent)
	//}
	if api.ipInfoAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+api.ipInfoAPIKey)
	}

	return req, nil
}

// `do` sends an API request and returns the API response. The API response is
// JSON decoded and stored in the value pointed to by v, or returned as an
// error if an API error has occurred. If v implements the io.Writer interface,
// the raw response body will be written to v, without attempting to first
// decode it.
func (api *Api) do(
	ctx context.Context,
	req *http.Request,
	v interface{},
) (_ *http.Response, err error) {
	_, span := tracing.GlobalTracer.Start(ctx, "geoIp.do")
	defer func() {
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		} else {
			span.SetStatus(codes.Ok, "do done")
		}
		span.End()
	}()

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := api.checkResponse(resp); err != nil {
		// even though there was an error, we still return the response
		// in case the caller wants to inspect it further
		return resp, err
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			if _, err := io.Copy(w, resp.Body); err != nil {
				return resp, err
			}
		} else {
			err = json.NewDecoder(resp.Body).Decode(v)
			if err == io.EOF {
				// ignore EOF errors caused by empty response body
				err = nil
			}
		}
	}

	return resp, err
}

// `checkResponse` checks the API response for errors, and returns them if
// present. A response is considered an error if it has a status code outside
// the 200 range.
func (api *Api) checkResponse(r *http.Response) error {
	if c := r.StatusCode; 200 >= c && c <= 299 {
		return nil
	}
	errorResponse := &ipinfo.ErrorResponse{Response: r}
	data, err := io.ReadAll(r.Body)
	if err == nil && data != nil {
		if err := json.Unmarshal(data, errorResponse); err != nil {
			return fmt.Errorf("unmarshal err resp: %w", err)
		}
	}
	return errorResponse
}
