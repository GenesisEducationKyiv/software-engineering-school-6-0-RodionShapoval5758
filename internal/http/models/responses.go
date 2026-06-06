package models

import "GithubReleaseNotificationAPI/internal/domain"

type SubscriptionResponse struct {
	Email       string `json:"email"`
	Repo        string `json:"repo"`
	Confirmed   bool   `json:"confirmed"`
	LastSeenTag string `json:"last_seen_tag"`
}

func ConvertToResponseModel(details []domain.SubscriptionDetails) []SubscriptionResponse {
	responses := make([]SubscriptionResponse, 0, len(details))
	for _, detail := range details {
		lastSeenTag := detail.LastSeenTag
		if lastSeenTag == "" {
			lastSeenTag = "not available yet"
		}

		response := SubscriptionResponse{
			Email:       detail.Email,
			Repo:        detail.Repo,
			Confirmed:   detail.Confirmed,
			LastSeenTag: lastSeenTag,
		}

		responses = append(responses, response)
	}

	return responses
}
