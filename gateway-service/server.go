package gateway_service

import (
	"2025_2_a4code/internal/config"
	"2025_2_a4code/internal/http-server/middleware/cors"
	csrfcheck "2025_2_a4code/internal/http-server/middleware/csrf-check"
	"2025_2_a4code/internal/http-server/middleware/logger"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"2025_2_a4code/pkg/authproto"
	"2025_2_a4code/pkg/messagesproto"
	"2025_2_a4code/pkg/profileproto"
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

	conn1, err := grpc.NewClient(cfg.AuthPort, opts...)
	if err != nil {
		return nil, err
	}

	authClient := authproto.NewAuthServiceClient(conn1)

	conn2, err := grpc.NewClient(cfg.ProfilePort, opts...)
	if err != nil {
		return nil, err
	}

	profileClient := profileproto.NewProfileServiceClient(conn2)

	conn3, err := grpc.NewClient(cfg.MessagesPort, opts...)
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

	// Создаем роутер
	mainMux := http.NewServeMux()

	// Роутинг
	mainMux.Handle("POST /auth/login", http.HandlerFunc(s.loginHandler))
	mainMux.Handle("POST /auth/signup", http.HandlerFunc(s.signupHandler))
	mainMux.Handle("POST /auth/refresh", http.HandlerFunc(s.refreshHandler))
	mainMux.Handle("POST /auth/logout", http.HandlerFunc(s.logoutHandler))

	mainMux.Handle("GET /user/profile", http.HandlerFunc(s.getProfileHandler))
	mainMux.Handle("PUT /user/profile", http.HandlerFunc(s.updateProfileHandler))
	mainMux.Handle("GET /user/settings", http.HandlerFunc(s.settingsHandler))
	mainMux.Handle("POST /user/upload/avatar", http.HandlerFunc(s.uploadAvatarHandler))

	mainMux.Handle("GET /messages/inbox", http.HandlerFunc(s.inboxHandler))
	mainMux.Handle("GET /messages/{message_id}", http.HandlerFunc(s.messagePageHandler))
	mainMux.Handle("POST /messages/reply", http.HandlerFunc(s.replyHandler))
	mainMux.Handle("POST /messages/send", http.HandlerFunc(s.sendHandler))
	mainMux.Handle("GET /messages/sent", http.HandlerFunc(s.sentHandler))

	// подключение middleware
	var handler http.Handler = mainMux
	handler = logger.New(log)(handler)
	handler = cors.New()(handler)
	handler = csrfcheck.New()(handler)

	// Настройка сервера
	s.httpServer = &http.Server{
		Addr:         s.cfg.GatewayPort,
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

// Auth handlers

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
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) signupHandler(w http.ResponseWriter, r *http.Request) {
	var req authproto.SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := s.authClient.Signup(r.Context(), &req)
	if err != nil {
		http.Error(w, "Signup failed", http.StatusBadRequest)
		return
	}

	setAuthCookies(w, resp.AccessToken, resp.RefreshToken)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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
	json.NewEncoder(w).Encode(resp)
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
func (s *Server) inboxHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		http.Error(w, "Access token required", http.StatusUnauthorized)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	req := &messagesproto.InboxRequest{
		LastMessageId: r.URL.Query().Get("last_message_id"),
		LastDatetime:  r.URL.Query().Get("last_datetime"),
		Limit:         r.URL.Query().Get("limit"),
	}

	resp, err := s.messageClient.Inbox(ctx, req)
	if err != nil {
		http.Error(w, "Failed to get inbox", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

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

func (s *Server) sentHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		http.Error(w, "Access token required", http.StatusUnauthorized)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	req := &messagesproto.SentRequest{
		LastMessageId: r.URL.Query().Get("last_message_id"),
		LastDatetime:  r.URL.Query().Get("last_datetime"),
		Limit:         r.URL.Query().Get("limit"),
	}

	resp, err := s.messageClient.Sent(ctx, req)
	if err != nil {
		http.Error(w, "Failed to get sent messages", http.StatusInternalServerError)
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

func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
