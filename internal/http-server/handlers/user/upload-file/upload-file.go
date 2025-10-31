package upload_file

import (
	resp "2025_2_a4code/internal/lib/api/response"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
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
}

func New(uploadPath string) (*HandlerUploadFile, error) {
	// проверка существования директории (пока сохраняем просто в директории)
	err := os.MkdirAll(uploadPath, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("failed to create upload directory '%s': %w", uploadPath, err)
	}
	return &HandlerUploadFile{uploadPath: uploadPath}, nil
}

func (h *HandlerUploadFile) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		resp.SendErrorResponse(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		resp.SendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		resp.SendErrorResponse(w, "Unable to save file: "+err.Error(), http.StatusInternalServerError)
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename) // получение расширения

	// Генерация уникального имя файла
	randBytes := make([]byte, 16)
	if _, err := rand.Read(randBytes); err != nil {
		resp.SendErrorResponse(w, "Unable to name file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	uniqName := fmt.Sprintf("%x%s", randBytes, ext)
	storagePath := filepath.Join(h.uploadPath, uniqName)

	dst, err := os.Create(storagePath)
	if err != nil {
		resp.SendErrorResponse(w, "Unable to save file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	size, err := io.Copy(dst, file)
	if err != nil {
		resp.SendErrorResponse(w, "Unable to save file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := Response{
		Response: resp.Response{
			Status: http.StatusText(http.StatusOK),
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
