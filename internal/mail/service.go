package mail

import "GithubReleaseNotificationAPI/internal/github"

type Service interface {
	SendConfirmationEmail(toEmail, repoName, confirmToken string) error
	SendReleaseNotification(toEmail string, unsubscribeToken string, release *github.Release) error
}
