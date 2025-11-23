package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/demeann/avito-pr-reviewer/internal/service"
)

type prCreateRequest struct {
	ID       string `json:"pull_request_id"`
	Name     string `json:"pull_request_name"`
	AuthorID string `json:"author_id"`
}

type prResponse struct {
	PR struct {
		ID                string   `json:"pull_request_id"`
		Name              string   `json:"pull_request_name"`
		AuthorID          string   `json:"author_id"`
		Status            string   `json:"status"`
		AssignedReviewers []string `json:"assigned_reviewers"`
		CreatedAt         *string  `json:"createdAt,omitempty"`
		MergedAt          *string  `json:"mergedAt,omitempty"`
	} `json:"pr"`
}

func (h *Handler) handlePRCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req prCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, string(service.ErrCodeNotFound), http.StatusBadRequest, "invalid json")
		return
	}

	pr, err := h.svc.CreatePullRequest(context.Background(), req.ID, req.Name, req.AuthorID)
	if err != nil {
		code, status, msg := service.ErrorToCodeMessage(err)
		writeError(w, code, status, msg)
		return
	}

	var resp prResponse
	resp.PR.ID = pr.ID
	resp.PR.Name = pr.Name
	resp.PR.AuthorID = pr.AuthorID
	resp.PR.Status = string(pr.Status)
	resp.PR.AssignedReviewers = pr.AssignedReviewers
	if pr.CreatedAt != nil {
		s := pr.CreatedAt.Format(time.RFC3339)
		resp.PR.CreatedAt = &s
	}
	if pr.MergedAt != nil {
		s := pr.MergedAt.Format(time.RFC3339)
		resp.PR.MergedAt = &s
	}

	writeJSON(w, http.StatusCreated, resp)
}

type prMergeRequest struct {
	ID string `json:"pull_request_id"`
}

func (h *Handler) handlePRMerge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req prMergeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, string(service.ErrCodeNotFound), http.StatusBadRequest, "invalid json")
		return
	}

	pr, err := h.svc.MergePullRequest(context.Background(), req.ID)
	if err != nil {
		code, status, msg := service.ErrorToCodeMessage(err)
		writeError(w, code, status, msg)
		return
	}

	var resp prResponse
	resp.PR.ID = pr.ID
	resp.PR.Name = pr.Name
	resp.PR.AuthorID = pr.AuthorID
	resp.PR.Status = string(pr.Status)
	resp.PR.AssignedReviewers = pr.AssignedReviewers
	if pr.CreatedAt != nil {
		s := pr.CreatedAt.Format(time.RFC3339)
		resp.PR.CreatedAt = &s
	}
	if pr.MergedAt != nil {
		s := pr.MergedAt.Format(time.RFC3339)
		resp.PR.MergedAt = &s
	}
	writeJSON(w, http.StatusOK, resp)
}

type prReassignRequest struct {
	ID        string `json:"pull_request_id"`
	OldUserID string `json:"old_user_id"`
}

type prReassignResponse struct {
	PR         prResponsePR `json:"pr"`
	ReplacedBy string       `json:"replaced_by"`
}

type prResponsePR struct {
	ID                string   `json:"pull_request_id"`
	Name              string   `json:"pull_request_name"`
	AuthorID          string   `json:"author_id"`
	Status            string   `json:"status"`
	AssignedReviewers []string `json:"assigned_reviewers"`
	CreatedAt         *string  `json:"createdAt,omitempty"`
	MergedAt          *string  `json:"mergedAt,omitempty"`
}

func (h *Handler) handlePRReassign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req prReassignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, string(service.ErrCodeNotFound), http.StatusBadRequest, "invalid json")
		return
	}

	pr, replacedBy, err := h.svc.ReassignReviewer(context.Background(), req.ID, req.OldUserID)
	if err != nil {
		code, status, msg := service.ErrorToCodeMessage(err)
		writeError(w, code, status, msg)
		return
	}

	var createdAt, mergedAt *string
	if pr.CreatedAt != nil {
		s := pr.CreatedAt.Format(time.RFC3339)
		createdAt = &s
	}
	if pr.MergedAt != nil {
		s := pr.MergedAt.Format(time.RFC3339)
		mergedAt = &s
	}

	resp := prReassignResponse{
		PR: prResponsePR{
			ID:                pr.ID,
			Name:              pr.Name,
			AuthorID:          pr.AuthorID,
			Status:            string(pr.Status),
			AssignedReviewers: pr.AssignedReviewers,
			CreatedAt:         createdAt,
			MergedAt:          mergedAt,
		},
		ReplacedBy: replacedBy,
	}

	writeJSON(w, http.StatusOK, resp)
}
