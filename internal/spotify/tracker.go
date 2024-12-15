package spotify

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"

	log "github.com/sirupsen/logrus"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

type Tracker struct {
	auth               *spotifyauth.Authenticator
	client             *spotify.Client
	randStateGenerator func() string
	state              string
}

// https://developer.spotify.com/documentation/web-api/reference/get-recently-played

func NewTracker(
	redirectURI string,
	spotifyClientID string,
	spotifyClientSecret string,
	randStateGenerator func() string,
) *Tracker {
	return &Tracker{
		auth: spotifyauth.New(
			spotifyauth.WithRedirectURL(redirectURI),
			spotifyauth.WithScopes(spotifyauth.ScopeUserReadRecentlyPlayed),
			spotifyauth.WithClientID(spotifyClientID),
			spotifyauth.WithClientSecret(spotifyClientSecret),
		),
		randStateGenerator: randStateGenerator,
	}
}

func GenerateStateString() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func (t *Tracker) Authenticate(w http.ResponseWriter, r *http.Request) {
	_, span := tracing.GlobalTracer.Start(r.Context(), "spotify.tracker.authenticate")
	defer span.End()

	t.state = t.randStateGenerator()
	redirectURL := t.auth.AuthURL(t.state)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (t *Tracker) AuthRedirect(w http.ResponseWriter, r *http.Request) {
	var err error
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "spotify.tracker.authRedirect")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	tok, err := t.auth.Token(ctx, t.state, r)
	if err != nil {
		http.Error(w, "failed to get get token", http.StatusForbidden)
		log.Errorf("failed to get token: %v", err)
		return
	}
	if st := r.FormValue("state"); st != t.state {
		http.Error(w, "state mismatch", http.StatusForbidden)
		log.Fatalf("state mismatch: %s != %s\n", st, t.state)
	}

	// redirect to the main page
	http.Redirect(w, r, "/", http.StatusFound)

	// let the request finish, and we set the spotify client in a new goroutine
	go func() {
		var err error
		innerCtx, innerSpan := tracing.GlobalTracer.Start(
			context.WithoutCancel(ctx),
			"spotify.tracker.authRedirect.setClient",
		)
		defer func() {
			tracing.EndSpanWithErrCheck(innerSpan, err)
		}()

		// use the token to get an authenticated client
		t.client = spotify.New(t.auth.Client(innerCtx, tok))

		u, err := t.client.CurrentUser(innerCtx)
		if err != nil {
			log.Errorf("failed to get current user: %s", err)
		} else {
			log.Debugf("authenticated as: %s\n", u.DisplayName)
		}
	}()
}

func (t *Tracker) GetRecentlyPlayed(w http.ResponseWriter, r *http.Request) {
	var err error
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "spotify.tracker.getRecentlyPlayed")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	if t.client == nil {
		log.Debugln("get latest songs - spotify client is nil, redirecting to authenticate")
		// redirect the request to authenticate
		t.Authenticate(w, r.WithContext(ctx))
	}

	// check if the client is still unauthenticated / nil, then return error
	if t.client == nil {
		http.Error(w, "failed to authenticate", http.StatusForbidden)
		return
	}

	// get the latest played songs
	plays, err := t.client.PlayerRecentlyPlayed(ctx)
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
