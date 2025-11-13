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

	"2025_2_a4code/pkg/authproto"
)

type Server struct {
	cfg        *config.AppConfig
	httpServer *http.Server
	authClient authproto.AuthServiceClient
}

func NewServer(cfg *config.AppConfig) (*Server, error) {
	// Создаем gRPC соединение
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(cfg.AuthPort, opts...)
	if err != nil {
		return nil, err
	}

	// TODO: еще сервисы

	authClient := authproto.NewAuthServiceClient(conn)

	return &Server{
		cfg:        cfg,
		authClient: authClient,
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
