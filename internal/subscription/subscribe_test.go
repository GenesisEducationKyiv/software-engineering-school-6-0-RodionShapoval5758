package subscription_test

import (
	"context"
	"errors"

	"GithubReleaseNotificationAPI/internal/subscription"

	githubclient "GithubReleaseNotificationAPI/internal/github"
	"GithubReleaseNotificationAPI/internal/shared"

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

func (s *ServiceTestSuite) TestSubscribe_InvalidInput() {
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

			err := s.svc.Subscribe(context.Background(), tc.email, tc.repo)

			s.ErrorIs(err, tc.wantErr)
			s.assertExpectations()
		})
	}
}

func (s *ServiceTestSuite) TestSubscribe_NormalizesInput() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.catalog.On("Ensure", mock.Anything, "owner/repo").Return(1, nil)

	var capturedToken string
	s.subRepo.On("Create", mock.Anything, validSubMatcher("user@example.com", int64(1))).
		Run(func(args mock.Arguments) {
			capturedToken = args.Get(1).(subscription.Subscription).ConfirmToken
		}).
		Return(nil)
	s.smtp.On("SendConfirmation", "user@example.com", "owner/repo", mock.MatchedBy(func(token string) bool {
		return token != "" && token == capturedToken
	})).Return(nil)

	err := s.svc.Subscribe(context.Background(), "  user@example.com  ", "  owner/repo  ")

	s.NoError(err)
	s.assertExpectations()
}

func (s *ServiceTestSuite) TestSubscribe_GithubRepoNotFound() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(shared.ErrNotFound)

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.ErrorIs(err, subscription.ErrRepoNotFound)
	s.assertExpectations()
}

func (s *ServiceTestSuite) TestSubscribe_GithubRateLimited() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(githubclient.ErrRateLimited)

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.ErrorIs(err, subscription.ErrTooMuchRequests)
	s.assertExpectations()
}

func (s *ServiceTestSuite) TestSubscribe_GithubUnauthorized() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(githubclient.ErrUnauthorized)

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.ErrorIs(err, subscription.ErrGitHubUnauthorized)
	s.assertExpectations()
}

func (s *ServiceTestSuite) TestSubscribe_GithubUnknownError() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(errors.New("network timeout"))

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.Error(err)
	s.assertExpectations()
}

func (s *ServiceTestSuite) TestSubscribe_CatalogEnsureError() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.catalog.On("Ensure", mock.Anything, "owner/repo").Return(0, errors.New("db error"))

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.Error(err)
	s.assertExpectations()
}

func (s *ServiceTestSuite) TestSubscribe_SubscriptionAlreadyExists() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.catalog.On("Ensure", mock.Anything, "owner/repo").Return(1, nil)
	s.subRepo.On("Create", mock.Anything, validSubMatcher("user@example.com", int64(1))).
		Return(shared.ErrAlreadyExists)

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.ErrorIs(err, subscription.ErrSubscriptionAlreadyExists)
	s.assertExpectations()
}

func (s *ServiceTestSuite) TestSubscribe_TokenCollisionRetry() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.catalog.On("Ensure", mock.Anything, "owner/repo").Return(1, nil)

	matcher := validSubMatcher("user@example.com", int64(1))

	var capturedToken string
	s.subRepo.On("Create", mock.Anything, matcher).Return(shared.ErrTokenConflict).Once()
	s.subRepo.On("Create", mock.Anything, matcher).
		Run(func(args mock.Arguments) {
			capturedToken = args.Get(1).(subscription.Subscription).ConfirmToken
		}).
		Return(nil).Once()
	s.smtp.On("SendConfirmation", "user@example.com", "owner/repo", mock.MatchedBy(func(token string) bool {
		return token != "" && token == capturedToken
	})).Return(nil)

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.NoError(err)
	s.assertExpectations()
}

func (s *ServiceTestSuite) TestSubscribe_TokenCollisionExhausted() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.catalog.On("Ensure", mock.Anything, "owner/repo").Return(1, nil)
	s.subRepo.On("Create", mock.Anything, validSubMatcher("user@example.com", int64(1))).
		Return(shared.ErrTokenConflict).Times(5)

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.Error(err)
	s.assertExpectations()
}

func (s *ServiceTestSuite) TestSubscribe_SubscriptionDBError() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.catalog.On("Ensure", mock.Anything, "owner/repo").Return(1, nil)
	s.subRepo.On("Create", mock.Anything, validSubMatcher("user@example.com", int64(1))).
		Return(errors.New("db error"))

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.Error(err)
	s.assertExpectations()
}

func (s *ServiceTestSuite) TestSubscribe_EmailSendFails() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.catalog.On("Ensure", mock.Anything, "owner/repo").Return(1, nil)

	var capturedToken string
	s.subRepo.On("Create", mock.Anything, validSubMatcher("user@example.com", int64(1))).
		Run(func(args mock.Arguments) {
			capturedToken = args.Get(1).(subscription.Subscription).ConfirmToken
		}).
		Return(nil)
	s.smtp.On("SendConfirmation", "user@example.com", "owner/repo", mock.MatchedBy(func(token string) bool {
		return token != "" && token == capturedToken
	})).Return(errors.New("smtp error"))

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.Error(err)
	s.assertExpectations()
}
