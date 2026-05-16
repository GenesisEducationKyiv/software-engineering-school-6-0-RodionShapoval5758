package service_test

import (
	"context"
	"errors"
	"testing"

	"GithubReleaseNotificationAPI/internal/domain"
	gh "GithubReleaseNotificationAPI/internal/github"
	"GithubReleaseNotificationAPI/internal/service"
	"GithubReleaseNotificationAPI/internal/store"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type SubscriptionServiceTestSuite struct {
	suite.Suite

	subRepo  *mockSubscriptionRepository
	repoRepo *mockRepositoryRepository
	github   *mockGithubClient
	smtp     *mockSmtpClient

	svc service.SubscriptionService
}

func (s *SubscriptionServiceTestSuite) SetupTest() {
	s.subRepo = new(mockSubscriptionRepository)
	s.repoRepo = new(mockRepositoryRepository)
	s.github = new(mockGithubClient)
	s.smtp = new(mockSmtpClient)

	s.svc = service.NewSubscriptionService(s.subRepo, s.repoRepo, s.github, s.smtp)
}

func (s *SubscriptionServiceTestSuite) assertExpectations() {
	s.subRepo.AssertExpectations(s.T())
	s.repoRepo.AssertExpectations(s.T())
	s.github.AssertExpectations(s.T())
	s.smtp.AssertExpectations(s.T())
}

func TestSubscriptionServiceTestSuite(t *testing.T) {
	suite.Run(t, new(SubscriptionServiceTestSuite))
}

// --- Subscribe ---

func (s *SubscriptionServiceTestSuite) TestSubscribe_InvalidInput() {
	cases := []struct {
		name    string
		email   string
		repo    string
		wantErr error
	}{
		{"empty email", "", "owner/repo", service.ErrInvalidEmailFormat},
		{"malformed email", "not-an-email", "owner/repo", service.ErrInvalidEmailFormat},
		{"no slash in repo", "user@example.com", "noslash", service.ErrInvalidRepoFormat},
		{"too many slashes", "user@example.com", "a/b/c", service.ErrInvalidRepoFormat},
		{"empty owner", "user@example.com", "/repo", service.ErrInvalidRepoFormat},
		{"empty repo name", "user@example.com", "owner/", service.ErrInvalidRepoFormat},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			err := s.svc.Subscribe(context.Background(), tc.email, tc.repo)

			s.ErrorIs(err, tc.wantErr)
			s.assertExpectations()
		})
	}
}

func (s *SubscriptionServiceTestSuite) TestSubscribe_GithubRepoNotFound() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(gh.ErrNotFound)

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.ErrorIs(err, service.ErrRepoNotFound)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestSubscribe_GithubRateLimited() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(gh.ErrRateLimited)

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.ErrorIs(err, service.ErrTooMuchRequests)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestSubscribe_GithubUnexpectedResponse() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(gh.ErrUnexpectedResponse)

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.Error(err)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestSubscribe_GithubUnknownError() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(errors.New("network timeout"))

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.Error(err)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestSubscribe_RepoExistsInDB() {
	repo := &domain.Repository{ID: 1, FullName: "owner/repo"}

	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.repoRepo.On("FindByFullName", mock.Anything, "owner/repo").Return(repo, nil)
	s.subRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	s.smtp.On("SendConfirmationEmail", "user@example.com", "owner/repo", mock.Anything).Return(nil)

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.NoError(err)
	s.assertExpectations() // repoRepo.Create has no .On() — if called, test panics
}

func (s *SubscriptionServiceTestSuite) TestSubscribe_RepoNotInDB_Created() {
	repo := &domain.Repository{ID: 1, FullName: "owner/repo"}

	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.repoRepo.On("FindByFullName", mock.Anything, "owner/repo").Return(nil, store.ErrNotFound)
	s.repoRepo.On("Create", mock.Anything, "owner/repo").Return(repo, nil)
	s.subRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	s.smtp.On("SendConfirmationEmail", "user@example.com", "owner/repo", mock.Anything).Return(nil)

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.NoError(err)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestSubscribe_RepoFindDBError() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.repoRepo.On("FindByFullName", mock.Anything, "owner/repo").Return(nil, errors.New("db error"))

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.Error(err)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestSubscribe_RepoCreateRaceRecovery() {
	repo := &domain.Repository{ID: 1, FullName: "owner/repo"}

	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.repoRepo.On("FindByFullName", mock.Anything, "owner/repo").Return(nil, store.ErrNotFound).Once()
	s.repoRepo.On("Create", mock.Anything, "owner/repo").Return(nil, store.ErrAlreadyExists)
	s.repoRepo.On("FindByFullName", mock.Anything, "owner/repo").Return(repo, nil).Once()
	s.subRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	s.smtp.On("SendConfirmationEmail", "user@example.com", "owner/repo", mock.Anything).Return(nil)

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.NoError(err)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestSubscribe_RepoCreateRaceRecoveryFails() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.repoRepo.On("FindByFullName", mock.Anything, "owner/repo").Return(nil, store.ErrNotFound).Once()
	s.repoRepo.On("Create", mock.Anything, "owner/repo").Return(nil, store.ErrAlreadyExists)
	s.repoRepo.On("FindByFullName", mock.Anything, "owner/repo").Return(nil, errors.New("db error")).Once()

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.Error(err)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestSubscribe_RepoCreateDBError() {
	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.repoRepo.On("FindByFullName", mock.Anything, "owner/repo").Return(nil, store.ErrNotFound)
	s.repoRepo.On("Create", mock.Anything, "owner/repo").Return(nil, errors.New("db error"))

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.Error(err)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestSubscribe_SubscriptionAlreadyExists() {
	repo := &domain.Repository{ID: 1, FullName: "owner/repo"}

	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.repoRepo.On("FindByFullName", mock.Anything, "owner/repo").Return(repo, nil)
	s.subRepo.On("Create", mock.Anything, mock.Anything).Return(store.ErrAlreadyExists)

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.ErrorIs(err, service.ErrSubscriptionAlreadyExists)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestSubscribe_TokenCollisionRetry() {
	repo := &domain.Repository{ID: 1, FullName: "owner/repo"}

	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.repoRepo.On("FindByFullName", mock.Anything, "owner/repo").Return(repo, nil)
	s.subRepo.On("Create", mock.Anything, mock.Anything).Return(store.ErrTokensAlreadyExists).Once()
	s.subRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
	s.smtp.On("SendConfirmationEmail", "user@example.com", "owner/repo", mock.Anything).Return(nil)

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.NoError(err)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestSubscribe_TokenCollisionExhausted() {
	repo := &domain.Repository{ID: 1, FullName: "owner/repo"}

	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.repoRepo.On("FindByFullName", mock.Anything, "owner/repo").Return(repo, nil)
	s.subRepo.On("Create", mock.Anything, mock.Anything).Return(store.ErrTokensAlreadyExists).Times(5)

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.Error(err)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestSubscribe_SubscriptionDBError() {
	repo := &domain.Repository{ID: 1, FullName: "owner/repo"}

	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.repoRepo.On("FindByFullName", mock.Anything, "owner/repo").Return(repo, nil)
	s.subRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error"))

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.Error(err)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestSubscribe_EmailSendFails() {
	repo := &domain.Repository{ID: 1, FullName: "owner/repo"}

	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.repoRepo.On("FindByFullName", mock.Anything, "owner/repo").Return(repo, nil)
	s.subRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	s.smtp.On("SendConfirmationEmail", "user@example.com", "owner/repo", mock.Anything).Return(errors.New("smtp error"))

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.Error(err)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestSubscribe_HappyPath_RepoExistsInDB() {
	repo := &domain.Repository{ID: 1, FullName: "owner/repo"}

	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.repoRepo.On("FindByFullName", mock.Anything, "owner/repo").Return(repo, nil)
	s.subRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	s.smtp.On("SendConfirmationEmail", "user@example.com", "owner/repo", mock.Anything).Return(nil)

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.NoError(err)
	s.assertExpectations()
}

func (s *SubscriptionServiceTestSuite) TestSubscribe_HappyPath_RepoCreated() {
	repo := &domain.Repository{ID: 1, FullName: "owner/repo"}

	s.github.On("CheckRepo", mock.Anything, "owner/repo").Return(nil)
	s.repoRepo.On("FindByFullName", mock.Anything, "owner/repo").Return(nil, store.ErrNotFound)
	s.repoRepo.On("Create", mock.Anything, "owner/repo").Return(repo, nil)
	s.subRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	s.smtp.On("SendConfirmationEmail", "user@example.com", "owner/repo", mock.Anything).Return(nil)

	err := s.svc.Subscribe(context.Background(), "user@example.com", "owner/repo")

	s.NoError(err)
	s.assertExpectations()
}
