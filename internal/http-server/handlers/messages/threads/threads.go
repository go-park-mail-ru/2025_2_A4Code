package threads

import (
	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/http-server/middleware/logger"
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"context"
	"strconv"
	"time"

	"encoding/json"
	"net/http"
)

type MessageUsecase interface {
	FindThreadsByProfileID(ctx context.Context, profileID int64) ([]domain.ThreadInfo, error)
}

type Response struct {
	resp.Response
}

type threadResponse struct {
	ID           string `json:"id"`
	RootMessage  string `json:"root_message"`
	LastActivity string `json:"last_activity"`
}

type threadsList struct {
	Threads []threadResponse `json:"threads"`
	Total   int              `json:"total"`
}

type HandlerThreads struct {
	messageUCase MessageUsecase
	secret       []byte
}

func New(messageUCase MessageUsecase, SECRET []byte) *HandlerThreads {
	return &HandlerThreads{
		messageUCase: messageUCase,
		secret:       SECRET,
	}
}

func (h *HandlerThreads) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := logger.GetLogger(r.Context())
	log.Debug("handle /messages/threads")

	if r.Method != http.MethodGet {
		resp.SendErrorResponse(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	id, err := session.GetProfileID(r, h.secret)
	if err != nil {
		resp.SendErrorResponse(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	threads, err := h.messageUCase.FindThreadsByProfileID(r.Context(), id)
	if err != nil {
		log.Error("failed to get threads", "error", err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	threadResponses := make([]threadResponse, 0, len(threads))
	for _, thread := range threads {
		threadResponses = append(threadResponses, threadResponse{
			ID:           strconv.FormatInt(thread.ID, 10),
			RootMessage:  strconv.FormatInt(thread.RootMessage, 10),
			LastActivity: thread.LastActivity.Format(time.RFC3339),
		})
	}

	responseBody := threadsList{
		Threads: threadResponses,
		Total:   len(threadResponses),
	}

	response := Response{
		Response: resp.Response{
			Status:  http.StatusOK,
			Message: "success",
			Body:    responseBody,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}
}
