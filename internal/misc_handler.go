package internal

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type MiscHandler struct {
	geoIp         *GeoIp
	quotesManager *QuotesManager
	versionInfo   string
}

func NewMiscHandler(
	mainRouter *mux.Router,
	geoIp *GeoIp,
	quotesManager *QuotesManager,
	versionInfo string,
) *MiscHandler {
	handler := &MiscHandler{
		geoIp:         geoIp,
		quotesManager: quotesManager,
		versionInfo:   versionInfo,
	}

	mainRouter.HandleFunc("/", handler.handleRoot).Methods("GET", "POST", "OPTIONS").Name("root")
	mainRouter.HandleFunc("/quote/random", handler.handleGetRandomQuote).Methods("GET").Name("quote")
	mainRouter.HandleFunc("/whereami", handler.handleWhereAmI).Methods("GET").Name("whereami")
	mainRouter.HandleFunc("/myip", handler.handleGetMyIp).Methods("GET").Name("myip")
	mainRouter.HandleFunc("/version", handler.handleGetVersionInfo).Methods("GET").Name("version")

	// all the rest - unhandled paths
	mainRouter.HandleFunc("/{unknown}", handler.handleUnknownPath).Methods("GET", "POST", "PUT", "OPTIONS").Name("unknown")

	return handler
}

func (handler *MiscHandler) handleRoot(w http.ResponseWriter, r *http.Request) {
	WriteResponse(w, "", "I'm OK, thanks")
}

func (handler *MiscHandler) handleGetRandomQuote(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	q := handler.quotesManager.RandomQuote()
	qBytes, err := json.Marshal(q)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		log.Errorf("marshal quote error: %s", err)
		return
	}

	WriteResponseBytes(w, "", qBytes)
}

func (handler *MiscHandler) handleWhereAmI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	geoIpInfo, err := handler.geoIp.GetRequestGeoInfo(r)
	if err != nil {
		log.Errorf("error getting geo ip info: %s", err)
		http.Error(w, "geo ip info error", http.StatusInternalServerError)
		return
	}

	geoResp := fmt.Sprintf(`{"city":"%s", "country":"%s"}`, geoIpInfo.City, geoIpInfo.CountryName)
	WriteResponse(w, "application/json", geoResp)
}

func (handler *MiscHandler) handleGetMyIp(w http.ResponseWriter, r *http.Request) {
	ip, err := handler.geoIp.ReadUserIP(r)
	if err != nil {
		log.Errorf("failed to get user IP address: %s", err)
		http.Error(w, "failed to get IP", http.StatusInternalServerError)
	}
	WriteResponse(w, "", ip)
}

func (handler *MiscHandler) handleGetVersionInfo(w http.ResponseWriter, r *http.Request) {
	WriteResponse(w, "", handler.versionInfo)
}

func (handler *MiscHandler) handleUnknownPath(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
