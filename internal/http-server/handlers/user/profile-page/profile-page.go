package profile_page

import (
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"2025_2_a4code/internal/usecase/profile"
	"encoding/json"
	"log/slog"
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

type HandlerProfile struct {
	profileUCase *profile.ProfileUcase
	secret       []byte
	log          *slog.Logger
}

func New(profileUCase *profile.ProfileUcase, SECRET []byte, log *slog.Logger) *HandlerProfile {
	return &HandlerProfile{
		profileUCase: profileUCase,
		secret:       SECRET,
		log:          log,
	}
}

func (h *HandlerProfile) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := h.log
	log.Info("handle profile page")

	if r.Method != http.MethodGet {
		resp.SendErrorResponse(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	id, err := session.GetProfileID(r, h.secret)
	if err != nil {
		resp.SendErrorResponse(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	profileInfo, err := h.profileUCase.FindInfoByID(r.Context(), id)
	if err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
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
			Status:  http.StatusOK,
			Message: "success",
			Body:    profileInfoResponse,
		},
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}
}
