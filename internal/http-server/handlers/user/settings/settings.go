package settings

import (
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"2025_2_a4code/internal/usecase/profile"
	"encoding/json"
	"log/slog"
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
	log          *slog.Logger
}

func New(profileUCase *profile.ProfileUcase, SECRET []byte, log *slog.Logger) *HandlerSettings {
	return &HandlerSettings{
		profileUCase: profileUCase,
		secret:       SECRET,
		log:          log,
	}
}

func (h *HandlerSettings) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := h.log
	log.Info("handle settings")

	if r.Method != http.MethodGet {
		resp.SendErrorResponse(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	id, err := session.GetProfileID(r, h.secret)
	if err != nil {
		resp.SendErrorResponse(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	settings, err := h.profileUCase.FindSettingsByProfileId(r.Context(), id)
	if err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	settingsResponse := Settings{
		NotificationTolerance: settings.NotificationTolerance,
		Language:              settings.Language,
		Theme:                 settings.Theme,
		Signatures:            settings.Signatures,
	}

	reponse := Response{
		Response: resp.Response{
			Status:  http.StatusText(http.StatusOK),
			Message: "success",
			Body:    settingsResponse,
		},
	}

	if err := json.NewEncoder(w).Encode(reponse); err != nil {
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
	}
}
