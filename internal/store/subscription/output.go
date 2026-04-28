package subscription

type Details struct {
	Email       string
	Repo        string
	Confirmed   bool
	LastSeenTag *string
}
