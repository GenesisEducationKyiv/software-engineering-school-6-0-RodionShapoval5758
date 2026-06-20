package usecase_test

import (
	"context"
	"errors"

	"GithubReleaseNotificationAPI/services/subscription/internal/subscription"

	"GithubReleaseNotificationAPI/services/subscription/internal/db"
	githubclient "GithubReleaseNotificationAPI/services/subscription/internal/github"

	"github.com/stretchr/testify/mock"
)

func validSubMatcher(email string, repoID int64) any {
	return mock.MatchedBy(func(sub subscription.Subscription) bool {
		return sub.Email == email &&
			sub.RepositoryID == repoID &&
			sub.ConfirmToken != "" &&
			sub.UnsubscribeToken != ""
	})
}

func (s *UseCaseSuite) TestSubscribe_InvalidInput() {
	cases := []struct {
		name    string
		email   string
		repo    string
		wantErr error
	}{
		{"empty email", "", "owner/repo", subscription.ErrInvalidEmailFormat},
		{"malformed email", "not-an-email", "owner/repo", subscription.ErrInvalidEmailFormat},
		{"no slash in repo", "user@example.com", "noslash", subscription.ErrInvalidRepoFormat},
		{"too many slashes", "user@example.com", "a/b/c", subscription.ErrInvalidRepoFormat},
		{"empty owner", "user@example.com", "/repo", subscription.ErrInvalidRepoFormat},
		{"empty repo name", "user@example.com", "owner/", subscription.ErrInvalidRepoFormat},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest()

			err := s.subscribe.Execute(context.Background(), tc.email, tc.repo)

			s.ErrorIs(err, tc.wantErr)
			s.assertExpectations()
		})
	}
}

func (s *UseCaseSuite) TestSubscribe_NormalizesInput() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.catalog.On("Ensure", mock.Anything, "owner/repo").Return(1, nil)
	s.repo.On("CreateInTx", mock.Anything, validSubMatcher("user@example.com", int64(1))).
		Return(nil)
	s.outbox.On("Insert", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := s.subscribe.Execute(context.Background(), "  user@example.com  ", "  owner/repo  ")

	s.NoError(err)
	s.assertExpectations()
}

func (s *UseCaseSuite) TestSubscribe_GithubRepoNotFound() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(db.ErrNotFound)

	err := s.subscribe.Execute(context.Background(), "user@example.com", "owner/repo")

	s.ErrorIs(err, subscription.ErrRepoNotFound)
	s.assertExpectations()
}

func (s *UseCaseSuite) TestSubscribe_GithubRateLimited() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(githubclient.ErrRateLimited)

	err := s.subscribe.Execute(context.Background(), "user@example.com", "owner/repo")

	s.ErrorIs(err, subscription.ErrTooMuchRequests)
	s.assertExpectations()
}

func (s *UseCaseSuite) TestSubscribe_GithubUnauthorized() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(githubclient.ErrUnauthorized)

	err := s.subscribe.Execute(context.Background(), "user@example.com", "owner/repo")

	s.ErrorIs(err, subscription.ErrGitHubUnauthorized)
	s.assertExpectations()
}

func (s *UseCaseSuite) TestSubscribe_GithubUnknownError() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(errors.New("network timeout"))

	err := s.subscribe.Execute(context.Background(), "user@example.com", "owner/repo")

	s.Error(err)
	s.assertExpectations()
}

func (s *UseCaseSuite) TestSubscribe_CatalogEnsureError() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.catalog.On("Ensure", mock.Anything, "owner/repo").Return(0, errors.New("db error"))

	err := s.subscribe.Execute(context.Background(), "user@example.com", "owner/repo")

	s.Error(err)
	s.assertExpectations()
}

func (s *UseCaseSuite) TestSubscribe_SubscriptionAlreadyExists() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.catalog.On("Ensure", mock.Anything, "owner/repo").Return(1, nil)
	s.repo.On("CreateInTx", mock.Anything, validSubMatcher("user@example.com", int64(1))).
		Return(db.ErrAlreadyExists)

	err := s.subscribe.Execute(context.Background(), "user@example.com", "owner/repo")

	s.ErrorIs(err, subscription.ErrSubscriptionAlreadyExists)
	s.assertExpectations()
}

func (s *UseCaseSuite) TestSubscribe_TokenCollisionRetry() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.catalog.On("Ensure", mock.Anything, "owner/repo").Return(1, nil)

	matcher := validSubMatcher("user@example.com", int64(1))

	s.repo.On("CreateInTx", mock.Anything, matcher).Return(db.ErrTokenConflict).Once()
	s.repo.On("CreateInTx", mock.Anything, matcher).Return(nil).Once()
	s.outbox.On("Insert", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := s.subscribe.Execute(context.Background(), "user@example.com", "owner/repo")

	s.NoError(err)
	s.assertExpectations()
}

func (s *UseCaseSuite) TestSubscribe_TokenCollisionExhausted() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.catalog.On("Ensure", mock.Anything, "owner/repo").Return(1, nil)
	s.repo.On("CreateInTx", mock.Anything, validSubMatcher("user@example.com", int64(1))).
		Return(db.ErrTokenConflict).Times(5)

	err := s.subscribe.Execute(context.Background(), "user@example.com", "owner/repo")

	s.Error(err)
	s.assertExpectations()
}

func (s *UseCaseSuite) TestSubscribe_SubscriptionDBError() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.catalog.On("Ensure", mock.Anything, "owner/repo").Return(1, nil)
	s.repo.On("CreateInTx", mock.Anything, validSubMatcher("user@example.com", int64(1))).
		Return(errors.New("db error"))

	err := s.subscribe.Execute(context.Background(), "user@example.com", "owner/repo")

	s.Error(err)
	s.assertExpectations()
}

func (s *UseCaseSuite) TestSubscribe_OutboxEnqueueFails() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.catalog.On("Ensure", mock.Anything, "owner/repo").Return(1, nil)
	s.repo.On("CreateInTx", mock.Anything, validSubMatcher("user@example.com", int64(1))).
		Return(nil)
	s.outbox.On("Insert", mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("broker down"))

	err := s.subscribe.Execute(context.Background(), "user@example.com", "owner/repo")

	s.Error(err)
	s.assertExpectations()
}
