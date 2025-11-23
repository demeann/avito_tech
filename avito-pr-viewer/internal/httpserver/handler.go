package httpserver

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/demeann/avito-pr-reviewer/internal/domain"
	"github.com/demeann/avito-pr-reviewer/internal/service"
)

type Service interface {
	CreateTeam(ctx context.Context, name string, members []domain.TeamMember) (domain.Team, error)
	GetTeam(ctx context.Context, name string) (domain.Team, error)

	SetUserActive(ctx context.Context, userID string, isActive bool) (domain.User, error)
	CreatePullRequest(ctx context.Context, id, name, authorID string) (domain.PullRequest, error)
	MergePullRequest(ctx context.Context, id string) (domain.PullRequest, error)
	ReassignReviewer(ctx context.Context, prID, oldUserID string) (domain.PullRequest, string, error)
	GetUserReviewPRs(ctx context.Context, userID string) ([]domain.PullRequestShort, error)
}

type Handler struct {
	svc Service
	mux *http.ServeMux
}

func NewHandler(svc Service) *Handler {
	h := &Handler{
		svc: svc,
		mux: http.NewServeMux(),
	}
	h.registerRoutes()
	return h
}

func (h *Handler) Router() http.Handler {
	return h.mux
}

func (h *Handler) registerRoutes() {
	h.mux.HandleFunc("/health", h.handleHealth)

	h.mux.HandleFunc("/team/add", h.handleTeamAdd)
	h.mux.HandleFunc("/team/get", h.handleTeamGet)

	h.mux.HandleFunc("/users/setIsActive", h.handleSetIsActive)
	h.mux.HandleFunc("/users/getReview", h.handleGetUserReview)

	h.mux.HandleFunc("/pullRequest/create", h.handlePRCreate)
	h.mux.HandleFunc("/pullRequest/merge", h.handlePRMerge)
	h.mux.HandleFunc("/pullRequest/reassign", h.handlePRReassign)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, code string, status int, message string) {
	if status == 0 {
		status = http.StatusInternalServerError
	}
	if code == "" {
		code = string(service.ErrCodeNotFound)
	}
	type errBody struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	var body errBody
	body.Error.Code = code
	body.Error.Message = message
	writeJSON(w, status, body)
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
