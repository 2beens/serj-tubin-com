package geoip

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/go-redis/redis/v8"
	"github.com/ipinfo/go/v2/ipinfo"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
)

type Api struct {
	ipInfoClient *ipinfo.Client
	redisClient  *redis.Client
}

var (
	localIP      = "127.0.0.1"
	devGeoIpInfo = &ipinfo.Core{
		City:        "Berlin",
		Region:      "Berlin",
		Country:     "Germany",
		CountryName: "Germany",
		IsEU:        true,
		Postal:      "12099",
	}
)

func NewApi(
	ipInfoAPIKey string,
	httpClient *http.Client,
	redisClient *redis.Client,
) *Api {
	return &Api{
		ipInfoClient: ipinfo.NewClient(httpClient, nil, ipInfoAPIKey),
		redisClient:  redisClient,
	}
}

func (gi *Api) GetIPGeoInfo(ctx context.Context, userIp string) (*ipinfo.Core, error) {
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
	cmd := gi.redisClient.Get(ctx, userIpKey)
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

	ipInfo, err = gi.ipInfoClient.GetIPInfo(net.ParseIP(userIp))
	if err != nil {
		return nil, fmt.Errorf("call ip info api: %w", err)
	}

	ipInfoJson, err := json.Marshal(ipInfo)
	if err != nil {
		return nil, fmt.Errorf("marshal ip info api received object: %w", err)
	}

	// cache response object in redis
	cmdSet := gi.redisClient.Set(ctx, userIpKey, ipInfoJson, 0)
	if err := cmdSet.Err(); err != nil {
		log.Errorf("failed to cache ip info in redis for %s: %s", userIp, err)
	} else {
		log.Debugf("ip info cache set in redis for: %s", userIp)
	}

	return ipInfo, nil
}
