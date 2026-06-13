package server

import (
	"encoding/json"
	"net/http"

	"GithubReleaseNotificationAPI/contract"
)

type Mailer interface {
	SendConfirmation(toEmail, repoName, confirmToken string) error
	SendRelease(toEmail, unsubscribeToken string, releaseTag, releaseName, releaseURL string) error
}

type Server struct {
	mailer Mailer
}

func New(m Mailer) http.Handler {
	s := &Server{mailer: m}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/emails/confirmation", s.handleConfirmation)
	mux.HandleFunc("POST /v1/emails/release", s.handleRelease)
	mux.HandleFunc("GET /healthz", s.handleHealth)

	return mux
}

func (s *Server) handleConfirmation(w http.ResponseWriter, r *http.Request) {
	var ev contract.ConfirmationRequested
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if err := s.mailer.SendConfirmation(ev.Email, ev.RepoName, ev.ConfirmToken); err != nil {
		http.Error(w, "failed to send confirmation email", http.StatusBadGateway)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleRelease(w http.ResponseWriter, r *http.Request) {
	var ev contract.ReleasePublished
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if err := s.mailer.SendRelease(ev.Email, ev.UnsubscribeToken, ev.ReleaseTag, ev.ReleaseName, ev.ReleaseURL); err != nil {
		http.Error(w, "failed to send release email", http.StatusBadGateway)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
