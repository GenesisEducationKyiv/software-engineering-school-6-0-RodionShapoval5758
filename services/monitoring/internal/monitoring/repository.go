package monitoring

// noReleaseYet is the cursor for a repo that had no release when first scanned,
// so its first release is not mistaken for the baseline. It contains a space,
// which a git tag name cannot, so it never collides with a real tag.
const noReleaseYet = "(no release)"

type TrackedRepo struct {
	ID          int64
	FullName    string
	LastSeenTag string
}

func (r TrackedRepo) HasNewRelease(tag string) bool {
	return r.LastSeenTag != "" && tag != r.LastSeenTag
}

type DetectedRelease struct {
	RepoID      int64
	RepoName    string
	ReleaseTag  string
	ReleaseName string
	ReleaseURL  string
}
