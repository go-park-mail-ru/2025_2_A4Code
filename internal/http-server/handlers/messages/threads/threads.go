package threads

import (
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"log/slog"
	"strconv"
	"time"

	messageUcase "2025_2_a4code/internal/usecase/message"
	profileUcase "2025_2_a4code/internal/usecase/profile"
	"encoding/json"
	"net/http"
)

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
	profileUCase *profileUcase.ProfileUcase
	messageUCase *messageUcase.MessageUcase
	log          *slog.Logger
	secret       []byte
}

func New(profileUCase *profileUcase.ProfileUcase, messageUCase *messageUcase.MessageUcase, log *slog.Logger, SECRET []byte) *HandlerThreads {
	return &HandlerThreads{
		profileUCase: profileUCase,
		messageUCase: messageUCase,
		log:          log,
		secret:       SECRET,
	}
}

func (h *HandlerThreads) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := h.log
	log.Info("handle /messages/threads")

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
