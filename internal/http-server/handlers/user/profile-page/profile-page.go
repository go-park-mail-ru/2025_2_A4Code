package profile_page

import (
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"2025_2_a4code/internal/usecase/profile"
	"encoding/json"
	"net/http"
	"time"
)

var SECRET = []byte("secret") // TODO: убрать отсюда

type Profile struct {
	Username   string    `json:"username"`
	CreatedAt  time.Time `json:"created_at"`
	Name       string    `json:"name"`
	Surname    string    `json:"surname"`
	Patronymic string    `json:"patronymic"`
	Gender     string    `json:"gender"`
	Birthday   string    `json:"date_of_birth"`
	AvatarPath string    `json:"avatar_path"`
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
		sendErrorResponse(w, "Метод не разрешен", http.StatusMethodNotAllowed)
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

	response := Response{
		Response: resp.Response{
			Status:  http.StatusText(http.StatusOK),
			Message: "Страница пользователя получена",
		},
		Body: struct {
			Profile Profile
		}{Profile: profileInfoResponse},
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
			Status:  http.StatusText(statusCode),
			Message: "Ошибка: " + errorMsg,
		},
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(&response)
}
