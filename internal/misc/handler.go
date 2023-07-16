package misc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/geoip"
	"github.com/2beens/serjtubincom/internal/middleware"
	"github.com/2beens/serjtubincom/internal/telemetry/metrics"
	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

//go:generate mockgen -source=$GOFILE -destination=mocks_test.go -package=misc_test
//go:generate mockgen -source=../middleware/rate_limiting.go -destination=mocks_rate_limiter_test.go -package=misc_test

type authService interface {
	Login(ctx context.Context, creds auth.Credentials, createdAt time.Time) (string, error)
	Logout(ctx context.Context, token string) (bool, error)
}

type Handler struct {
	geoIp         *geoip.Api
	quotesManager *QuotesManager
	versionInfo   string
	authService   authService
}

func NewHandler(
	geoIp *geoip.Api,
	quotesManager *QuotesManager,
	versionInfo string,
	authService authService,
) *Handler {
	return &Handler{
		geoIp:         geoIp,
		quotesManager: quotesManager,
		versionInfo:   versionInfo,
		authService:   authService,
	}
}

func (handler *Handler) SetupRoutes(
	mainRouter *mux.Router,
	rateLimiter middleware.RequestRateLimiter,
	metricsManager *metrics.Manager,
	loginRateLimitAllowedPerMin int,
) {
	mainRouter.HandleFunc("/", handler.handleRoot).Methods("GET", "POST", "OPTIONS").Name("root")
	mainRouter.HandleFunc("/quote/random", handler.handleGetRandomQuote).Methods("GET").Name("quote")
	mainRouter.HandleFunc("/whereami", handler.handleWhereAmI).Methods("GET").Name("whereami")
	mainRouter.HandleFunc("/myip", handler.handleGetMyIp).Methods("GET").Name("myip")
	mainRouter.HandleFunc("/version", handler.handleGetVersionInfo).Methods("GET").Name("version")

	loginSubrouter := mainRouter.PathPrefix("/a").Subrouter()
	loginSubrouter.
		HandleFunc("/login", handler.handleLogin).
		Methods("POST", "OPTIONS").Name("login")
	loginSubrouter.
		HandleFunc("/logout", handler.handleLogout).
		Methods("GET", "OPTIONS").Name("logout")

	// rate limit the /login and /logout endpoints to prevent abuse
	loginSubrouter.Use(middleware.RateLimit(
		rateLimiter,
		"login",
		loginRateLimitAllowedPerMin,
		metricsManager,
	))
	loginSubrouter.Use(middleware.Cors())
}

func (handler *Handler) handleRoot(w http.ResponseWriter, _ *http.Request) {
	pkg.WriteTextResponseOK(w, "I'm OK, thanks ;)")
}

func (handler *Handler) handleGetRandomQuote(w http.ResponseWriter, r *http.Request) {
	_, span := tracing.GlobalTracer.Start(r.Context(), "miscHandler.quote")
	defer span.End()

	w.Header().Set("Content-Type", "application/json")

	q := handler.quotesManager.RandomQuote()
	qBytes, err := json.Marshal(q)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		log.Errorf("marshal quote error: %s", err)
		return
	}

	pkg.WriteResponseBytesOK(w, pkg.ContentType.JSON, qBytes)
}

func (handler *Handler) handleWhereAmI(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "miscHandler.whereAmI")
	defer span.End()

	w.Header().Set("Content-Type", "application/json")

	userIP, err := pkg.ReadUserIP(r)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("get user ip: %s", err))
		http.Error(w, "geo ip info error", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(attribute.String("user.ip", userIP))

	ipInfo, err := handler.geoIp.GetIPGeoInfo(ctx, userIP)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("get request geo info: %s", err))
		log.Errorf("error getting geo ip info: %s", err)
		http.Error(w, "geo ip info error", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(attribute.String("user.city", ipInfo.City))
	span.SetAttributes(attribute.String("user.country", ipInfo.Country))

	geoResp := fmt.Sprintf(`{"city":"%s", "country":"%s"}`, ipInfo.City, ipInfo.Country)
	pkg.WriteJSONResponseOK(w, geoResp)
}

func (handler *Handler) handleGetMyIp(w http.ResponseWriter, r *http.Request) {
	_, span := tracing.GlobalTracer.Start(r.Context(), "miscHandler.getMyIp")
	defer span.End()

	ip, err := pkg.ReadUserIP(r)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("failed to get user IP address: %s", err))
		log.Errorf("failed to get user IP address: %s", err)
		http.Error(w, "failed to get IP", http.StatusInternalServerError)
	}

	span.SetAttributes(attribute.String("user.ip", ip))
	span.SetStatus(codes.Ok, fmt.Sprintf("user IP address: %s", ip))
	pkg.WriteTextResponseOK(w, ip)
}

type LoginResponse struct {
	Token string `json:"token"`
}

func (handler *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "miscHandler.login")
	defer span.End()

	if r.Method == http.MethodOptions {
		w.Header().Add("Allow", "POST, OPTIONS")
		w.WriteHeader(http.StatusOK)
		return
	}

	var loginReq LoginRequest
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
			log.Errorf("login, unmarshal json params: %s", err)
			http.Error(w, "login failed", http.StatusBadRequest)
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			log.Errorf("login failed, parse form error: %s", err)
			http.Error(w, "parse form error", http.StatusInternalServerError)
			return
		}
		loginReq = LoginRequest{
			Username: r.Form.Get("username"),
			Password: r.Form.Get("password"),
		}
	}

	if loginReq.Username == "" {
		http.Error(w, "error, username empty", http.StatusBadRequest)
		return
	}
	if loginReq.Password == "" {
		http.Error(w, "error, password empty", http.StatusBadRequest)
		return
	}

	token, err := handler.authService.Login(ctx, auth.Credentials{
		Username: loginReq.Username,
		Password: loginReq.Password,
	}, time.Now())
	if err != nil {
		log.Tracef("auth service login: %s", err)
		http.Error(w, "login failed", http.StatusBadRequest)
		return
	}

	tokenResp := LoginResponse{Token: token}
	tokenRespBytes, err := json.Marshal(tokenResp)
	if err != nil {
		log.Errorf("login, marshal token response: %s", err)
		http.Error(w, "login failed", http.StatusInternalServerError)
		return
	}

	log.Trace("new login success")
	pkg.WriteJSONResponseOK(w, string(tokenRespBytes))
}

func (handler *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	_, span := tracing.GlobalTracer.Start(r.Context(), "miscHandler.logout")
	defer span.End()

	if r.Method == http.MethodOptions {
		w.Header().Add("Allow", "POST, OPTIONS")
		w.WriteHeader(http.StatusOK)
		return
	}

	authToken := r.Header.Get("X-SERJ-TOKEN")
	if authToken == "" {
		http.Error(w, "no can do", http.StatusUnauthorized)
		return
	}

	loggedOut, err := handler.authService.Logout(r.Context(), authToken)
	if err != nil {
		log.Tracef("[failed login check] => %s: %s", r.URL.Path, err)
		http.Error(w, "no can do", http.StatusUnauthorized)
		return
	}
	if !loggedOut {
		http.Error(w, "no can do", http.StatusUnauthorized)
		return
	}

	log.Debugf("logout for [%s] success", authToken)
	pkg.WriteTextResponseOK(w, "logged-out")
}

func (handler *Handler) handleGetVersionInfo(w http.ResponseWriter, _ *http.Request) {
	pkg.WriteTextResponseOK(w, handler.versionInfo)
}
