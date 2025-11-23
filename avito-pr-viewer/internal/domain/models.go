package domain

import "time"

type TeamMember struct {
	UserID   string
	Username string
	IsActive bool
}

type Team struct {
	Name    string
	Members []TeamMember
}

type User struct {
	UserID   string
	Username string
	TeamName string
	IsActive bool
}

type PRStatus string

const (
	StatusOpen   PRStatus = "OPEN"
	StatusMerged PRStatus = "MERGED"
)

type PullRequest struct {
	ID                string
	Name              string
	AuthorID          string
	Status            PRStatus
	AssignedReviewers []string
	CreatedAt         *time.Time
	MergedAt          *time.Time
}

type PullRequestShort struct {
	ID       string
	Name     string
	AuthorID string
	Status   PRStatus
}
