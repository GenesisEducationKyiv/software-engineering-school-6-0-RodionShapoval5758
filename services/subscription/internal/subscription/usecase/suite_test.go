package usecase_test

import (
	"testing"

	"GithubReleaseNotificationAPI/services/subscription/internal/subscription/usecase"

	"github.com/stretchr/testify/suite"
)

type UseCaseSuite struct {
	suite.Suite

	repo    *mockRepository
	catalog *mockCatalog
	github  *mockGithub
	outbox  *mockOutbox

	subscribe   *usecase.Subscribe
	confirm     *usecase.Confirm
	unsubscribe *usecase.Unsubscribe
	list        *usecase.List
}

func (s *UseCaseSuite) SetupTest() {
	s.repo = new(mockRepository)
	s.catalog = new(mockCatalog)
	s.github = new(mockGithub)
	s.outbox = new(mockOutbox)

	s.subscribe = usecase.NewSubscribe(s.repo, s.catalog, s.github, s.outbox, &fakeTxBeginner{})
	s.confirm = usecase.NewConfirm(s.repo)
	s.unsubscribe = usecase.NewUnsubscribe(s.repo, s.catalog)
	s.list = usecase.NewList(s.repo)
}

func (s *UseCaseSuite) assertExpectations() {
	s.repo.AssertExpectations(s.T())
	s.catalog.AssertExpectations(s.T())
	s.github.AssertExpectations(s.T())
	s.outbox.AssertExpectations(s.T())
}

func TestUseCaseSuite(t *testing.T) {
	suite.Run(t, new(UseCaseSuite))
}
