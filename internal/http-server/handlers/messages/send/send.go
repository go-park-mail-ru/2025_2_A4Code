package send

import (
	"2025_2_a4code/internal/http-server/middleware/logger"
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"2025_2_a4code/internal/lib/validation"
	"context"
	"encoding/json"
	"html"
	"net/http"
	"net/mail"
	"path/filepath"
	"strings"
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

const (
	maxTopicLen       = 255
	maxTextLen        = 10000
	maxFileSize       = 10 * 1024 * 1024 // 10 MB
	defaultLimitFiles = 20
)

var allowedFileTypes = map[string]struct{}{
	"image/jpeg":      {},
	"image/png":       {},
	"application/pdf": {},
	"text/plain":      {},
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

	if errMsg := validateRequest(&req); errMsg != "" {
		resp.SendErrorResponse(w, errMsg, http.StatusBadRequest)
		return
	}

	id, err := session.GetProfileID(r, h.secret)
	if err != nil {
		resp.SendErrorResponse(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	escapedTopic := html.EscapeString(req.Topic)
	escapedText := html.EscapeString(req.Text)

	for _, receiver := range req.Receivers {
		messageID, err := h.messageUCase.SaveMessage(r.Context(), receiver.Email, id, escapedTopic, escapedText)
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

func validateRequest(req *Request) string {
	// topic length
	if len(req.Topic) > maxTopicLen {
		return "topic too long"
	}
	// text length
	if len(req.Text) > maxTextLen {
		return "text too long"
	}

	// запрещаем опасные последовательности в topic/text
	if validation.HasDangerousCharacters(req.Topic) {
		return "topic contains forbidden characters"
	}
	if validation.HasDangerousCharacters(req.Text) {
		return "text contains forbidden characters"
	}

	// receivers: проверяем корректность email и дубликаты
	seen := make(map[string]struct{})
	for _, r := range req.Receivers {
		email := strings.TrimSpace(r.Email)
		if email == "" {
			return "empty receiver email"
		}
		// корректность email по net/mail
		if _, err := mail.ParseAddress(email); err != nil {
			return "invalid receiver email: " + email
		}
		lower := strings.ToLower(email)
		if _, ok := seen[lower]; ok {
			return "duplicate receiver: " + email
		}
		seen[lower] = struct{}{}

		// дополнительно блокируем опасные последовательности в email (избыточно, но безопасно)
		if validation.HasDangerousCharacters(email) {
			return "receiver email contains forbidden characters: " + email
		}
	}

	// files: ограничение количества, размера и типов; проверка имени
	if len(req.Files) > defaultLimitFiles {
		return "too many files"
	}
	for _, f := range req.Files {
		// размер
		if f.Size < 0 || f.Size > maxFileSize {
			return "file size invalid or too large: " + f.Name
		}
		// тип
		if _, ok := allowedFileTypes[f.FileType]; !ok {
			return "unsupported file type: " + f.FileType
		}
		// имя не должно содержать путь traversal
		base := filepath.Base(f.Name)
		if base != f.Name || strings.Contains(f.Name, "..") {
			return "invalid file name: " + f.Name
		}
		// storage path тоже проверяем на опасные последовательности
		if validation.HasDangerousCharacters(f.StoragePath) {
			return "invalid storage path for file: " + f.Name
		}
		// запрещаем опасные последовательности в имени
		if validation.HasDangerousCharacters(f.Name) {
			return "invalid file name: " + f.Name
		}
	}

	return ""
}
