package spotify

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

type Handler struct {
	db                  *pgxpool.Pool
	repo                *Repo
	auth                *spotifyauth.Authenticator
	client              *spotify.Client
	tracker             *Tracker
	fireIntervalMinutes int
	randStateGenerator  func() (string, error)
	stateToken          string
	// authRedirectURL is the URL to redirect to after successful authentication with Spotify, not the URL to authenticate
	// e.g. after successful authentication, redirect to the main page (www.serj-tubin.com/spotify)
	// just check the config.toml to see the actual values and make it clear.
	authRedirectURL  string
	authRequestToken string
}

// https://developer.spotify.com/documentation/web-api/reference/get-recently-played

func NewHandler(
	db *pgxpool.Pool,
	repo *Repo,
	redirectURL string, // spotify will invoke this URL after successful/unsuccessful authentication
	authRedirectURL string,
	spotifyClientID string,
	spotifyClientSecret string,
	randStateGenerator func() (string, error),
	fireIntervalMinutes int,
	authRequestToken string,
) *Handler {
	return &Handler{
		db:                  db,
		repo:                repo,
		randStateGenerator:  randStateGenerator,
		tracker:             nil,
		fireIntervalMinutes: fireIntervalMinutes,
		authRedirectURL:     authRedirectURL,
		authRequestToken:    authRequestToken,
		auth: spotifyauth.New(
			spotifyauth.WithRedirectURL(redirectURL),
			spotifyauth.WithScopes(spotifyauth.ScopeUserReadRecentlyPlayed),
			spotifyauth.WithClientID(spotifyClientID),
			spotifyauth.WithClientSecret(spotifyClientSecret),
		),
	}
}

func GenerateStateString() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand read: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (h *Handler) Authenticate(w http.ResponseWriter, r *http.Request) {
	var err error
	_, span := tracing.GlobalTracer.Start(r.Context(), "spotify.handler.authenticate")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	// get the token from the url params
	token := r.URL.Query().Get("token")
	if token != h.authRequestToken {
		http.Error(w, "invalid token", http.StatusForbidden)
		log.Errorf("invalid token: %s", token)
		return
	}

	h.stateToken, err = h.randStateGenerator()
	if err != nil {
		http.Error(w, "failed to generate state token", http.StatusInternalServerError)
		log.Errorf("failed to generate state token: %v", err)
		return
	}
	redirectURL := h.auth.AuthURL(h.stateToken)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *Handler) AuthRedirect(w http.ResponseWriter, r *http.Request) {
	var err error
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "spotify.handler.authRedirect")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	tok, err := h.auth.Token(ctx, h.stateToken, r)
	if err != nil {
		http.Error(w, "failed to get get token", http.StatusForbidden)
		log.Errorf("failed to get token: %v", err)
		return
	}
	if st := r.FormValue("state"); st != h.stateToken {
		http.Error(w, "state mismatch", http.StatusForbidden)
		log.Fatalf("state mismatch: %s != %s\n", st, h.stateToken)
	}

	// redirect to the main page
	http.Redirect(w, r, h.authRedirectURL, http.StatusFound)

	// let the request finish, and we set the spotify client in a new goroutine
	go func() {
		var err error
		innerCtx, innerSpan := tracing.GlobalTracer.Start(
			context.WithoutCancel(ctx),
			"spotify.handler.authRedirect.setClient",
		)
		defer func() {
			tracing.EndSpanWithErrCheck(innerSpan, err)
		}()

		// use the token to get an authenticated client
		h.client = spotify.New(h.auth.Client(innerCtx, tok))

		u, err := h.client.CurrentUser(innerCtx)
		if err != nil {
			log.Errorf("failed to get current user: %s", err)
		} else {
			log.Debugf("authenticated as: %s\n", u.DisplayName)
		}

		if h.tracker != nil {
			h.tracker.Stop()
		}
		h.tracker = NewTracker(NewRepo(h.db), h.client, h.fireIntervalMinutes)

		// start the tracker
		h.tracker.Start()
		log.Debugln("spotify tracker started")
	}()
}

type TrackerStatusResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func (h *Handler) GetTrackerStatus(w http.ResponseWriter, r *http.Request) {
	_, span := tracing.GlobalTracer.Start(r.Context(), "spotify.handler.getTrackerStatus")
	defer span.End()

	if h.tracker == nil {
		pkg.SendJsonResponse(w, http.StatusOK, TrackerStatusResponse{Status: "stopped"})
		return
	}
	status := h.tracker.Status()
	pkg.SendJsonResponse(w, http.StatusOK, TrackerStatusResponse{Status: status})
}

func (h *Handler) StartTracker(w http.ResponseWriter, r *http.Request) {
	_, span := tracing.GlobalTracer.Start(r.Context(), "spotify.handler.startTracker")
	defer span.End()

	if h.tracker == nil {
		respMsg := TrackerStatusResponse{Status: "stopped", Message: "tracker not initialized"}
		pkg.SendJsonResponse(w, http.StatusBadRequest, respMsg)
		return
	}
	h.tracker.Start()
	pkg.SendJsonResponse(w, http.StatusOK, TrackerStatusResponse{Status: "started"})
}

func (h *Handler) StopTracker(w http.ResponseWriter, r *http.Request) {
	_, span := tracing.GlobalTracer.Start(r.Context(), "spotify.handler.stopTracker")
	defer span.End()

	if h.tracker == nil {
		respMsg := TrackerStatusResponse{Status: "stopped", Message: "tracker not initialized"}
		pkg.SendJsonResponse(w, http.StatusBadRequest, respMsg)
		return
	}
	h.tracker.Stop()
	pkg.SendJsonResponse(w, http.StatusOK, TrackerStatusResponse{Status: "stopped"})
}

func (h *Handler) GetPage(w http.ResponseWriter, r *http.Request) {
	_, span := tracing.GlobalTracer.Start(r.Context(), "spotify.handler.getPage")
	defer span.End()

	if h.client == nil {
		log.Debugln("get page - spotify client is nil, redirecting to authenticate")
		// redirect the request to authenticate
		h.Authenticate(w, r)
	}

	// check if the client is still unauthenticated / nil, then return error
	if h.client == nil {
		http.Error(w, "failed to authenticate", http.StatusForbidden)
		return
	}

	pageStr := r.URL.Query().Get("page")
	sizeStr := r.URL.Query().Get("size")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		http.Error(w, "invalid page", http.StatusBadRequest)
		return
	}
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		http.Error(w, "invalid size", http.StatusBadRequest)
		return
	}

	tracks, err := h.repo.GetPage(r.Context(), page, size)
	if err != nil {
		log.Warnf("failed to get page [%d, %d]: %v", page, size, err)
		http.Error(w, "failed to get page", http.StatusInternalServerError)
		return
	}

	pkg.SendJsonResponse(w, http.StatusOK, tracks)
}

func (h *Handler) GetRecentlyPlayed(w http.ResponseWriter, r *http.Request) {
	var err error
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "spotify.handler.getRecentlyPlayed")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	if h.client == nil {
		log.Debugln("get latest songs - spotify client is nil, redirecting to authenticate")
		// redirect the request to authenticate
		h.Authenticate(w, r.WithContext(ctx))
	}

	// check if the client is still unauthenticated / nil, then return error
	if h.client == nil {
		http.Error(w, "failed to authenticate", http.StatusForbidden)
		return
	}

	// get the latest played songs
	plays, err := h.client.PlayerRecentlyPlayed(ctx)
	if err != nil {
		log.Errorf("failed to get recently played songs: %v", err)
		http.Error(w, "failed to get recently played songs", http.StatusInternalServerError)
		return
	}

	// return the latest played songs
	playsJson, err := json.Marshal(plays)
	if err != nil {
		log.Errorf("failed to marshal plays to json: %v", err)
		http.Error(w, "failed to marshal plays to json", http.StatusInternalServerError)
		return
	}

	pkg.WriteJSONResponseOK(w, string(playsJson))
}
