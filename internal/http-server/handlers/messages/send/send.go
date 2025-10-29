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
	Body struct{} `json:"body"`
}

type HandlerSend struct {
	messageUCase *message.MessageUcase
}

func New(ucM *message.MessageUcase) *HandlerSend {
	return &HandlerSend{messageUCase: ucM}
}

func (h *HandlerSend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorResponse(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Неправильный запрос", http.StatusBadRequest)
		return
	}

	if req.Text == "" || req.Receivers == nil || len(req.Receivers) == 0 {
		sendErrorResponse(w, "Неправильный запрос", http.StatusBadRequest)
		return
	}

	id, err := session.GetProfileID(r, SECRET)
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusUnauthorized)
		return
	}
	for _, receiver := range req.Receivers {
		messageID, err := h.messageUCase.SaveMessage(receiver.Email, id, req.Topic, req.Text)
		if err != nil {
			sendErrorResponse(w, "Не удалось сохранить файл: "+err.Error(), http.StatusInternalServerError)
			return
		}

		for _, file := range req.Files {
			_, err = h.messageUCase.SaveFile(messageID, file.Name, file.FileType, file.StoragePath, file.Size)
			if err != nil {
				sendErrorResponse(w, "Не удалось сохранить файл: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	response := Response{
		Response: resp.Response{
			Status: http.StatusText(http.StatusOK),
		},
		Body: struct{}{},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func sendErrorResponse(w http.ResponseWriter, errorMsg string, statusCode int) {

	response := Response{
		Response: resp.Response{
			Status:  http.StatusText(statusCode),
			Message: "Ошибка: " + errorMsg,
		},
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(&response)
}
