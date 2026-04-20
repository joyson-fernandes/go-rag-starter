package ragbot

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type Handler struct {
	svc *Service
	web fs.FS
}

func NewHandler(svc *Service, web fs.FS) *Handler {
	return &Handler{svc: svc, web: web}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/query", h.Query)
	mux.HandleFunc("POST /api/messages/{id}/feedback", h.Feedback)
	mux.HandleFunc("GET /widget.js", h.Widget)
	mux.HandleFunc("GET /", h.Index)
	mux.HandleFunc("GET /healthz", h.Health)
}

type queryRequest struct {
	Message        string `json:"message"`
	ConversationID string `json:"conversation_id,omitempty"`
}

func (h *Handler) Query(w http.ResponseWriter, r *http.Request) {
	var req queryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	var convID uuid.UUID
	if req.ConversationID != "" {
		id, err := uuid.Parse(req.ConversationID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid conversation_id")
			return
		}
		convID = id
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	events := make(chan StreamResult, 32)
	done := make(chan struct{})
	var svcErr error

	go func() {
		defer close(events)
		defer close(done)
		svcErr = h.svc.Query(ctx, QueryInput{
			UserID:         r.Header.Get("X-User-ID"), // optional
			ConversationID: convID,
			Message:        req.Message,
		}, events)
	}()

	for ev := range events {
		switch {
		case ev.Header != nil:
			sendEvent(w, flusher, "meta", map[string]any{
				"conversation_id": ev.Header.ConversationID.String(),
				"assistant_id":    ev.Header.AssistantMsgID,
				"sources":         ev.Header.Sources,
			})
		case ev.Err != nil:
			sendEvent(w, flusher, "error", map[string]string{"error": ev.Err.Error()})
			return
		case ev.Done:
			sendEvent(w, flusher, "done", map[string]any{"sources": ev.Sources})
			flusher.Flush()
			<-done
			return
		case ev.Token != "":
			sendEvent(w, flusher, "token", map[string]string{"t": ev.Token})
		}
	}
	<-done
	if svcErr != nil {
		sendEvent(w, flusher, "error", map[string]string{"error": svcErr.Error()})
	}
}

type feedbackRequest struct {
	Feedback int16 `json:"feedback"`
}

func (h *Handler) Feedback(w http.ResponseWriter, r *http.Request) {
	msgID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid message id")
		return
	}
	var req feedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Feedback < -1 || req.Feedback > 1 {
		writeError(w, http.StatusBadRequest, "feedback must be -1, 0, or 1")
		return
	}
	if err := h.svc.Store.UpdateFeedback(r.Context(), msgID, req.Feedback); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Widget(w http.ResponseWriter, r *http.Request) {
	b, err := fs.ReadFile(h.web, "web/widget.js")
	if err != nil {
		http.Error(w, "widget not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write(b)
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	b, err := fs.ReadFile(h.web, "web/index.html")
	if err != nil {
		http.Error(w, "index not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(b)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Store.Ping(r.Context()); err != nil {
		http.Error(w, "db unhealthy", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "ok")
}

func sendEvent(w http.ResponseWriter, flusher http.Flusher, event string, payload any) {
	b, _ := json.Marshal(payload)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
	flusher.Flush()
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
