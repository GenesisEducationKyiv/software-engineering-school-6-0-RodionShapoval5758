package mail

import "GithubReleaseNotificationAPI/internal/domain"

type Service interface {
	SendConfirmationEmail(toEmail, repoName, confirmToken string) error
	SendReleaseNotification(toEmail string, unsubscribeToken string, release *domain.Release) error
}
