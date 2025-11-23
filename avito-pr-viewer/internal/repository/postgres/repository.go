package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/demeann/avito-pr-reviewer/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) TeamExists(ctx context.Context, name string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(1) FROM teams WHERE name = $1`, name).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *Repository) PRExists(ctx context.Context, id string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(1) FROM pull_requests WHERE pr_id = $1`, id).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *Repository) CreateTeamWithMembers(ctx context.Context, name string, members []domain.TeamMember) (domain.Team, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Team{}, err
	}
	defer tx.Rollback(ctx)

	var teamID int
	err = tx.QueryRow(ctx, `INSERT INTO teams (name) VALUES ($1) RETURNING id`, name).Scan(&teamID)
	if err != nil {
		return domain.Team{}, err
	}

	for _, m := range members {
		_, err := tx.Exec(ctx, `
			INSERT INTO users (user_id, username, team_id, is_active)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id)
			DO UPDATE SET username = EXCLUDED.username, team_id = EXCLUDED.team_id, is_active = EXCLUDED.is_active
		`, m.UserID, m.Username, teamID, m.IsActive)
		if err != nil {
			return domain.Team{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Team{}, err
	}

	return r.GetTeam(ctx, name)
}

func (r *Repository) GetTeam(ctx context.Context, name string) (domain.Team, error) {
	var team domain.Team
	var teamID int
	err := r.pool.QueryRow(ctx, `SELECT id, name FROM teams WHERE name = $1`, name).Scan(&teamID, &team.Name)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Team{}, nil
		}
		return domain.Team{}, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT user_id, username, is_active
		FROM users
		WHERE team_id = $1
		ORDER BY user_id
	`, teamID)
	if err != nil {
		return domain.Team{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var m domain.TeamMember
		if err := rows.Scan(&m.UserID, &m.Username, &m.IsActive); err != nil {
			return domain.Team{}, err
		}
		team.Members = append(team.Members, m)
	}
	if err := rows.Err(); err != nil {
		return domain.Team{}, err
	}

	return team, nil
}

func (r *Repository) SetUserActive(ctx context.Context, userID string, isActive bool) (domain.User, error) {
	var u domain.User
	var teamID int
	err := r.pool.QueryRow(ctx, `
		UPDATE users SET is_active = $2
		WHERE user_id = $1
		RETURNING user_id, username, is_active, team_id
	`, userID, isActive).Scan(&u.UserID, &u.Username, &u.IsActive, &teamID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.User{}, nil
		}
		return domain.User{}, err
	}
	err = r.pool.QueryRow(ctx, `SELECT name FROM teams WHERE id = $1`, teamID).Scan(&u.TeamName)
	if err != nil {
		return domain.User{}, err
	}
	return u, nil
}

func (r *Repository) GetTeamNameByUserID(ctx context.Context, userID string) (string, error) {
	var teamName string
	err := r.pool.QueryRow(ctx, `
		SELECT t.name
		FROM users u
		JOIN teams t ON u.team_id = t.id
		WHERE u.user_id = $1
	`, userID).Scan(&teamName)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return teamName, nil
}

func (r *Repository) GetUserByID(ctx context.Context, userID string) (domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT u.user_id, u.username, u.is_active, t.name
		FROM users u
		JOIN teams t ON u.team_id = t.id
		WHERE u.user_id = $1
	`, userID).Scan(&u.UserID, &u.Username, &u.IsActive, &u.TeamName)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.User{}, nil
		}
		return domain.User{}, err
	}
	return u, nil
}

func (r *Repository) GetActiveTeamMembersExcept(ctx context.Context, teamName string, exclude []string) ([]domain.User, error) {
	if len(exclude) == 0 {
		exclude = []string{""}
	}
	args := []interface{}{teamName}
	ph := make([]string, len(exclude))
	for i, id := range exclude {
		args = append(args, id)
		ph[i] = fmt.Sprintf("$%d", i+2)
	}
	query := `
		SELECT u.user_id, u.username, u.is_active, t.name
		FROM users u
		JOIN teams t ON u.team_id = t.id
		WHERE t.name = $1 AND u.is_active = TRUE AND u.user_id NOT IN (` + strings.Join(ph, ",") + `)
	`
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.UserID, &u.Username, &u.IsActive, &u.TeamName); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *Repository) CreatePullRequest(ctx context.Context, id, name, authorID string, createdAt time.Time, reviewers []string) (domain.PullRequest, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.PullRequest{}, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO pull_requests (pr_id, pr_name, author_id, status, created_at, merged_at)
		VALUES ($1, $2, $3, $4, $5, NULL)
	`, id, name, authorID, string(domain.StatusOpen), createdAt)
	if err != nil {
		return domain.PullRequest{}, err
	}

	for _, rid := range reviewers {
		if _, err := tx.Exec(ctx, `
			INSERT INTO pull_request_reviewers (pr_id, reviewer_id)
			VALUES ($1, $2)
		`, id, rid); err != nil {
			return domain.PullRequest{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.PullRequest{}, err
	}

	return r.GetPullRequest(ctx, id)
}

func (r *Repository) GetPullRequest(ctx context.Context, id string) (domain.PullRequest, error) {
	var pr domain.PullRequest
	var createdAt, mergedAt *time.Time
	var status string
	err := r.pool.QueryRow(ctx, `
		SELECT pr_id, pr_name, author_id, status, created_at, merged_at
		FROM pull_requests
		WHERE pr_id = $1
	`, id).Scan(&pr.ID, &pr.Name, &pr.AuthorID, &status, &createdAt, &mergedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.PullRequest{}, nil
		}
		return domain.PullRequest{}, err
	}
	pr.Status = domain.PRStatus(status)
	pr.CreatedAt = createdAt
	pr.MergedAt = mergedAt

	rows, err := r.pool.Query(ctx, `
		SELECT reviewer_id FROM pull_request_reviewers
		WHERE pr_id = $1
		ORDER BY reviewer_id
	`, id)
	if err != nil {
		return domain.PullRequest{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var rid string
		if err := rows.Scan(&rid); err != nil {
			return domain.PullRequest{}, err
		}
		pr.AssignedReviewers = append(pr.AssignedReviewers, rid)
	}
	if err := rows.Err(); err != nil {
		return domain.PullRequest{}, err
	}

	return pr, nil
}

func (r *Repository) SetPullRequestMerged(ctx context.Context, id string, mergedAt time.Time) (domain.PullRequest, error) {
	_, err := r.pool.Exec(ctx, `
		UPDATE pull_requests
		SET status = $2,
			merged_at = COALESCE(merged_at, $3)
		WHERE pr_id = $1
	`, id, string(domain.StatusMerged), mergedAt)
	if err != nil {
		return domain.PullRequest{}, err
	}
	return r.GetPullRequest(ctx, id)
}

func (r *Repository) ReassignReviewer(ctx context.Context, prID, oldUserID, newUserID string) (domain.PullRequest, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.PullRequest{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		DELETE FROM pull_request_reviewers
		WHERE pr_id = $1 AND reviewer_id = $2
	`, prID, oldUserID); err != nil {
		return domain.PullRequest{}, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO pull_request_reviewers (pr_id, reviewer_id)
		VALUES ($1, $2)
	`, prID, newUserID); err != nil {
		return domain.PullRequest{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.PullRequest{}, err
	}

	return r.GetPullRequest(ctx, prID)
}

func (r *Repository) GetUserReviewPRs(ctx context.Context, userID string) ([]domain.PullRequestShort, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT pr.pr_id, pr.pr_name, pr.author_id, pr.status
		FROM pull_requests pr
		INNER JOIN pull_request_reviewers rvr ON pr.pr_id = rvr.pr_id
		WHERE rvr.reviewer_id = $1
		ORDER BY pr.created_at
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.PullRequestShort
	for rows.Next() {
		var pr domain.PullRequestShort
		var status string
		if err := rows.Scan(&pr.ID, &pr.Name, &pr.AuthorID, &status); err != nil {
			return nil, err
		}
		pr.Status = domain.PRStatus(status)
		result = append(result, pr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
