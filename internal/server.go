package internal

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type Server struct {
}

func NewServer() *Server {
	s := &Server{}
	return s
}

func (s *Server) routerSetup() (r *mux.Router) {
	r = mux.NewRouter()

	// server static files
	fs := http.FileServer(http.Dir("./web/"))
	r.PathPrefix("/web/").Handler(http.StripPrefix("/web/", fs))

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/web/index.html", http.StatusPermanentRedirect)
	})

	return r
}

func (s *Server) Serve() {
	router := s.routerSetup()

	ipAndPort := fmt.Sprintf("%s:%s", "localhost", "8080")

	httpServer := &http.Server{
		Handler:      router,
		Addr:         ipAndPort,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Infof(" > server listening on: [%s]", ipAndPort)
	log.Fatal(httpServer.ListenAndServe())
}
