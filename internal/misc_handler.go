package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type MiscHandler struct {
	geoIp         *GeoIp
	quotesManager *QuotesManager
	versionInfo   string
	authService   *auth.Service
	admin         *auth.Admin
}

func NewMiscHandler(
	mainRouter *mux.Router,
	geoIp *GeoIp,
	quotesManager *QuotesManager,
	versionInfo string,
	authService *auth.Service,
	admin *auth.Admin,
) *MiscHandler {
	handler := &MiscHandler{
		geoIp:         geoIp,
		quotesManager: quotesManager,
		versionInfo:   versionInfo,
		authService:   authService,
		admin:         admin,
	}

	mainRouter.HandleFunc("/", handler.handleRoot).Methods("GET", "POST", "OPTIONS").Name("root")
	mainRouter.HandleFunc("/quote/random", handler.handleGetRandomQuote).Methods("GET").Name("quote")
	mainRouter.HandleFunc("/whereami", handler.handleWhereAmI).Methods("GET").Name("whereami")
	mainRouter.HandleFunc("/myip", handler.handleGetMyIp).Methods("GET").Name("myip")
	mainRouter.HandleFunc("/version", handler.handleGetVersionInfo).Methods("GET").Name("version")

	mainRouter.HandleFunc("/login", handler.handleLogin).Methods("POST").Name("login")
	mainRouter.HandleFunc("/logout", handler.handleLogout).Methods("GET", "OPTIONS").Name("logout")

	return handler
}

func (handler *MiscHandler) handleRoot(w http.ResponseWriter, r *http.Request) {
	WriteResponse(w, "", "I'm OK, thanks ;)")
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

	geoResp := fmt.Sprintf(`{"city":"%s", "country":"%s"}`, geoIpInfo.Data.Location.City.Name, geoIpInfo.Data.Location.Country.Name)
	WriteResponse(w, "application/json", geoResp)
}

func (handler *MiscHandler) handleGetMyIp(w http.ResponseWriter, r *http.Request) {
	ip, err := ReadUserIP(r)
	if err != nil {
		log.Errorf("failed to get user IP address: %s", err)
		http.Error(w, "failed to get IP", http.StatusInternalServerError)
	}
	WriteResponse(w, "", ip)
}

func (handler *MiscHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Errorf("login failed, parse form error: %s", err)
		http.Error(w, "parse form error", http.StatusInternalServerError)
		return
	}

	username := r.Form.Get("username")
	if username == "" {
		http.Error(w, "error, username empty", http.StatusBadRequest)
		return
	}

	password := r.Form.Get("password")
	if password == "" {
		http.Error(w, "error, password empty", http.StatusBadRequest)
		return
	}

	if !CheckPasswordHash(password, handler.admin.PasswordHash) {
		log.Tracef("[password] failed login attempt for user: %s", username)
		log.Println(handler.admin)
		http.Error(w, "error, wrong credentials", http.StatusBadRequest)
		return
	}

	if username != handler.admin.Username {
		log.Tracef("[username] failed login attempt for user: %s", username)
		log.Println(handler.admin)
		http.Error(w, "error, wrong credentials", http.StatusBadRequest)
		return
	}

	token, err := handler.authService.Login(time.Now())
	if err != nil {
		log.Errorf("login failed, generate token error: %s", err)
		http.Error(w, "generate token error", http.StatusInternalServerError)
		return
	}

	// token should probably not be logged, but whatta hell
	log.Tracef("new login, token: %s", token)

	WriteResponse(w, "", fmt.Sprintf(`{"token": "%s"}`, token))
}

func (handler *MiscHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.WriteHeader(http.StatusOK)
		return
	}

	authToken := r.Header.Get("X-SERJ-TOKEN")
	if authToken == "" {
		http.Error(w, "no can do", http.StatusUnauthorized)
		return
	}

	loggedOut, err := handler.authService.Logout(authToken)
	if err != nil {
		log.Tracef("[failed login check] => %s: %s", r.URL.Path, err)
		http.Error(w, "no can do", http.StatusUnauthorized)
		return
	}
	if !loggedOut {
		http.Error(w, "no can do", http.StatusUnauthorized)
		return
	}

	log.Printf("logout for [%s] success", authToken)

	WriteResponse(w, "", "logged-out")
}

func (handler *MiscHandler) handleGetVersionInfo(w http.ResponseWriter, r *http.Request) {
	WriteResponse(w, "", handler.versionInfo)
}
