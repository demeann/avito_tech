package httpserver

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/demeann/avito-pr-reviewer/internal/service"
)

type setIsActiveRequest struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

type setIsActiveResponse struct {
	User struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		TeamName string `json:"team_name"`
		IsActive bool   `json:"is_active"`
	} `json:"user"`
}

func (h *Handler) handleSetIsActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req setIsActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, string(service.ErrCodeNotFound), http.StatusBadRequest, "invalid json")
		return
	}

	user, err := h.svc.SetUserActive(context.Background(), req.UserID, req.IsActive)
	if err != nil {
		code, status, msg := service.ErrorToCodeMessage(err)
		writeError(w, code, status, msg)
		return
	}

	var resp setIsActiveResponse
	resp.User.UserID = user.UserID
	resp.User.Username = user.Username
	resp.User.TeamName = user.TeamName
	resp.User.IsActive = user.IsActive

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleGetUserReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, string(service.ErrCodeNotFound), http.StatusBadRequest, "user_id is required")
		return
	}

	prs, err := h.svc.GetUserReviewPRs(context.Background(), userID)
	if err != nil {
		code, status, msg := service.ErrorToCodeMessage(err)
		writeError(w, code, status, msg)
		return
	}

	type prShort struct {
		ID       string `json:"pull_request_id"`
		Name     string `json:"pull_request_name"`
		AuthorID string `json:"author_id"`
		Status   string `json:"status"`
	}
	resp := struct {
		UserID       string    `json:"user_id"`
		PullRequests []prShort `json:"pull_requests"`
	}{
		UserID: userID,
	}

	for _, pr := range prs {
		resp.PullRequests = append(resp.PullRequests, prShort{
			ID:       pr.ID,
			Name:     pr.Name,
			AuthorID: pr.AuthorID,
			Status:   string(pr.Status),
		})
	}

	writeJSON(w, http.StatusOK, resp)
}
