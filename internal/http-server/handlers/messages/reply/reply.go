package reply

import (
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"2025_2_a4code/internal/usecase/message"
	"encoding/json"
	"log/slog"
	"net/http"
)

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
	RootMessageID int64     `json:"root_message_id"`
	Topic         string    `json:"topic"`
	Text          string    `json:"text"`
	ThreadRoot    string    `json:"thread_root"`
	Receivers     Receivers `json:"receivers"`
	Files         Files     `json:"files"`
}

type Response struct {
	resp.Response
}

type HandlerReply struct {
	messageUCase *message.MessageUcase
	secret       []byte
	log          *slog.Logger
}

func New(messageUCase *message.MessageUcase, SECRET []byte, log *slog.Logger) *HandlerReply {
	return &HandlerReply{
		messageUCase: messageUCase,
		secret:       SECRET,
		log:          log,
	}
}

func (h *HandlerReply) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := h.log
	log.Info("handle reply")

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

		threadId, err := h.messageUCase.SaveThread(r.Context(), req.RootMessageID, req.ThreadRoot) // TODO: проверка на существование треда
		err = h.messageUCase.SaveThreadIdToMessage(r.Context(), messageID, threadId)
		if err != nil {
			log.Error(err.Error())
			resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		}

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
			Status:  http.StatusText(http.StatusOK),
			Message: "success",
			Body:    struct{}{},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
