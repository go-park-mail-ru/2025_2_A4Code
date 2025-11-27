package gateway_service

import (
	"2025_2_a4code/auth-service/pkg/authproto"
	"2025_2_a4code/internal/config"
	"2025_2_a4code/internal/http-server/middleware/cors"
	"2025_2_a4code/internal/http-server/middleware/logger"
	"2025_2_a4code/internal/http-server/middleware/metrics"
	"2025_2_a4code/messages-service/pkg/messagesproto"
	"2025_2_a4code/profile-service/pkg/profileproto"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Server struct {
	cfg           *config.AppConfig
	httpServer    *http.Server
	authClient    authproto.AuthServiceClient
	profileClient profileproto.ProfileServiceClient
	messageClient messagesproto.MessagesServiceClient
}

type apiResponse struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Body    interface{} `json:"body,omitempty"`
}

type profileDTO struct {
	Username    string `json:"username"`
	CreatedAt   string `json:"created_at"`
	Name        string `json:"name"`
	Surname     string `json:"surname"`
	Patronymic  string `json:"patronymic"`
	Gender      string `json:"gender"`
	DateOfBirth string `json:"date_of_birth"`
	AvatarPath  string `json:"avatar_path"`
	Role        string `json:"role,omitempty"`
}

func NewServer(cfg *config.AppConfig) (*Server, error) {
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	authConn, err := grpc.NewClient(cfg.Host+":"+cfg.AuthPort, opts...)
	if err != nil {
		return nil, err
	}
	profileConn, err := grpc.NewClient(cfg.Host+":"+cfg.ProfilePort, opts...)
	if err != nil {
		return nil, err
	}
	messagesConn, err := grpc.NewClient(cfg.Host+":"+cfg.MessagesPort, opts...)
	if err != nil {
		return nil, err
	}

	return &Server{
		cfg:           cfg,
		authClient:    authproto.NewAuthServiceClient(authConn),
		profileClient: profileproto.NewProfileServiceClient(profileConn),
		messageClient: messagesproto.NewMessagesServiceClient(messagesConn),
	}, nil
}

func (s *Server) Start(ctx context.Context) error {
	log := logger.GetLogger(ctx)
	slog.SetDefault(log)

	// Запуск сервера метрик
	go func() {
		http.Handle("/metrics", promhttp.Handler()) // promhttp экспортирует CPU/Mem автоматически
		metricsAddr := ":" + s.cfg.GatewayMetricsPort
		log.Info("Gateway metrics server started on " + metricsAddr)
		if err := http.ListenAndServe(metricsAddr, nil); err != nil {
			log.Error("Failed to start metrics server: " + err.Error())
		}
	}()

	mux := http.NewServeMux()

	mux.Handle("POST /auth/login", http.HandlerFunc(s.loginHandler))
	mux.Handle("POST /auth/signup", http.HandlerFunc(s.signupHandler))
	mux.Handle("POST /auth/refresh", http.HandlerFunc(s.refreshHandler))
	mux.Handle("POST /auth/logout", http.HandlerFunc(s.logoutHandler))

	mux.Handle("GET /user/profile", http.HandlerFunc(s.getProfileHandler))
	mux.Handle("PUT /user/profile", http.HandlerFunc(s.updateProfileHandler))
	mux.Handle("GET /user/settings", http.HandlerFunc(s.settingsHandler))
	mux.Handle("POST /user/upload/avatar", http.HandlerFunc(s.uploadAvatarHandler))
	mux.Handle("GET /user/avatar", http.HandlerFunc(s.getAvatarHandler))

	mux.Handle("GET /messages/{message_id}", http.HandlerFunc(s.messagePageHandler))
	mux.Handle("POST /messages/reply", http.HandlerFunc(s.replyHandler))
	mux.Handle("POST /messages/send", http.HandlerFunc(s.sendHandler))
	mux.Handle("POST /messages/mark-as-spam", http.HandlerFunc(s.markAsSpamHandler))
	mux.Handle("POST /messages/move-to-folder", http.HandlerFunc(s.moveToFolderHandler))
	mux.Handle("POST /messages/create-folder", http.HandlerFunc(s.createFolderHandler))
	mux.Handle("GET /messages/inbox", http.HandlerFunc(s.inboxHandler))
	mux.Handle("GET /folders/{folder_name}", http.HandlerFunc(s.getFolderHandler))
	mux.Handle("GET /messages/get-folders", http.HandlerFunc(s.getFoldersHandler))
	mux.Handle("PUT /messages/rename-folder", http.HandlerFunc(s.renameFolderHandler))
	mux.Handle("DELETE /messages/delete-folder", http.HandlerFunc(s.deleteFolderHandler))
	mux.Handle("DELETE /messages/delete-message-from-folder", http.HandlerFunc(s.deleteMessageFromFolderHandler))
	mux.Handle("POST /messages/save-draft", http.HandlerFunc(s.saveDraftHandler))
	mux.Handle("DELETE /messages/delete-draft", http.HandlerFunc(s.deleteDraftHandler))
	mux.Handle("POST /messages/send-draft", http.HandlerFunc(s.sendDraftHandler))

	var handler http.Handler = mux
	handler = logger.New(log)(handler)
	handler = cors.New()(handler)
	handler = metrics.Middleware(handler)

	s.httpServer = &http.Server{
		Addr:         ":" + s.cfg.GatewayPort,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Info("Server is listening port: " + s.cfg.GatewayPort)
	if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Error("Server stopped: " + err.Error())
		return err
	}
	return nil
}

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	var req authproto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Invalid request body")
		return
	}

	resp, err := s.authClient.Login(r.Context(), &req)
	if err != nil {
		respondError(w, "Login failed")
		return
	}

	setAuthCookies(w, resp.AccessToken, resp.RefreshToken)
	respondSuccess(w, resp)
}

func (s *Server) signupHandler(w http.ResponseWriter, r *http.Request) {
	var req authproto.SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Invalid request body")
		return
	}

	resp, err := s.authClient.Signup(r.Context(), &req)
	if err != nil {
		if grpcErr, ok := status.FromError(err); ok {
			switch grpcErr.Code() {
			case codes.InvalidArgument, codes.AlreadyExists:
				writeResponse(w, http.StatusBadRequest, "Signup failed: "+grpcErr.Message(), nil)
			default:
				respondError(w, "Signup failed: "+grpcErr.Message())
			}
		} else {
			respondError(w, "Signup failed")
		}
		return
	}

	setAuthCookies(w, resp.AccessToken, resp.RefreshToken)
	respondSuccess(w, resp)
}

func (s *Server) refreshHandler(w http.ResponseWriter, r *http.Request) {
	refreshCookie, err := r.Cookie("refresh_token")
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Refresh token required", nil)
		return
	}

	req := &authproto.RefreshRequest{RefreshToken: refreshCookie.Value}
	resp, err := s.authClient.Refresh(r.Context(), req)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Refresh failed", nil)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    resp.AccessToken,
		MaxAge:   15 * 60,
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: http.SameSiteNoneMode,
	})

	respondSuccess(w, resp)
}

func (s *Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	req := &authproto.LogoutRequest{}
	resp, err := s.authClient.Logout(r.Context(), req)
	if err != nil {
		respondError(w, "Logout failed")
		return
	}

	clearCookie := func(name string) {
		http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteNoneMode})
	}
	clearCookie("access_token")
	clearCookie("refresh_token")

	respondSuccess(w, resp)
}

func (s *Server) getProfileHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Access token required", nil)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)
	resp, err := s.profileClient.GetProfile(ctx, &profileproto.GetProfileRequest{})
	if err != nil {
		writeGrpcAwareError(w, err, "Failed to get profile")
		return
	}

	respondSuccess(w, mapProfile(resp.Profile))
}

func (s *Server) updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Access token required", nil)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req profileproto.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	resp, err := s.profileClient.UpdateProfile(ctx, &req)
	if err != nil {
		writeGrpcAwareError(w, err, "Failed to update profile")
		return
	}

	respondSuccess(w, mapProfile(resp.Profile))
}

func (s *Server) settingsHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Access token required", nil)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)
	resp, err := s.profileClient.Settings(ctx, &profileproto.SettingsRequest{})
	if err != nil {
		writeGrpcAwareError(w, err, "Failed to get settings")
		return
	}

	respondSuccess(w, resp)
}

// Messages handlers

func (s *Server) messagePageHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Access token required", nil)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)
	req := &messagesproto.MessagePageRequest{MessageId: r.PathValue("message_id")}

	resp, err := s.messageClient.MessagePage(ctx, req)
	if err != nil {
		//respondError(w, "Failed to get message")
		writeGrpcAwareError(w, err, "Failed to get message")
		return
	}

	respondSuccess(w, resp)
}

func (s *Server) replyHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Access token required", nil)
		return
	}
	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.ReplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	if _, err := s.messageClient.Reply(ctx, &req); err != nil {
		respondError(w, "Failed to send reply")
		return
	}
	respondSuccess(w, map[string]string{"status": "ok"})
}

func (s *Server) sendHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Access token required", nil)
		return
	}
	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	if _, err := s.messageClient.Send(ctx, &req); err != nil {
		respondError(w, "Failed to send message")
		return
	}
	respondSuccess(w, map[string]string{"status": "ok"})
}

func (s *Server) getFolderHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Access token required", nil)
		return
	}
	ctx := s.addTokenToContext(r.Context(), accessToken)

	folderKey := r.PathValue("folder_name")
	lastMessageID := r.URL.Query().Get("last_message_id")
	lastDatetime := r.URL.Query().Get("last_datetime")
	limit := r.URL.Query().Get("limit")

	foldersResp, err := s.messageClient.GetFolders(ctx, &messagesproto.GetFoldersRequest{})
	if err != nil {
		respondError(w, "Failed to get folders")
		return
	}

	folderID := resolveFolderID(foldersResp.Folders, folderKey)
	if folderID == "" {
		if strings.EqualFold(folderKey, "inbox") {
			s.handleInbox(ctx, w, r)
			return
		}
		writeResponse(w, http.StatusNotFound, "Folder not found", nil)
		return
	}

	s.respondFolder(ctx, w, r, folderID, lastMessageID, lastDatetime, limit, "Failed to get folder")
}

func (s *Server) getFoldersHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Access token required", nil)
		return
	}
	ctx := s.addTokenToContext(r.Context(), accessToken)

	resp, err := s.messageClient.GetFolders(ctx, &messagesproto.GetFoldersRequest{})
	if err != nil {
		respondError(w, "Failed to get folders")
		return
	}
	respondSuccess(w, resp)
}

func (s *Server) inboxHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Access token required", nil)
		return
	}
	ctx := s.addTokenToContext(r.Context(), accessToken)
	s.handleInbox(ctx, w, r)
}

func (s *Server) handleInbox(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	foldersResp, err := s.messageClient.GetFolders(ctx, &messagesproto.GetFoldersRequest{})
	if err != nil {
		respondError(w, "Failed to get folders")
		return
	}

	folderID := resolveFolderID(foldersResp.Folders, "inbox")
	if folderID == "" {
		respondSuccess(w, emptyFolderResponse())
		return
	}

	s.respondFolder(ctx, w, r, folderID, r.URL.Query().Get("last_message_id"), r.URL.Query().Get("last_datetime"), r.URL.Query().Get("limit"), "Failed to get inbox")
}

func (s *Server) respondFolder(ctx context.Context, w http.ResponseWriter, r *http.Request, folderID, lastMessageID, lastDatetime, limit, errorMessage string) {
	req := &messagesproto.GetFolderRequest{
		FolderId:      folderID,
		LastMessageId: lastMessageID,
		LastDatetime:  lastDatetime,
		Limit:         limit,
	}

	resp, err := s.messageClient.GetFolder(ctx, req)
	if err != nil {
		respondSuccess(w, emptyFolderResponse())
		return
	}

	respondSuccess(w, resp)
}

// Helpers

func emptyFolderResponse() *messagesproto.GetFolderResponse {
	return &messagesproto.GetFolderResponse{
		MessageTotal:  "0",
		MessageUnread: "0",
		Messages:      []*messagesproto.Message{},
		Pagination:    &messagesproto.PaginationInfo{HasNext: "false"},
	}
}

func (s *Server) renameFolderHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Access token required", nil)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.RenameFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	resp, err := s.messageClient.RenameFolder(ctx, &req)
	if err != nil {
		writeGrpcAwareError(w, err, "Failed to rename folder")
		return
	}

	respondSuccess(w, resp)
}

func (s *Server) deleteFolderHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Access token required", nil)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	folderID := r.URL.Query().Get("folder_id")
	req := &messagesproto.DeleteFolderRequest{FolderId: folderID}

	resp, err := s.messageClient.DeleteFolder(ctx, req)
	if err != nil {
		respondError(w, "Failed to delete folder")
		return
	}

	respondSuccess(w, resp)
}

func (s *Server) deleteMessageFromFolderHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Access token required", nil)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.DeleteMessageFromFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	resp, err := s.messageClient.DeleteMessageFromFolder(ctx, &req)
	if err != nil {
		respondError(w, "Failed to delete message from folder")
		return
	}

	respondSuccess(w, resp)
}

func (s *Server) saveDraftHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Access token required", nil)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.SaveDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	resp, err := s.messageClient.SaveDraft(ctx, &req)
	if err != nil {
		respondError(w, "Failed to save draft")
		return
	}

	respondSuccess(w, resp)
}

func (s *Server) deleteDraftHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Access token required", nil)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.DeleteDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	resp, err := s.messageClient.DeleteDraft(ctx, &req)
	if err != nil {
		respondError(w, "Failed to delete draft")
		return
	}

	respondSuccess(w, resp)
}

func (s *Server) sendDraftHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Access token required", nil)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.SendDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	resp, err := s.messageClient.SendDraft(ctx, &req)
	if err != nil {
		respondError(w, "Failed to send draft")
		return
	}

	respondSuccess(w, resp)
}

func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// Ancillary handlers
func (s *Server) uploadAvatarHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Access token required", nil)
		return
	}
	ctx := s.addTokenToContext(r.Context(), accessToken)

	if err := r.ParseMultipartForm(5 << 20); err != nil {
		writeResponse(w, http.StatusBadRequest, "File too large", nil)
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "No avatar file provided", nil)
		return
	}
	defer file.Close()

	read, err := io.ReadAll(file)
	if err != nil {
		respondError(w, "Failed to read file")
		return
	}

	req := &profileproto.UploadAvatarRequest{
		AvatarData:  read,
		FileName:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
	}

	resp, err := s.profileClient.UploadAvatar(ctx, req)
	if err != nil {
		writeGrpcAwareError(w, err, "Failed to upload avatar")
		return
	}

	respondSuccess(w, resp)
}

func (s *Server) getAvatarHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Access token required", nil)
		return
	}
	ctx := s.addTokenToContext(r.Context(), accessToken)

	targetURL := strings.TrimSpace(r.URL.Query().Get("url"))
	if targetURL == "" {
		profileResp, err := s.profileClient.GetProfile(ctx, &profileproto.GetProfileRequest{})
		if err != nil {
			writeGrpcAwareError(w, err, "Failed to get profile")
			return
		}
		targetURL = strings.TrimSpace(profileResp.GetProfile().GetAvatarPath())
	}

	if targetURL == "" {
		writeResponse(w, http.StatusNotFound, "Avatar not found", nil)
		return
	}

	resolvedURL, err := normalizeAvatarURL(targetURL)
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid avatar url", nil)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resolvedURL, nil)
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Failed to request avatar", nil)
		return
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil || resp == nil {
		writeResponse(w, http.StatusBadGateway, "Failed to fetch avatar", nil)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		writeResponse(w, http.StatusBadGateway, "Failed to fetch avatar", nil)
		return
	}

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, resp.Body)
}

func normalizeAvatarURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}

	parsed.Scheme = "http"

	host := parsed.Hostname()
	if host == "" || host == "127.0.0.1" || host == "localhost" {
		parsed.Host = "minio:9000"
	}

	return parsed.String(), nil
}

func (s *Server) markAsSpamHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Access token required", nil)
		return
	}
	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.MarkAsSpamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	resp, err := s.messageClient.MarkAsSpam(ctx, &req)
	if err != nil {
		respondError(w, "Failed to mark as spam")
		return
	}

	respondSuccess(w, resp)
}

func (s *Server) moveToFolderHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Access token required", nil)
		return
	}
	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.MoveToFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	resp, err := s.messageClient.MoveToFolder(ctx, &req)
	if err != nil {
		respondError(w, "Failed to move to folder")
		return
	}

	respondSuccess(w, resp)
}

func (s *Server) createFolderHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Access token required", nil)
		return
	}
	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.CreateFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	resp, err := s.messageClient.CreateFolder(ctx, &req)
	if err != nil {
		writeGrpcAwareError(w, err, "Failed to create folder")
		return
	}

	respondSuccess(w, resp)
}

func respondSuccess(w http.ResponseWriter, body interface{}) {
	writeResponse(w, http.StatusOK, "success", body)
}

func respondError(w http.ResponseWriter, message string) {
	writeResponse(w, http.StatusInternalServerError, message, nil)
}

func writeResponse(w http.ResponseWriter, status int, message string, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	code := http.StatusOK
	if status >= 400 {
		code = status
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(apiResponse{
		Status:  status,
		Message: message,
		Body:    body,
	})
}

func writeGrpcAwareError(w http.ResponseWriter, err error, defaultMessage string) {
	if grpcStatus, ok := status.FromError(err); ok {
		switch grpcStatus.Code() {
		case codes.Unauthenticated:
			writeResponse(w, http.StatusUnauthorized, defaultMessage, nil)
			return
		case codes.InvalidArgument:
			writeResponse(w, http.StatusBadRequest, defaultMessage, nil)
			return
		case codes.NotFound:
			writeResponse(w, http.StatusNotFound, defaultMessage, nil)
			return
		case codes.PermissionDenied:
			writeResponse(w, http.StatusForbidden, defaultMessage, nil)
			return
		case codes.AlreadyExists:
			msg := grpcStatus.Message()
			if strings.TrimSpace(msg) == "" {
				msg = defaultMessage
			}
			writeResponse(w, http.StatusConflict, msg, nil)
			return
		}
	}
	respondError(w, defaultMessage)
}

func mapProfile(profile *profileproto.Profile) profileDTO {
	if profile == nil {
		return profileDTO{}
	}
	return profileDTO{
		Username:    profile.Username,
		CreatedAt:   profile.CreatedAt,
		Name:        profile.Name,
		Surname:     profile.Surname,
		Patronymic:  profile.Patronymic,
		Gender:      profile.Gender,
		DateOfBirth: profile.Birthday,
		AvatarPath:  profile.AvatarPath,
		Role:        "user",
	}
}

func resolveFolderID(folders []*messagesproto.Folder, target string) string {
	raw := strings.TrimSpace(target)
	lowerTarget := strings.ToLower(raw)

	for _, folder := range folders {
		if raw != "" && folder.FolderId == raw {
			return folder.FolderId
		}

		name := strings.ToLower(strings.TrimSpace(folder.FolderName))
		ftype := strings.ToLower(strings.TrimSpace(folder.FolderType))
		if lowerTarget != "" && (name == lowerTarget || ftype == lowerTarget) {
			return folder.FolderId
		}
	}
	return ""
}

func getAccessToken(r *http.Request) (string, error) {
	cookie, err := r.Cookie("access_token")
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

func (s *Server) addTokenToContext(ctx context.Context, token string) context.Context {
	md := metadata.Pairs("authorization", "Bearer "+token)
	return metadata.NewOutgoingContext(ctx, md)
}

func setAuthCookies(w http.ResponseWriter, access, refresh string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    access,
		MaxAge:   15 * 60,
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: http.SameSiteNoneMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refresh,
		MaxAge:   7 * 24 * 3600,
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: http.SameSiteNoneMode,
	})
}
