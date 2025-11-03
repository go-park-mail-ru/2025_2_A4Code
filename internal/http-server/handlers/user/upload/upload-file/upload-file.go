package upload_file

import (
	resp "2025_2_a4code/internal/lib/api/response"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

type Response struct {
	resp.Response
}

type File struct {
	FileType    string `json:"file_type,omitempty"`
	Size        int64  `json:"size,omitempty"`
	StoragePath string `json:"storage_path,omitempty"`
}

type HandlerUploadFile struct {
	uploadPath string // куда сохраняем
	log        *slog.Logger
}

func New(uploadPath string, log *slog.Logger) (*HandlerUploadFile, error) {
	// проверка существования директории (пока сохраняем просто в директории)
	err := os.MkdirAll(uploadPath, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("failed to create upload directory '%s': %w", uploadPath, err)
	}
	return &HandlerUploadFile{uploadPath: uploadPath, log: log}, nil
}

func (h *HandlerUploadFile) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := h.log
	log.Info("handle upload file")

	if r.Method != http.MethodPost {
		resp.SendErrorResponse(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename) // получение расширения

	// Генерация уникального имя файла
	randBytes := make([]byte, 16)
	if _, err := rand.Read(randBytes); err != nil {
		log.Error("unable to name file: " + err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}
	uniqName := fmt.Sprintf("%x%s", randBytes, ext)
	storagePath := filepath.Join(h.uploadPath, uniqName)

	dst, err := os.Create(storagePath)
	if err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	size, err := io.Copy(dst, file)
	if err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	response := Response{
		Response: resp.Response{
			Status:  http.StatusOK,
			Message: "success",
			Body: File{
				FileType:    header.Header.Get("Content-Type"),
				Size:        size,
				StoragePath: storagePath,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
