package contract

type ConfirmationRequested struct {
	Email        string `json:"email"`
	RepoName     string `json:"repo_name"`
	ConfirmToken string `json:"confirm_token"`
}

type ReleasePublished struct {
	Email            string `json:"email"`
	UnsubscribeToken string `json:"unsubscribe_token"`
	ReleaseTag       string `json:"release_tag"`
	ReleaseName      string `json:"release_name"`
	ReleaseURL       string `json:"release_url"`
}
