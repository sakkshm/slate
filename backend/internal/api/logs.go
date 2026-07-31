package api

import (
	"fmt"
	"net/http"
	"strings"

	buildpkg "slate-backend/internal/build"
	"slate-backend/internal/project"
	"slate-backend/internal/queue"
	"slate-backend/pkg/types"
	"slate-backend/pkg/utils"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (e *APIEngine) HandleBuildLogs(w http.ResponseWriter, r *http.Request) {
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
		utils.WriteHTTPError(w, http.StatusNotFound, "BAD_REQ", "Project not owned by current user")
		return
	}

	buildID, err := uuid.Parse(chi.URLParam(r, "buildID"))
	if err != nil {
		utils.WriteHTTPError(w, http.StatusBadRequest, "BAD_REQ", "Invalid build ID")
		return
	}

	b, err := buildpkg.GetBuildByID(e.clients.DB, buildID, r.Context())
	if err != nil {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "DB_ERR", "Failed to fetch build")
		return
	}
	if b == nil || b.ProjectID != projectID {
		utils.WriteHTTPError(w, http.StatusNotFound, "BAD_REQ", "Build not found")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		utils.WriteHTTPError(w, http.StatusInternalServerError, "SSE_ERR", "Streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	bid := buildID.String()

	emit := func(line string) {
		fmt.Fprintf(w, "data: %s\n\n", line)
		flusher.Flush()
	}
	isTerminal := func(status types.BuildEvents) bool {
		return status == types.StatusReady || status == types.StatusFailed || status == types.StatusCancelled
	}
	finish := func(b *types.Build) {
		if b.LogContent != "" {
			for _, line := range strings.Split(b.LogContent, "\n") {
				emit(line)
			}
		}
		fmt.Fprintf(w, "event: done\n\n")
		flusher.Flush()
	}

	if isTerminal(b.Status) {
		finish(b)
		return
	}

	sub := e.clients.Redis.Subscribe(r.Context(), queue.LogChanKey(bid), queue.DoneChanKey(bid))
	defer sub.Close()

	b, err = buildpkg.GetBuildByID(e.clients.DB, buildID, r.Context())
	if err == nil && b != nil && isTerminal(b.Status) {
		finish(b)
		return
	}

	l1, _ := queue.GetLogLines(r.Context(), e.clients.Redis, bid)
	for _, line := range l1 {
		emit(line)
	}

	l2, _ := queue.GetLogLines(r.Context(), e.clients.Redis, bid)
	delta := l2[len(l1):]
	for _, line := range delta {
		emit(line)
	}
	skip := len(delta)

	ch := sub.Channel()
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if msg.Channel == queue.DoneChanKey(bid) {
				fmt.Fprintf(w, "event: done\n\n")
				flusher.Flush()
				return
			}
			if skip > 0 {
				skip--
				continue
			}
			emit(msg.Payload)
		case <-r.Context().Done():
			return
		}
	}
}
