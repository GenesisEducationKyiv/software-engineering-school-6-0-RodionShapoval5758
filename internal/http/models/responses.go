package models

import "GithubReleaseNotificationAPI/internal/store/subscription"

type SubscriptionResponse struct {
	Email       string `json:"email"`
	Repo        string `json:"repo"`
	Confirmed   bool   `json:"confirmed"`
	LastSeenTag string `json:"last_seen_tag"`
}

func ConvertToResponseModel(details []subscription.Details) []SubscriptionResponse {
	responses := make([]SubscriptionResponse, 0, len(details))
	for _, detail := range details {
		lastSeenTag := "not available yet"
		if detail.LastSeenTag != nil {
			lastSeenTag = *detail.LastSeenTag
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
