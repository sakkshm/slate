package api

import (
	"encoding/json"
	"net/http"
	"slate-backend/internal/project"
	"slate-backend/internal/envvar"
	"slate-backend/pkg/types"
	"slate-backend/pkg/utils"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (e *APIEngine) HandleListEnvVars(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		utils.WriteHTTPError(w, http.StatusUnauthorized, "BAD_REQ", "User ID not found")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "Invalid project ID")
		return
	}

	proj, err := project.GetProjectByID(projectID, e.clients.DB, r.Context())
	if err != nil || proj == nil {
		utils.WriteHTTPError(w, http.StatusNotFound, "BAD_REQ", "Project not found")
		return
	}
	if proj.OwnerID != userID {
		utils.WriteHTTPError(w, http.StatusForbidden, "FORBIDDEN", "You do not have access to this project")
		return
	}

	vars, err := envvar.ListByProject(e.clients.DB, projectID, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "DB_ERR", "Failed to fetch env vars")
		return
	}

	resp := make([]types.EnvVarResponse, 0, len(vars))
	for _, v := range vars {
		resp = append(resp, types.EnvVarResponse{
			Key:       v.Key,
			Value:     "*********",
			UpdatedAt: v.UpdatedAt,
		})
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (e *APIEngine) HandleUpsertEnvVar(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		utils.WriteHTTPError(w, http.StatusUnauthorized, "BAD_REQ", "User ID not found")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "Invalid project ID")
		return
	}

	proj, err := project.GetProjectByID(projectID, e.clients.DB, r.Context())
	if err != nil || proj == nil {
		utils.WriteHTTPError(w, http.StatusNotFound, "BAD_REQ", "Project not found")
		return
	}
	if proj.OwnerID != userID {
		utils.WriteHTTPError(w, http.StatusForbidden, "FORBIDDEN", "You do not have access to this project")
		return
	}

	var req types.UpsertEnvVarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "Invalid request body")
		return
	}
	if req.Key == "" || req.Value == "" {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "key and value are required")
		return
	}

	encrypted, err := utils.EncryptAESString(req.Value, []byte(e.config.EncryptionKey))
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "ENCRYPT_ERR", "Failed to encrypt value")
		return
	}

	if err := envvar.UpsertEnvVars(e.clients.DB, projectID, req.Key, encrypted, r.Context()); err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "DB_ERR", "Failed to save env var")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (e *APIEngine) HandleDeleteEnvVar(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		utils.WriteHTTPError(w, http.StatusUnauthorized, "BAD_REQ", "User ID not found")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "Invalid project ID")
		return
	}

	proj, err := project.GetProjectByID(projectID, e.clients.DB, r.Context())
	if err != nil || proj == nil {
		utils.WriteHTTPError(w, http.StatusNotFound, "BAD_REQ", "Project not found")
		return
	}
	if proj.OwnerID != userID {
		utils.WriteHTTPError(w, http.StatusForbidden, "FORBIDDEN", "You do not have access to this project")
		return
	}

	key := chi.URLParam(r, "key")
	if err := envvar.Delete(e.clients.DB, projectID, key, r.Context()); err != nil {
		utils.WriteHTTPError(w, http.StatusNotFound, "BAD_REQ", "Env var not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
