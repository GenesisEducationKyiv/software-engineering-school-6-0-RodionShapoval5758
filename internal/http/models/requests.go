package models

type SubscriptionRequest struct {
	Email string `json:"email"`
	Repo  string `json:"repo"`
}
