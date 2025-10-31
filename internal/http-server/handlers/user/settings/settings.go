package settings

import (
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"2025_2_a4code/internal/usecase/profile"
	"encoding/json"
	"net/http"
)

type Settings struct {
	NotificationTolerance string   `json:"notification_tolerance"`
	Language              string   `json:"language"`
	Theme                 string   `json:"theme"`
	Signatures            []string `json:"signatures"`
}

type Response struct {
	resp.Response
}

type Signatures []string

type HandlerSettings struct {
	profileUCase *profile.ProfileUcase
	secret       []byte
}

func New(profileUCase *profile.ProfileUcase, SECRET []byte) *HandlerSettings {
	return &HandlerSettings{
		profileUCase: profileUCase,
		secret:       SECRET,
	}
}

func (h *HandlerSettings) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		resp.SendErrorResponse(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	id, err := session.GetProfileID(r, h.secret)

	settings, err := h.profileUCase.FindSettingsByProfileId(id)
	if err != nil {
		resp.SendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	settingsResponse := Settings{ // TODO: поменять наполнение
		NotificationTolerance: settings.NotificationTolerance,
		Language:              settings.Language,
		Theme:                 settings.Theme,
		Signatures:            settings.Signatures,
	}

	reponse := Response{
		Response: resp.Response{
			Status:  http.StatusText(http.StatusOK),
			Message: "Настройки получены",
			Body:    settingsResponse,
		},
	}

	if err := json.NewEncoder(w).Encode(reponse); err != nil {
		resp.SendErrorResponse(w, err.Error(), http.StatusInternalServerError)
	}
}
