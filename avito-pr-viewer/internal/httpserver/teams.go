package httpserver

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/demeann/avito-pr-reviewer/internal/domain"
	"github.com/demeann/avito-pr-reviewer/internal/service"
)

type teamRequest struct {
	TeamName string `json:"team_name"`
	Members  []struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		IsActive bool   `json:"is_active"`
	} `json:"members"`
}

type teamResponse struct {
	Team struct {
		TeamName string `json:"team_name"`
		Members  []struct {
			UserID   string `json:"user_id"`
			Username string `json:"username"`
			IsActive bool   `json:"is_active"`
		} `json:"members"`
	} `json:"team"`
}

func (h *Handler) handleTeamAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req teamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, string(service.ErrCodeNotFound), http.StatusBadRequest, "invalid json")
		return
	}

	members := make([]domain.TeamMember, 0, len(req.Members))
	for _, m := range req.Members {
		members = append(members, domain.TeamMember{
			UserID:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}

	team, err := h.svc.CreateTeam(context.Background(), req.TeamName, members)
	if err != nil {
		code, status, msg := service.ErrorToCodeMessage(err)
		writeError(w, code, status, msg)
		return
	}

	var resp teamResponse
	resp.Team.TeamName = team.Name
	for _, m := range team.Members {
		resp.Team.Members = append(resp.Team.Members, struct {
			UserID   string "json:\"user_id\""
			Username string "json:\"username\""
			IsActive bool   "json:\"is_active\""
		}{
			UserID:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) handleTeamGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		writeError(w, string(service.ErrCodeNotFound), http.StatusBadRequest, "team_name is required")
		return
	}

	team, err := h.svc.GetTeam(context.Background(), teamName)
	if err != nil {
		code, status, msg := service.ErrorToCodeMessage(err)
		writeError(w, code, status, msg)
		return
	}

	type member struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		IsActive bool   `json:"is_active"`
	}
	resp := struct {
		TeamName string   `json:"team_name"`
		Members  []member `json:"members"`
	}{
		TeamName: team.Name,
	}
	for _, m := range team.Members {
		resp.Members = append(resp.Members, member{
			UserID:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}
