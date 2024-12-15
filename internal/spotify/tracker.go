package spotify

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"

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

	w.WriteHeader(http.StatusNoContent)

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
