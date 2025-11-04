package profile_page

import (
	"2025_2_a4code/internal/domain"
	resp "2025_2_a4code/internal/lib/api/response"
	"2025_2_a4code/internal/lib/session"
	avatar "2025_2_a4code/internal/usecase/avatar"
	"2025_2_a4code/internal/usecase/profile"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
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

type UpdateProfileRequest struct {
	Name       string `json:"name"`
	Surname    string `json:"surname"`
	Patronymic string `json:"patronymic"`
	Gender     string `json:"gender"`
	Birthday   string `json:"date_of_birth"`
}

type HandlerProfile struct {
	profileUCase *profile.ProfileUcase
	avatarUCase  *avatar.AvatarUcase
	secret       []byte
	log          *slog.Logger
}

func New(profileUCase *profile.ProfileUcase, avatarUCase *avatar.AvatarUcase, SECRET []byte, log *slog.Logger) *HandlerProfile {
	return &HandlerProfile{
		profileUCase: profileUCase,
		avatarUCase:  avatarUCase,
		secret:       SECRET,
		log:          log,
	}
}

func (h *HandlerProfile) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r)
	case http.MethodPut:
		h.handleUpdate(w, r)
	default:
		resp.SendErrorResponse(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func (h *HandlerProfile) handleGet(w http.ResponseWriter, r *http.Request) {
	log := h.log
	log.Info("handle profile page (GET)")

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

	if err := h.enrichAvatarURL(r.Context(), &profileInfo); err != nil {
		log.Warn("failed to enrich avatar url: " + err.Error())
	}

	h.writeProfileResponse(w, profileInfo)
}

func (h *HandlerProfile) handleUpdate(w http.ResponseWriter, r *http.Request) {
	log := h.log
	log.Info("handle profile page (PUT)")

	id, err := session.GetProfileID(r, h.secret)
	if err != nil {
		resp.SendErrorResponse(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	var req UpdateProfileRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		resp.SendErrorResponse(w, "invalid request body", http.StatusBadRequest)
		return
	}

	updateReq := profile.UpdateProfileRequest{
		FirstName:  req.Name,
		LastName:   req.Surname,
		MiddleName: req.Patronymic,
		Gender:     req.Gender,
		Birthday:   req.Birthday,
	}

	if err := h.profileUCase.UpdateProfileInfo(r.Context(), id, updateReq); err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	profileInfo, err := h.profileUCase.FindInfoByID(r.Context(), id)
	if err != nil {
		log.Error(err.Error())
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	if err := h.enrichAvatarURL(r.Context(), &profileInfo); err != nil {
		log.Warn("failed to enrich avatar url: " + err.Error())
	}

	h.writeProfileResponse(w, profileInfo)
}

func (h *HandlerProfile) enrichAvatarURL(ctx context.Context, profileInfo *domain.ProfileInfo) error {
	if profileInfo.AvatarPath == "" {
		return nil
	}

	objectName := profileInfo.AvatarPath
	if strings.HasPrefix(profileInfo.AvatarPath, "http://") || strings.HasPrefix(profileInfo.AvatarPath, "https://") {
		parsed, err := url.Parse(profileInfo.AvatarPath)
		if err != nil {
			return err
		}
		objectName = strings.TrimPrefix(parsed.Path, "/")
	}

	objectName = strings.TrimLeft(objectName, "/")
	if objectName == "" {
		return nil
	}

	if idx := strings.Index(objectName, "/"); idx != -1 {
		prefix := objectName[:idx]
		if strings.EqualFold(prefix, "avatars") {
			objectName = objectName[idx+1:]
		}
	}

	if objectName == "" {
		return nil
	}

	presignedURL, err := h.avatarUCase.GetAvatarPresignedURL(ctx, objectName, 15*time.Minute)
	if err != nil {
		return err
	}

	profileInfo.AvatarPath = presignedURL.String()

	if !strings.HasPrefix(profileInfo.AvatarPath, "http") {
		profileInfo.AvatarPath = presignedURL.String()
	}

	return nil
}

func (h *HandlerProfile) writeProfileResponse(w http.ResponseWriter, profileInfo domain.ProfileInfo) {
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

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}
}
