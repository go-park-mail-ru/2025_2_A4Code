package upload_avatar

import (
	"2025_2_a4code/internal/http-server/middleware/logger"
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"2025_2_a4code/internal/usecase/avatar"
	"2025_2_a4code/internal/usecase/profile"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// Максимальный размер загружаемого файла - 5 Мб
const maxAvatarSize = 5 << 20

type Response struct {
	resp.Response
}

type HandlerUploadAvatar struct {
	avatarUcase  *avatar.AvatarUcase
	profileUcase profile.ProfileUsecase
	secret       []byte
}

func New(avatarUcase *avatar.AvatarUcase, profileUcase profile.ProfileUsecase, secret []byte) *HandlerUploadAvatar {
	return &HandlerUploadAvatar{
		avatarUcase:  avatarUcase,
		profileUcase: profileUcase,
		secret:       secret,
	}
}

func (h *HandlerUploadAvatar) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := logger.GetLogger(r.Context())
	log.Info("Handling user/upload/avatar")

	if r.Method != http.MethodPost {
		resp.SendErrorResponse(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	id, err := session.GetProfileID(r, h.secret)
	if err != nil {
		resp.SendErrorResponse(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarSize)
	err = r.ParseMultipartForm(maxAvatarSize)
	if err != nil {
		log.Error("Error parsing avatar form: " + err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		log.Error("Error getting file from form: " + err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
	defer cancel()

	stringId := strconv.Itoa(int(id))
	objectName, presignedURL, err := h.avatarUcase.UploadAvatar(ctx, stringId, file, header.Size, header.Filename)
	if err != nil {
		log.Error("Error uploading avatar: " + err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	err = h.profileUcase.InsertProfileAvatar(ctx, id, objectName)
	if err != nil {
		log.Error("Error inserting avatar: " + err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := Response{
		resp.Response{
			Status:  http.StatusOK,
			Message: "success",
			Body: struct {
				AvatarPath string `json:"avatar_path"`
			}{
				AvatarPath: presignedURL,
			},
		},
	}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error("Error encoding response: " + err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}
}
