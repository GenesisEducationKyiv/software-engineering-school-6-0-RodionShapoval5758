module GithubReleaseNotificationAPI/services/notification

go 1.26

require (
	GithubReleaseNotificationAPI/contract v0.0.0
	github.com/joho/godotenv v1.5.1
	github.com/nats-io/nats.go v1.52.0
)

require (
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/nats-io/nkeys v0.4.15 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
)

replace GithubReleaseNotificationAPI/contract => ../contract
