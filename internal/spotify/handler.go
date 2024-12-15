package spotify

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

type Handler struct {
	auth               *spotifyauth.Authenticator
	client             *spotify.Client
	tracker            *Tracker
	db                 *pgxpool.Pool
	randStateGenerator func() string
	state              string
}

// https://developer.spotify.com/documentation/web-api/reference/get-recently-played

func NewHandler(
	redirectURI string,
	spotifyClientID string,
	spotifyClientSecret string,
	randStateGenerator func() string,
	db *pgxpool.Pool,
) *Handler {
	return &Handler{
		auth: spotifyauth.New(
			spotifyauth.WithRedirectURL(redirectURI),
			spotifyauth.WithScopes(spotifyauth.ScopeUserReadRecentlyPlayed),
			spotifyauth.WithClientID(spotifyClientID),
			spotifyauth.WithClientSecret(spotifyClientSecret),
		),
		randStateGenerator: randStateGenerator,
		tracker:            nil,
		db:                 db,
	}
}

func GenerateStateString() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func (h *Handler) Authenticate(w http.ResponseWriter, r *http.Request) {
	_, span := tracing.GlobalTracer.Start(r.Context(), "spotify.handler.authenticate")
	defer span.End()

	h.state = h.randStateGenerator()
	redirectURL := h.auth.AuthURL(h.state)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *Handler) AuthRedirect(w http.ResponseWriter, r *http.Request) {
	var err error
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "spotify.handler.authRedirect")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	tok, err := h.auth.Token(ctx, h.state, r)
	if err != nil {
		http.Error(w, "failed to get get token", http.StatusForbidden)
		log.Errorf("failed to get token: %v", err)
		return
	}
	if st := r.FormValue("state"); st != h.state {
		http.Error(w, "state mismatch", http.StatusForbidden)
		log.Fatalf("state mismatch: %s != %s\n", st, h.state)
	}

	// redirect to the main page
	http.Redirect(w, r, "/", http.StatusFound)

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
		h.tracker = NewTracker(NewRepo(h.db), h.client)
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
