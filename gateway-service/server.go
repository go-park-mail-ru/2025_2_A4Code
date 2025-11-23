package gateway_service

import (
	"2025_2_a4code/internal/config"
	"2025_2_a4code/internal/http-server/middleware/cors"
	"2025_2_a4code/internal/http-server/middleware/logger"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"2025_2_a4code/auth-service/pkg/authproto"
	"2025_2_a4code/messages-service/pkg/messagesproto"
	"2025_2_a4code/profile-service/pkg/profileproto"
)

type Server struct {
	cfg           *config.AppConfig
	httpServer    *http.Server
	authClient    authproto.AuthServiceClient
	profileClient profileproto.ProfileServiceClient
	messageClient messagesproto.MessagesServiceClient
}

func NewServer(cfg *config.AppConfig) (*Server, error) {
	// Создаем gRPC соединение
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn1, err := grpc.NewClient(cfg.Host+":"+cfg.AuthPort, opts...)
	if err != nil {
		return nil, err
	}
	authClient := authproto.NewAuthServiceClient(conn1)

	conn2, err := grpc.NewClient(cfg.Host+":"+cfg.ProfilePort, opts...)
	if err != nil {
		return nil, err
	}
	profileClient := profileproto.NewProfileServiceClient(conn2)

	conn3, err := grpc.NewClient(cfg.Host+":"+cfg.MessagesPort, opts...)
	if err != nil {
		return nil, err
	}
	messagesClient := messagesproto.NewMessagesServiceClient(conn3)

	return &Server{
		cfg:           cfg,
		authClient:    authClient,
		profileClient: profileClient,
		messageClient: messagesClient,
	}, nil
}

func (s *Server) Start(ctx context.Context) error {
	log := logger.GetLogger(ctx)
	slog.SetDefault(log)

	// Создаем роутер
	mux := http.NewServeMux()

	mux.Handle("POST /auth/login", http.HandlerFunc(s.loginHandler))
	mux.Handle("POST /auth/signup", http.HandlerFunc(s.signupHandler))

	// Роутер с csrf middleware
	//csrfMux := http.NewServeMux()

	mux.Handle("POST /auth/refresh", http.HandlerFunc(s.refreshHandler))
	mux.Handle("POST /auth/logout", http.HandlerFunc(s.logoutHandler))

	mux.Handle("GET /user/profile", http.HandlerFunc(s.getProfileHandler))
	mux.Handle("PUT /user/profile", http.HandlerFunc(s.updateProfileHandler))
	mux.Handle("GET /user/settings", http.HandlerFunc(s.settingsHandler))
	mux.Handle("POST /user/upload/avatar", http.HandlerFunc(s.uploadAvatarHandler))

	mux.Handle("GET /messages/{message_id}", http.HandlerFunc(s.messagePageHandler))
	mux.Handle("POST /messages/reply", http.HandlerFunc(s.replyHandler))
	mux.Handle("POST /messages/send", http.HandlerFunc(s.sendHandler))
	mux.Handle("POST /messages/mark-as-spam", http.HandlerFunc(s.markAsSpamHandler))
	mux.Handle("POST /messages/move-to-folder", http.HandlerFunc(s.moveToFolderHandler))
	mux.Handle("POST /messages/create-folder", http.HandlerFunc(s.createFolderHandler))
	mux.Handle("GET /folders/{folder_name}", http.HandlerFunc(s.getFolderHandler))
	mux.Handle("GET /messages/get-folders", http.HandlerFunc(s.getFoldersHandler))
	mux.Handle("PUT /messages/rename-folder", http.HandlerFunc(s.renameFolderHandler))
	mux.Handle("DELETE /messages/delete-folder", http.HandlerFunc(s.deleteFolderHandler))
	mux.Handle("DELETE /messages/delete-message-from-folder", http.HandlerFunc(s.deleteMessageFromFolderHandler))
	mux.Handle("POST /messages/save-draft", http.HandlerFunc(s.saveDraftHandler))
	mux.Handle("DELETE /messages/delete-draft", http.HandlerFunc(s.deleteDraftHandler))
	mux.Handle("POST /messages/send-draft", http.HandlerFunc(s.sendDraftHandler))

	// csrf middleware
	//csrfProtectedHandler := csrfcheck.New()(csrfMux)
	//
	//mux.Handle("POST /auth/refresh", csrfProtectedHandler)
	//mux.Handle("POST /auth/logout", csrfProtectedHandler)
	//
	//mux.Handle("GET /user/profile", csrfProtectedHandler)
	//mux.Handle("PUT /user/profile", csrfProtectedHandler)
	//mux.Handle("GET /user/settings", csrfProtectedHandler)
	//mux.Handle("POST /user/upload/avatar", csrfProtectedHandler)
	//
	//mux.Handle("GET /messages/inbox", csrfProtectedHandler)
	//mux.Handle("GET /messages/{message_id}", csrfProtectedHandler)
	//mux.Handle("POST /messages/reply", csrfProtectedHandler)
	//mux.Handle("POST /messages/send", csrfProtectedHandler)
	//mux.Handle("GET /messages/sent", csrfProtectedHandler)

	// logger и cors middleware
	var handler http.Handler = mux
	handler = logger.New(log)(handler)
	handler = cors.New()(handler)

	// Настройка сервера
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
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := s.authClient.Login(r.Context(), &req)
	if err != nil {
		http.Error(w, "Login failed", http.StatusUnauthorized)
		return
	}

	setAuthCookies(w, resp.AccessToken, resp.RefreshToken)

	w.Header().Set("Content-Type", "application/json")
}

func (s *Server) signupHandler(w http.ResponseWriter, r *http.Request) {
	var req authproto.SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := s.authClient.Signup(r.Context(), &req)
	if err != nil {
		if grpcErr, ok := status.FromError(err); ok {
			switch grpcErr.Code() {
			case codes.InvalidArgument:
				http.Error(w, "Signup failed: "+grpcErr.Message(), http.StatusBadRequest)
			case codes.AlreadyExists:
				http.Error(w, "Signup failed: "+grpcErr.Message(), http.StatusConflict)
			default:
				http.Error(w, "Signup failed: "+grpcErr.Message(), http.StatusInternalServerError)
			}
		} else {
			http.Error(w, "Signup failed", http.StatusBadRequest)
		}
		return
	}

	setAuthCookies(w, resp.AccessToken, resp.RefreshToken)

	w.Header().Set("Content-Type", "application/json")
}

func (s *Server) refreshHandler(w http.ResponseWriter, r *http.Request) {
	refreshCookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "Refresh token required", http.StatusBadRequest)
		return
	}

	req := &authproto.RefreshRequest{
		RefreshToken: refreshCookie.Value,
	}

	resp, err := s.authClient.Refresh(r.Context(), req)
	if err != nil {
		http.Error(w, "Refresh failed", http.StatusUnauthorized)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    resp.AccessToken,
		MaxAge:   15 * 60, // 15 минут
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
}

func (s *Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	req := &authproto.LogoutRequest{}

	resp, err := s.authClient.Logout(r.Context(), req)
	if err != nil {
		http.Error(w, "Logout failed", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Profile handlers

func (s *Server) getProfileHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		http.Error(w, "Access token required", http.StatusUnauthorized)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)
	req := &profileproto.GetProfileRequest{}
	resp, err := s.profileClient.GetProfile(ctx, req)
	if err != nil {
		http.Error(w, "Failed to get profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		http.Error(w, "Access token required", http.StatusUnauthorized)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req profileproto.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := s.profileClient.UpdateProfile(ctx, &req)
	if err != nil {
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) settingsHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		http.Error(w, "Access token required", http.StatusUnauthorized)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)
	req := &profileproto.SettingsRequest{}
	resp, err := s.profileClient.Settings(ctx, req)
	if err != nil {
		http.Error(w, "Failed to get settings", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) uploadAvatarHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		http.Error(w, "Access token required", http.StatusUnauthorized)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	// Парсим multipart форму
	if err := r.ParseMultipartForm(5 << 20); err != nil { // 5MB
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		http.Error(w, "No avatar file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Читаем файл в байты
	fileData := make([]byte, header.Size)
	if _, err := file.Read(fileData); err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	req := &profileproto.UploadAvatarRequest{
		AvatarData:  fileData,
		FileName:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
	}

	resp, err := s.profileClient.UploadAvatar(ctx, req)
	if err != nil {
		http.Error(w, "Failed to upload avatar", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Messages handlers

func (s *Server) messagePageHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		http.Error(w, "Access token required", http.StatusUnauthorized)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	messageID := r.PathValue("message_id")
	req := &messagesproto.MessagePageRequest{
		MessageId: messageID,
	}

	resp, err := s.messageClient.MessagePage(ctx, req)
	if err != nil {
		http.Error(w, "Failed to get message", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) replyHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		http.Error(w, "Access token required", http.StatusUnauthorized)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.ReplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := s.messageClient.Reply(ctx, &req)
	if err != nil {
		http.Error(w, "Failed to send reply", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) sendHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		http.Error(w, "Access token required", http.StatusUnauthorized)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := s.messageClient.Send(ctx, &req)
	if err != nil {
		http.Error(w, "Failed to send message", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func getAccessToken(r *http.Request) (string, error) {
	accessCookie, err := r.Cookie("access_token")
	if err != nil {
		return "", err
	}
	return accessCookie.Value, nil
}

func (s *Server) addTokenToContext(ctx context.Context, token string) context.Context {
	md := metadata.Pairs("authorization", "Bearer "+token)
	return metadata.NewOutgoingContext(ctx, md)
}

func setAuthCookies(w http.ResponseWriter, access, refresh string) {
	accessCookie := &http.Cookie{
		Name:     "access_token",
		Value:    access,
		MaxAge:   15 * 60, // 15 минут
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, accessCookie)

	refreshCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    refresh,
		MaxAge:   7 * 24 * 3600, // 7 дней
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, refreshCookie)
}

func (s *Server) markAsSpamHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		http.Error(w, "Access token required", http.StatusUnauthorized)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.MarkAsSpamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := s.messageClient.MarkAsSpam(ctx, &req)
	if err != nil {
		http.Error(w, "Failed to mark as spam", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) moveToFolderHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		http.Error(w, "Access token required", http.StatusUnauthorized)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.MoveToFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := s.messageClient.MoveToFolder(ctx, &req)
	if err != nil {
		http.Error(w, "Failed to move to folder", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) createFolderHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		http.Error(w, "Access token required", http.StatusUnauthorized)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.CreateFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := s.messageClient.CreateFolder(ctx, &req)
	if err != nil {
		http.Error(w, "Failed to create folder", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) getFolderHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		http.Error(w, "Access token required", http.StatusUnauthorized)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	folderName := r.PathValue("folder_name")
	lastMessageID := r.URL.Query().Get("last_message_id")
	lastDatetime := r.URL.Query().Get("last_datetime")
	limit := r.URL.Query().Get("limit")

	foldersReq := &messagesproto.GetFoldersRequest{}
	foldersResp, err := s.messageClient.GetFolders(ctx, foldersReq)
	if err != nil {
		http.Error(w, "Failed to get folders", http.StatusInternalServerError)
		return
	}

	var folderID string
	for _, folder := range foldersResp.Folders {
		if folder.FolderName == folderName {
			folderID = folder.FolderId
			break
		}
	}

	if folderID == "" {
		http.Error(w, "Folder not found", http.StatusNotFound)
		return
	}

	req := &messagesproto.GetFolderRequest{
		FolderId:      folderID,
		LastMessageId: lastMessageID,
		LastDatetime:  lastDatetime,
		Limit:         limit,
	}

	resp, err := s.messageClient.GetFolder(ctx, req)
	if err != nil {
		http.Error(w, "Failed to get folder", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) getFoldersHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		http.Error(w, "Access token required", http.StatusUnauthorized)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	req := &messagesproto.GetFoldersRequest{}

	resp, err := s.messageClient.GetFolders(ctx, req)
	if err != nil {
		http.Error(w, "Failed to get folders", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) renameFolderHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		http.Error(w, "Access token required", http.StatusUnauthorized)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.RenameFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := s.messageClient.RenameFolder(ctx, &req)
	if err != nil {
		http.Error(w, "Failed to rename folder", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) deleteFolderHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		http.Error(w, "Access token required", http.StatusUnauthorized)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	folderID := r.URL.Query().Get("folder_id")
	req := &messagesproto.DeleteFolderRequest{
		FolderId: folderID,
	}

	resp, err := s.messageClient.DeleteFolder(ctx, req)
	if err != nil {
		http.Error(w, "Failed to delete folder", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) deleteMessageFromFolderHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		http.Error(w, "Access token required", http.StatusUnauthorized)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.DeleteMessageFromFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := s.messageClient.DeleteMessageFromFolder(ctx, &req)
	if err != nil {
		http.Error(w, "Failed to delete message from folder", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) saveDraftHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		http.Error(w, "Access token required", http.StatusUnauthorized)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.SaveDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := s.messageClient.SaveDraft(ctx, &req)
	if err != nil {
		http.Error(w, "Failed to save draft", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) deleteDraftHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		http.Error(w, "Access token required", http.StatusUnauthorized)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.DeleteDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := s.messageClient.DeleteDraft(ctx, &req)
	if err != nil {
		http.Error(w, "Failed to delete draft", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) sendDraftHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		http.Error(w, "Access token required", http.StatusUnauthorized)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.SendDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := s.messageClient.SendDraft(ctx, &req)
	if err != nil {
		http.Error(w, "Failed to send draft", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
