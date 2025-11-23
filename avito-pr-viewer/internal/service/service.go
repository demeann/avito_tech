package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/demeann/avito-pr-reviewer/internal/domain"
)

type Repository interface {
	CreateTeamWithMembers(ctx context.Context, name string, members []domain.TeamMember) (domain.Team, error)
	GetTeam(ctx context.Context, name string) (domain.Team, error)

	SetUserActive(ctx context.Context, userID string, isActive bool) (domain.User, error)
	GetUserByID(ctx context.Context, userID string) (domain.User, error)

	CreatePullRequest(ctx context.Context, id, name, authorID string, createdAt time.Time, reviewers []string) (domain.PullRequest, error)
	GetPullRequest(ctx context.Context, id string) (domain.PullRequest, error)
	SetPullRequestMerged(ctx context.Context, id string, mergedAt time.Time) (domain.PullRequest, error)

	GetTeamNameByUserID(ctx context.Context, userID string) (string, error)
	GetActiveTeamMembersExcept(ctx context.Context, teamName string, exclude []string) ([]domain.User, error)

	ReassignReviewer(ctx context.Context, prID, oldUserID, newUserID string) (domain.PullRequest, error)

	GetUserReviewPRs(ctx context.Context, userID string) ([]domain.PullRequestShort, error)

	PRExists(ctx context.Context, id string) (bool, error)
	TeamExists(ctx context.Context, name string) (bool, error)
}

type Service struct {
	repo Repository
}

func NewService(r Repository) *Service {
	rand.Seed(time.Now().UnixNano())
	return &Service{repo: r}
}

func (s *Service) CreateTeam(ctx context.Context, name string, members []domain.TeamMember) (domain.Team, error) {
	exists, err := s.repo.TeamExists(ctx, name)
	if err != nil {
		return domain.Team{}, err
	}
	if exists {
		return domain.Team{}, &AppError{
			Code:       ErrCodeTeamExists,
			HTTPStatus: 400,
			Message:    "team_name already exists",
		}
	}
	return s.repo.CreateTeamWithMembers(ctx, name, members)
}

func (s *Service) GetTeam(ctx context.Context, name string) (domain.Team, error) {
	team, err := s.repo.GetTeam(ctx, name)
	if err != nil {
		return domain.Team{}, err
	}
	if team.Name == "" {
		return domain.Team{}, notFound("team not found")
	}
	return team, nil
}

func (s *Service) SetUserActive(ctx context.Context, userID string, isActive bool) (domain.User, error) {
	user, err := s.repo.SetUserActive(ctx, userID, isActive)
	if err != nil {
		return domain.User{}, err
	}
	if user.UserID == "" {
		return domain.User{}, notFound("user not found")
	}
	return user, nil
}

func (s *Service) CreatePullRequest(ctx context.Context, id, name, authorID string) (domain.PullRequest, error) {
	exists, err := s.repo.PRExists(ctx, id)
	if err != nil {
		return domain.PullRequest{}, err
	}
	if exists {
		return domain.PullRequest{}, &AppError{
			Code:       ErrCodePRExists,
			HTTPStatus: 409,
			Message:    "PR id already exists",
		}
	}

	author, err := s.repo.GetUserByID(ctx, authorID)
	if err != nil {
		return domain.PullRequest{}, err
	}
	if author.UserID == "" {
		return domain.PullRequest{}, notFound("author not found")
	}

	team, err := s.repo.GetTeam(ctx, author.TeamName)
	if err != nil {
		return domain.PullRequest{}, err
	}
	if team.Name == "" {
		return domain.PullRequest{}, notFound("author team not found")
	}

	var candidates []string
	for _, m := range team.Members {
		if m.UserID == authorID {
			continue
		}
		if !m.IsActive {
			continue
		}
		candidates = append(candidates, m.UserID)
	}

	reviewers := chooseRandom(candidates, 2)

	now := time.Now().UTC()
	return s.repo.CreatePullRequest(ctx, id, name, authorID, now, reviewers)
}

func chooseRandom(ids []string, n int) []string {
	if len(ids) == 0 || n <= 0 {
		return nil
	}
	if len(ids) <= n {
		out := make([]string, len(ids))
		copy(out, ids)
		return out
	}
	out := make([]string, 0, n)
	perm := rand.Perm(len(ids))
	for i := 0; i < n; i++ {
		out = append(out, ids[perm[i]])
	}
	return out
}

func (s *Service) MergePullRequest(ctx context.Context, id string) (domain.PullRequest, error) {
	pr, err := s.repo.GetPullRequest(ctx, id)
	if err != nil {
		return domain.PullRequest{}, err
	}
	if pr.ID == "" {
		return domain.PullRequest{}, notFound("pull request not found")
	}

	if pr.Status == domain.StatusMerged {
		return pr, nil
	}

	now := time.Now().UTC()
	return s.repo.SetPullRequestMerged(ctx, id, now)
}

func (s *Service) ReassignReviewer(ctx context.Context, prID, oldUserID string) (domain.PullRequest, string, error) {
	pr, err := s.repo.GetPullRequest(ctx, prID)
	if err != nil {
		return domain.PullRequest{}, "", err
	}
	if pr.ID == "" {
		return domain.PullRequest{}, "", notFound("pull request not found")
	}
	if pr.Status == domain.StatusMerged {
		return domain.PullRequest{}, "", &AppError{
			Code:       ErrCodePRMerged,
			HTTPStatus: 409,
			Message:    "cannot reassign on merged PR",
		}
	}

	found := false
	for _, id := range pr.AssignedReviewers {
		if id == oldUserID {
			found = true
			break
		}
	}
	if !found {
		return domain.PullRequest{}, "", &AppError{
			Code:       ErrCodeNotAssigned,
			HTTPStatus: 409,
			Message:    "reviewer is not assigned to this PR",
		}
	}

	user, err := s.repo.GetUserByID(ctx, oldUserID)
	if err != nil {
		return domain.PullRequest{}, "", err
	}
	if user.UserID == "" {
		return domain.PullRequest{}, "", notFound("user not found")
	}

	exclude := append([]string{oldUserID}, pr.AssignedReviewers...)
	candidates, err := s.repo.GetActiveTeamMembersExcept(ctx, user.TeamName, exclude)
	if err != nil {
		return domain.PullRequest{}, "", err
	}
	if len(candidates) == 0 {
		return domain.PullRequest{}, "", &AppError{
			Code:       ErrCodeNoCandidate,
			HTTPStatus: 409,
			Message:    "no active replacement candidate in team",
		}
	}

	idx := rand.Intn(len(candidates))
	newUser := candidates[idx]

	updatedPR, err := s.repo.ReassignReviewer(ctx, prID, oldUserID, newUser.UserID)
	if err != nil {
		return domain.PullRequest{}, "", err
	}

	return updatedPR, newUser.UserID, nil
}

func (s *Service) GetUserReviewPRs(ctx context.Context, userID string) ([]domain.PullRequestShort, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.UserID == "" {
		return nil, notFound("user not found")
	}
	return s.repo.GetUserReviewPRs(ctx, userID)
}

func ErrorToCodeMessage(err error) (string, int, string) {
	if err == nil {
		return "", 0, ""
	}
	if appErr, ok := err.(*AppError); ok {
		return string(appErr.Code), appErr.HTTPStatus, appErr.Message
	}
	return string(ErrCodeNotFound), 500, fmt.Sprintf("internal error: %v", err)
}
