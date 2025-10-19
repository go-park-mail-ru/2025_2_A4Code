package me

import (
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"2025_2_a4code/internal/usecase/profile"
	"encoding/json"
	"net/http"
	"time"
)

var SECRET = []byte("secret") // TODO: убрать отсюда

type Settings struct {
	NotificationTolerance string
	Language              string
	Theme                 string
	Signature             string
}
type Profile struct {
	Username   string    `json:"username"`
	CreatedAt  time.Time `json:"created_at"`
	Name       string    `json:"name"`
	Surname    string    `json:"surname"`
	Patronymic string    `json:"patronymic"`
	Gender     string    `json:"gender"`
	Birthday   string    `json:"birthday"`
	AvatarPath string    `json:"avatar_path"`
	Settings
}
type Response struct {
	resp.Response
	Body interface{} `json:"body,omitempty"`
}

type HandlerMe struct {
	profileUCase *profile.ProfileUcase
}

func New(ucP *profile.ProfileUcase) *HandlerMe {
	return &HandlerMe{profileUCase: ucP}
}

func (h *HandlerMe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendErrorResponse(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := session.GetProfileID(r, SECRET)
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusUnauthorized)
		return
	}

	profileInfo, err := h.profileUCase.FindInfoByID(id)
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusNotFound)
		return
	}

	profileInfoResponse := Profile{
		Username:   profileInfo.Username,
		CreatedAt:  profileInfo.CreatedAt,
		Name:       profileInfo.Name,
		Surname:    profileInfo.Surname,
		Patronymic: profileInfo.Patronymic,
		Gender:     profileInfo.Gender,
		Birthday:   profileInfo.Birthday,
		AvatarPath: profileInfo.AvatarPath,
	}
	settingsResponse := Settings{
		NotificationTolerance: profileInfo.NotificationTolerance,
		Language:              profileInfo.Language,
		Theme:                 profileInfo.Theme,
		Signature:             profileInfo.Signature,
	}

	response := Response{
		Response: resp.Response{
			Status: http.StatusText(http.StatusOK),
		},
		Body: struct {
			Profile  Profile
			Settings Settings
		}{Profile: profileInfoResponse, Settings: settingsResponse},
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func sendErrorResponse(w http.ResponseWriter, errorMsg string, statusCode int) {

	response := Response{
		Response: resp.Response{
			Status: http.StatusText(statusCode),
			Error:  errorMsg,
		},
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(&response)
}
