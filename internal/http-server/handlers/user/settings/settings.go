package settings

//go:generate mockgen -source=$GOFILE -destination=./mocks/mock_profile_usecase.go -package=mocks

import (
	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/http-server/middleware/logger"
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	"context"
	"encoding/json"
	"net/http"
)

type ProfileUsecase interface {
	FindSettingsByProfileId(ctx context.Context, profileID int64) (domain.Settings, error)
}

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
	profileUCase ProfileUsecase
	secret       []byte
}

func New(profileUCase ProfileUsecase, SECRET []byte) *HandlerSettings {
	return &HandlerSettings{
		profileUCase: profileUCase,
		secret:       SECRET,
	}
}

func (h *HandlerSettings) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := logger.GetLogger(r.Context())
	log.Debug("handle user/settings")

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
			Status:  http.StatusOK,
			Message: "success",
			Body:    settingsResponse,
		},
	}

	if err := json.NewEncoder(w).Encode(reponse); err != nil {
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
	}
}
