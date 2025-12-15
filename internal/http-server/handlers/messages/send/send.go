package send

//go:generate mockgen -source=$GOFILE -destination=./mocks/mock_send_usecase.go -package=mocks

import (
	"2025_2_a4code/internal/http-server/middleware/logger"
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"2025_2_a4code/internal/lib/validation"
	"context"
	"encoding/json"
	"fmt"
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
	maxFileSize       = 40_000_000 // 40 MB (decimal)
	maxTotalFilesSize = 40_000_000 // 40 MB (decimal)
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
		resp.SendErrorResponse(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.SendErrorResponse(w, "Некорректный формат запроса", http.StatusBadRequest)
		return
	}

	if err := validateRequest(&req); err != nil {
		resp.SendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := session.GetProfileID(r, h.secret)
	if err != nil {
		resp.SendErrorResponse(w, "Необходима авторизация", http.StatusUnauthorized)
		return
	}

	for _, receiver := range req.Receivers {
		email := strings.TrimSpace(strings.ToLower(receiver.Email))
		messageID, err := h.messageUCase.SaveMessage(r.Context(), email, id, req.Topic, req.Text)
		if err != nil {
			log.Error(err.Error())
			resp.SendErrorResponse(w, "Произошла ошибка", http.StatusInternalServerError)
			return
		}
		threadID, err := h.messageUCase.SaveThread(r.Context(), messageID)
		if err != nil {
			log.Error(err.Error())
			resp.SendErrorResponse(w, "Произошла ошибка", http.StatusInternalServerError)
			return
		}

		err = h.messageUCase.SaveThreadIdToMessage(r.Context(), messageID, threadID)
		if err != nil {
			log.Error(err.Error())
			resp.SendErrorResponse(w, "Произошла ошибка", http.StatusInternalServerError)
			return
		}

		for _, file := range req.Files {
			_, err = h.messageUCase.SaveFile(r.Context(), messageID, file.Name, file.FileType, file.StoragePath, file.Size)
			if err != nil {
				log.Error(err.Error())
				resp.SendErrorResponse(w, "Произошла ошибка", http.StatusInternalServerError)
				return
			}
		}
	}

	response := Response{
		Response: resp.Response{
			Status:  http.StatusOK,
			Message: "успешно",
			Body:    struct{}{},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, "Произошла ошибка", http.StatusInternalServerError)
	}
}

func validateRequest(req *Request) error {
	if req.Text == "" || req.Receivers == nil || len(req.Receivers) == 0 {
		return fmt.Errorf("Текст письма и список получателей не должны быть пустыми")
	}

	if len(req.Topic) > maxTopicLen {
		return fmt.Errorf("Тема слишком длинная (максимум %d символов)", maxTopicLen)
	}
	if len(req.Text) > maxTextLen {
		return fmt.Errorf("Текст слишком длинный (максимум %d символов)", maxTextLen)
	}

	if validation.HasDangerousCharacters(req.Topic) {
		return fmt.Errorf("Тема содержит недопустимые символы")
	}
	if validation.HasDangerousCharacters(req.Text) {
		return fmt.Errorf("Текст содержит недопустимые символы")
	}

	seen := make(map[string]struct{})
	for _, r := range req.Receivers {
		email := strings.TrimSpace(r.Email)
		if email == "" {
			return fmt.Errorf("Email получателя обязателен")
		}
		if _, err := mail.ParseAddress(email); err != nil {
			return fmt.Errorf("Некорректный email получателя: %s", email)
		}
		lower := strings.ToLower(email)
		if _, ok := seen[lower]; ok {
			return fmt.Errorf("Дублирующийся email получателя: %s", email)
		}
		seen[lower] = struct{}{}

		if validation.HasDangerousCharacters(email) {
			return fmt.Errorf("Email получателя содержит недопустимые символы: %s", email)
		}
	}

	if len(req.Files) > defaultLimitFiles {
		return fmt.Errorf("Превышено количество файлов")
	}
	var totalSize int64
	seenPaths := make(map[string]struct{})
	for _, f := range req.Files {
		if f.Size < 0 || f.Size > maxFileSize {
			return fmt.Errorf("Недопустимый размер файла: %s", f.Name)
		}
		totalSize += f.Size
		if _, ok := allowedFileTypes[f.FileType]; !ok {
			return fmt.Errorf("Недопустимый тип файла: %s", f.FileType)
		}
		base := filepath.Base(f.Name)
		if base != f.Name || strings.Contains(f.Name, "..") {
			return fmt.Errorf("Некорректное имя файла: %s", f.Name)
		}
		if validation.HasDangerousCharacters(f.StoragePath) {
			return fmt.Errorf("Недопустимый путь к файлу: %s", f.Name)
		}
		if validation.HasDangerousCharacters(f.Name) {
			return fmt.Errorf("Недопустимое имя файла: %s", f.Name)
		}
		path := strings.TrimSpace(f.StoragePath)
		if path != "" {
			if _, exists := seenPaths[path]; exists {
				return fmt.Errorf("Дублирующийся путь для файла: %s", f.Name)
			}
			seenPaths[path] = struct{}{}
		}
	}

	if totalSize > maxTotalFilesSize {
		return fmt.Errorf("Суммарный размер файлов превышает %d МБ", maxTotalFilesSize/(1024*1024))
	}

	return nil
}
