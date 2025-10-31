package send

import (
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"2025_2_a4code/internal/usecase/message"
	"encoding/json"
	"net/http"
)

var SECRET = []byte("secret") // TODO: убрать отсюда

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
	messageUCase *message.MessageUcase
}

func New(ucM *message.MessageUcase) *HandlerSend {
	return &HandlerSend{messageUCase: ucM}
}

func (h *HandlerSend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		resp.SendErrorResponse(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.SendErrorResponse(w, "Неправильный запрос", http.StatusBadRequest)
		return
	}

	if req.Text == "" || req.Receivers == nil || len(req.Receivers) == 0 {
		resp.SendErrorResponse(w, "Неправильный запрос", http.StatusBadRequest)
		return
	}

	id, err := session.GetProfileID(r, SECRET)
	if err != nil {
		resp.SendErrorResponse(w, err.Error(), http.StatusUnauthorized)
		return
	}
	for _, receiver := range req.Receivers {
		messageID, err := h.messageUCase.SaveMessage(r.Context(), receiver.Email, id, req.Topic, req.Text)
		if err != nil {
			resp.SendErrorResponse(w, "Не удалось сохранить файл: "+err.Error(), http.StatusInternalServerError)
			return
		}

		for _, file := range req.Files {
			_, err = h.messageUCase.SaveFile(r.Context(), messageID, file.Name, file.FileType, file.StoragePath, file.Size)
			if err != nil {
				resp.SendErrorResponse(w, "Не удалось сохранить файл: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	response := Response{
		Response: resp.Response{
			Status: http.StatusText(http.StatusOK),
			Body:   struct{}{},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
