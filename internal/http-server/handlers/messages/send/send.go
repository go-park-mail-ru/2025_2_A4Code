package send

import (
	"2025_2_a4code/internal/http-server/middleware/logger"
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"context"
	"encoding/json"
	"net/http"
)

type MessageUsecase interface {
	SaveMessage(ctx context.Context, receiverProfileEmail string, senderBaseProfileID int64, topic, text string) (int64, error)
	SaveFile(ctx context.Context, messageID int64, fileName, fileType, storagePath string, size int64) (fileID int64, err error)
	SaveThreadIdToMessage(ctx context.Context, messageID int64, threadID int64) error
	SaveThread(ctx context.Context, messageID int64) (threadID int64, err error)
}

type File struct {
	Name        string `json:"name"`
	FileType    string `json:"file_type"`
	Size        int64  `json:"size"`
	StoragePath string `json:"storage_path"`
}

type Files []File

type Receiver struct {
	Email string `json:"email"`
}
type Receivers []Receiver

type Request struct {
	Topic     string    `json:"topic"`
	Text      string    `json:"text"`
	Receivers Receivers `json:"receivers"`
	Files     Files     `json:"files"`
}

type Response struct {
	resp.Response
}

type HandlerSend struct {
	messageUCase MessageUsecase
	secret       []byte
}

func New(messageUCase MessageUsecase, SECRET []byte) *HandlerSend {
	return &HandlerSend{
		messageUCase: messageUCase,
		secret:       SECRET,
	}
}

func (h *HandlerSend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := logger.GetLogger(r.Context())
	log.Debug("handle messages/send")

	if r.Method != http.MethodPost {
		resp.SendErrorResponse(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.SendErrorResponse(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if req.Text == "" || req.Receivers == nil || len(req.Receivers) == 0 {
		resp.SendErrorResponse(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	id, err := session.GetProfileID(r, h.secret)
	if err != nil {
		resp.SendErrorResponse(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	for _, receiver := range req.Receivers {
		messageID, err := h.messageUCase.SaveMessage(r.Context(), receiver.Email, id, req.Topic, req.Text)
		if err != nil {
			log.Error(err.Error())
			resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
			return
		}
		threadID, err := h.messageUCase.SaveThread(r.Context(), messageID)
		if err != nil {
			log.Error(err.Error())
			resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
			return
		}

		err = h.messageUCase.SaveThreadIdToMessage(r.Context(), messageID, threadID)

		for _, file := range req.Files {
			_, err = h.messageUCase.SaveFile(r.Context(), messageID, file.Name, file.FileType, file.StoragePath, file.Size)
			if err != nil {
				log.Error(err.Error())
				resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
				return
			}
		}
	}

	response := Response{
		Response: resp.Response{
			Status:  http.StatusOK,
			Message: "success",
			Body:    struct{}{},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
