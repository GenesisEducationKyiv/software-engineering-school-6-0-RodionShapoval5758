package contract

import (
	"encoding/json"
	"time"
)

const (
	StreamName          = "NOTIFICATIONS"
	SubjectConfirmation = "notifications.confirmation"
	SubjectRelease      = "notifications.release"
	SubjectReleaseFound = "notifications.release_found"
	SubjectAll          = "notifications.>"

	StreamDLQ   = "NOTIFICATIONS_DLQ"
	SubjectDead = "dlq.notifications"

	StreamTracking       = "TRACKING"
	SubjectRepoTracked   = "tracking.repo.tracked"
	SubjectRepoUntracked = "tracking.repo.untracked"
	SubjectTrackingAll   = "tracking.>"

	StreamSaga         = "SAGA"
	SubjectEmailSent   = "saga.email.sent"
	SubjectEmailFailed = "saga.email.failed"
	SubjectSagaAll     = "saga.>"
)

type ConfirmationRequested struct {
	SagaID       string `json:"saga_id"`
	Email        string `json:"email"`
	RepoName     string `json:"repo_name"`
	ConfirmToken string `json:"confirm_token"`
}

type EmailSent struct {
	SagaID string `json:"saga_id"`
}

type EmailFailed struct {
	SagaID string `json:"saga_id"`
	Reason string `json:"reason"`
}

type ReleaseDetected struct {
	Email            string `json:"email"`
	UnsubscribeToken string `json:"unsubscribe_token"`
	ReleaseTag       string `json:"release_tag"`
	ReleaseName      string `json:"release_name"`
	ReleaseURL       string `json:"release_url"`
}

type RepoTracked struct {
	RepoID   int64  `json:"repo_id"`
	FullName string `json:"full_name"`
}

type RepoUntracked struct {
	RepoID int64 `json:"repo_id"`
}

type ReleaseFound struct {
	RepoID      int64  `json:"repo_id"`
	RepoName    string `json:"repo_name"`
	ReleaseTag  string `json:"release_tag"`
	ReleaseName string `json:"release_name"`
	ReleaseURL  string `json:"release_url"`
}

type DeadLetter struct {
	OriginalSubject string          `json:"original_subject"`
	Payload         json.RawMessage `json:"payload"`
	Reason          string          `json:"reason"`
	Attempts        uint64          `json:"attempts"`
	FailedAt        time.Time       `json:"failed_at"`
}
