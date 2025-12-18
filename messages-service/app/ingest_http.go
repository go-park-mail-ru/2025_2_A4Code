package app

import (
	"2025_2_a4code/internal/http-server/middleware/logger"
	"2025_2_a4code/internal/lib/validation"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

type ingestHandler struct {
	secret        string
	messageUCase  ingestMessageUsecase
	logger        *slog.Logger
}

type ingestMessageUsecase interface {
	EnsureBaseProfile(ctx context.Context, username, domain string) (int64, error)
	EnsureProfileForBase(ctx context.Context, baseProfileID int64, displayName string) error
	SaveMessage(ctx context.Context, receiverProfileEmail string, senderProfileID int64, topic, text string) (int64, error)
	SaveFile(ctx context.Context, messageID int64, fileName, fileType, storagePath string, size int64) (fileID int64, err error)
	SaveThread(ctx context.Context, messageID int64) (threadID int64, err error)
	SaveThreadIdToMessage(ctx context.Context, messageID int64, threadID int64) error
}

type ingestRequest struct {
	SenderEmail string       `json:"sender_email"`
	SenderName  string       `json:"sender_name"`
	Receivers   []string     `json:"receivers"`
	Subject     string       `json:"subject"`
	Text        string       `json:"text"`
	Files       []ingestFile `json:"files"`
}

type ingestFile struct {
	Name        string `json:"name"`
	FileType    string `json:"file_type"`
	Size        int64  `json:"size"`
	StoragePath string `json:"storage_path"`
}

type ingestResponse struct {
	MessageIDs []string `json:"message_ids"`
}

func newIngestHandler(secret string, uc ingestMessageUsecase, log *slog.Logger) http.Handler {
	return &ingestHandler{
		secret:       strings.TrimSpace(secret),
		messageUCase: uc,
		logger:       log,
	}
}

func (h *ingestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := logger.GetLogger(r.Context())

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if h.secret != "" {
		got := strings.TrimSpace(r.Header.Get("X-Ingest-Secret"))
		if got == "" || got != h.secret {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	var req ingestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.SenderEmail) == "" || len(req.Receivers) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	senderUser, senderDomain, err := splitEmail(req.SenderEmail)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	senderProfileID, err := h.messageUCase.EnsureBaseProfile(r.Context(), senderUser, senderDomain)
	if err != nil {
		log.Error("failed to ensure sender profile", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := h.messageUCase.EnsureProfileForBase(r.Context(), senderProfileID, req.SenderName); err != nil {
		log.Warn("failed to update sender display name", "err", err)
	}

	safeSubject, safeText := sanitizeIngestContent(req.Subject, req.Text)

	var messageIDs []string
	for _, rcpt := range dedupeEmails(req.Receivers) {
		msgID, err := h.messageUCase.SaveMessage(r.Context(), rcpt, senderProfileID, safeSubject, safeText)
		if err != nil {
			log.Error("failed to save message", "receiver", rcpt, "err", err)
			continue
		}

		threadID, err := h.messageUCase.SaveThread(r.Context(), msgID)
		if err != nil {
			log.Warn("failed to create thread", "err", err)
		} else {
			if err := h.messageUCase.SaveThreadIdToMessage(r.Context(), msgID, threadID); err != nil {
				log.Warn("failed to attach thread id", "err", err)
			}
		}

		for _, f := range req.Files {
			if err := validateIngestFile(f); err != nil {
				log.Warn("skip invalid file", "err", err, "name", f.Name)
				continue
			}
			if _, err := h.messageUCase.SaveFile(r.Context(), msgID, f.Name, f.FileType, f.StoragePath, f.Size); err != nil {
				log.Warn("failed to save file", "err", err, "name", f.Name)
				continue
			}
		}

		messageIDs = append(messageIDs, strconv.FormatInt(msgID, 10))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ingestResponse{MessageIDs: messageIDs})
}

func sanitizeIngestContent(topic, text string) (string, string) {
	return html.EscapeString(topic), html.EscapeString(text)
}

func dedupeEmails(list []string) []string {
	seen := make(map[string]struct{})
	var res []string
	for _, raw := range list {
		email := strings.ToLower(strings.TrimSpace(raw))
		if email == "" {
			continue
		}
		if _, ok := seen[email]; ok {
			continue
		}
		seen[email] = struct{}{}
		res = append(res, email)
	}
	return res
}

func splitEmail(email string) (string, string, error) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", "", errors.New("invalid email")
	}
	user := strings.TrimSpace(parts[0])
	domain := strings.TrimSpace(parts[1])
	if user == "" || domain == "" || validation.HasDangerousCharacters(email) {
		return "", "", errors.New("invalid email")
	}
	return user, domain, nil
}

func validateIngestFile(f ingestFile) error {
	if strings.TrimSpace(f.StoragePath) == "" || strings.Contains(f.StoragePath, "..") {
		return fmt.Errorf("invalid path")
	}
	if strings.TrimSpace(f.Name) == "" {
		return fmt.Errorf("empty name")
	}
	if f.Size < 0 {
		return fmt.Errorf("invalid size")
	}
	return nil
}
