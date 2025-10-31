package profile_page

import (
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"2025_2_a4code/internal/usecase/profile"
	"encoding/json"
	"net/http"
	"time"
)

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
}

type HandlerMe struct {
	profileUCase *profile.ProfileUcase
	secret       []byte
}

func New(ucP *profile.ProfileUcase, SECRET []byte) *HandlerMe {
	return &HandlerMe{profileUCase: ucP,
		secret: SECRET,
	}
}

func (h *HandlerMe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		resp.SendErrorResponse(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	id, err := session.GetProfileID(r, h.secret)
	if err != nil {
		resp.SendErrorResponse(w, err.Error(), http.StatusUnauthorized)
		return
	}

	profileInfo, err := h.profileUCase.FindInfoByID(r.Context(), id)
	if err != nil {
		resp.SendErrorResponse(w, err.Error(), http.StatusNotFound)
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
			Body:    profileInfoResponse,
		},
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		resp.SendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
