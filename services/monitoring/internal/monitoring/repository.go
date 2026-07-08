package monitoring

type TrackedRepo struct {
	ID          int64
	FullName    string
	LastSeenTag string
}

func (r TrackedRepo) HasNewRelease(tag string) bool {
	return r.LastSeenTag == "" || tag != r.LastSeenTag
}

type DetectedRelease struct {
	RepoID      int64
	RepoName    string
	ReleaseTag  string
	ReleaseName string
	ReleaseURL  string
}
