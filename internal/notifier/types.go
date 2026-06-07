package notifier

type ReleaseRecipient struct {
	Email            string
	UnsubscribeToken string
}

type ReleaseInfo struct {
	Tag  string
	Name string
	URL  string
}
