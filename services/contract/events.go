package contract

import (
	"encoding/json"
	"time"
)

const (
	StreamName          = "NOTIFICATIONS"
	SubjectConfirmation = "notifications.confirmation"
	SubjectRelease      = "notifications.release"
	SubjectAll          = "notifications.>"

	StreamDLQ   = "NOTIFICATIONS_DLQ"
	SubjectDead = "dlq.notifications"
)

type ConfirmationRequested struct {
	Email        string `json:"email"`
	RepoName     string `json:"repo_name"`
	ConfirmToken string `json:"confirm_token"`
}

type ReleaseDetected struct {
	Email            string `json:"email"`
	UnsubscribeToken string `json:"unsubscribe_token"`
	ReleaseTag       string `json:"release_tag"`
	ReleaseName      string `json:"release_name"`
	ReleaseURL       string `json:"release_url"`
}

type DeadLetter struct {
	OriginalSubject string          `json:"original_subject"`
	Payload         json.RawMessage `json:"payload"`
	Reason          string          `json:"reason"`
	Attempts        uint64          `json:"attempts"`
	FailedAt        time.Time       `json:"failed_at"`
}
