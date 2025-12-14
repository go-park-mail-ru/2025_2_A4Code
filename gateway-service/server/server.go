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
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Server struct {
	cfg               *config.Config
	httpServer        *http.Server
	authClient        authproto.AuthServiceClient
	profileClient     profileproto.ProfileServiceClient
	messageClient     messagesproto.MessagesServiceClient
	minioClient       *minio.Client
	minioBucket       string
	attachmentsBucket string
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

func NewServer(cfg *config.Config) (*Server, error) {
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	authConn, err := grpc.NewClient(cfg.AppConfig.Host+":"+cfg.AppConfig.AuthPort, opts...)
	if err != nil {
		return nil, err
	}
	profileConn, err := grpc.NewClient(cfg.AppConfig.Host+":"+cfg.AppConfig.ProfilePort, opts...)
	if err != nil {
		return nil, err
	}
	messagesConn, err := grpc.NewClient(cfg.AppConfig.Host+":"+cfg.AppConfig.MessagesPort, opts...)
	if err != nil {
		return nil, err
	}

	var minioClient *minio.Client
	bucketName := ""
	attachmentsBucket := ""
	if cfg.MinioConfig != nil {
		client, err := minio.New(cfg.MinioConfig.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.MinioConfig.User, cfg.MinioConfig.Password, ""),
			Secure: cfg.MinioConfig.UseSSL,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to init minio client: %w", err)
		}
		minioClient = client
		bucketName = cfg.MinioConfig.BucketName
		attachmentsBucket = cfg.MinioConfig.AttachmentsBucketName
		if strings.TrimSpace(attachmentsBucket) == "" {
			attachmentsBucket = "attachments"
		}

		if err := ensureBucket(context.Background(), minioClient, bucketName); err != nil {
			return nil, fmt.Errorf("ensure bucket %s: %w", bucketName, err)
		}
		if err := ensureBucket(context.Background(), minioClient, attachmentsBucket); err != nil {
			return nil, fmt.Errorf("ensure attachments bucket %s: %w", attachmentsBucket, err)
		}
	}

	return &Server{
		cfg:               cfg,
		authClient:        authproto.NewAuthServiceClient(authConn),
		profileClient:     profileproto.NewProfileServiceClient(profileConn),
		messageClient:     messagesproto.NewMessagesServiceClient(messagesConn),
		minioClient:       minioClient,
		minioBucket:       bucketName,
		attachmentsBucket: attachmentsBucket,
	}, nil
}

func (s *Server) Start(ctx context.Context) error {
	log := logger.GetLogger(ctx)
	slog.SetDefault(log)

	// Р—Р°РїСѓСЃРє СЃРµСЂРІРµСЂР° РјРµС‚СЂРёРє
	go func() {
		http.Handle("/metrics", promhttp.Handler()) // promhttp СЌРєСЃРїРѕСЂС‚РёСЂСѓРµС‚ CPU/Mem Р°РІС‚РѕРјР°С‚РёС‡РµСЃРєРё
		metricsAddr := ":" + s.cfg.AppConfig.GatewayMetricsPort
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
	mux.Handle("GET /files/{file_path...}", http.HandlerFunc(s.downloadFileHandler))
	mux.Handle("POST /files/upload", http.HandlerFunc(s.uploadFileHandler))
	mux.Handle("GET /ws/notifications", http.HandlerFunc(s.wsNotificationsHandler))

	var handler http.Handler = mux
	handler = logger.New(log)(handler)
	handler = cors.New()(handler)
	handler = metrics.Middleware(handler)

	s.httpServer = &http.Server{
		Addr:         ":" + s.cfg.AppConfig.GatewayPort,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Info("Server is listening port: " + s.cfg.AppConfig.GatewayPort)
	if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Error("Server stopped: " + err.Error())
		return err
	}
	return nil
}

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	var req authproto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Некорректное тело запроса")
		return
	}

	resp, err := s.authClient.Login(r.Context(), &req)
	if err != nil {
		writeGrpcAwareError(w, err, "Неверный логин или пароль")
		return
	}

	setAuthCookies(w, resp.AccessToken, resp.RefreshToken)
	respondSuccess(w, resp)
}

func (s *Server) signupHandler(w http.ResponseWriter, r *http.Request) {
	var req authproto.SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Некорректное тело запроса")
		return
	}

	resp, err := s.authClient.Signup(r.Context(), &req)
	if err != nil {
		if grpcErr, ok := status.FromError(err); ok {
			switch grpcErr.Code() {
			case codes.InvalidArgument, codes.AlreadyExists:
				writeResponse(w, http.StatusBadRequest, "Ошибка регистрации: "+grpcErr.Message(), nil)
			default:
				respondError(w, "Ошибка регистрации: "+grpcErr.Message())
			}
		} else {
			respondError(w, "Ошибка регистрации")
		}
		return
	}

	setAuthCookies(w, resp.AccessToken, resp.RefreshToken)
	respondSuccess(w, resp)
}

func (s *Server) refreshHandler(w http.ResponseWriter, r *http.Request) {
	refreshToken := ""

	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		refreshToken = strings.TrimSpace(authHeader[7:])
	}

	if refreshToken == "" {
		if cookie, err := r.Cookie("refresh_token"); err == nil {
			refreshToken = cookie.Value
		}
	}

	if refreshToken == "" {
		var body struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil && strings.TrimSpace(body.RefreshToken) != "" {
			refreshToken = body.RefreshToken
		}
	}

	if refreshToken == "" {
		writeResponse(w, http.StatusUnauthorized, "????????? refresh token", nil)
		return
	}

	req := &authproto.RefreshRequest{RefreshToken: refreshToken}
	resp, err := s.authClient.Refresh(r.Context(), req)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "?? ??????? ???????? ????? ???????", nil)
		return
	}

	setAuthCookies(w, resp.AccessToken, refreshToken)
	respondSuccess(w, resp)
}

func (s *Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	req := &authproto.LogoutRequest{}
	resp, err := s.authClient.Logout(r.Context(), req)
	if err != nil {
		respondError(w, "Ошибка выхода")
		return
	}

	clearCookie := func(name string) {
		http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	}
	clearCookie("access_token")
	clearCookie("refresh_token")

	respondSuccess(w, resp)
}

func (s *Server) getProfileHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Необходим access token", nil)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)
	resp, err := s.profileClient.GetProfile(ctx, &profileproto.GetProfileRequest{})
	if err != nil {
		writeGrpcAwareError(w, err, "Не удалось получить профиль")
		return
	}

	dto := mapProfile(resp.Profile)
	respondSuccess(w, dto)
}

func (s *Server) updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Необходим access token", nil)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req profileproto.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Некорректное тело запроса", nil)
		return
	}

	resp, err := s.profileClient.UpdateProfile(ctx, &req)
	if err != nil {
		writeGrpcAwareError(w, err, "Не удалось обновить профиль")
		return
	}

	respondSuccess(w, mapProfile(resp.Profile))
}

func (s *Server) settingsHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Необходим access token", nil)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)
	resp, err := s.profileClient.Settings(ctx, &profileproto.SettingsRequest{})
	if err != nil {
		writeGrpcAwareError(w, err, "Не удалось получить настройки")
		return
	}

	respondSuccess(w, resp)
}

// Messages handlers

func (s *Server) messagePageHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Необходим access token", nil)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)
	req := &messagesproto.MessagePageRequest{MessageId: r.PathValue("message_id")}

	resp, err := s.messageClient.MessagePage(ctx, req)
	if err != nil {
		//respondError(w, "Не удалось получить письмо")
		writeGrpcAwareError(w, err, "Не удалось получить письмо")
		return
	}

	respondSuccess(w, resp)
}

func (s *Server) replyHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Необходим access token", nil)
		return
	}
	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.ReplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Некорректное тело запроса", nil)
		return
	}

	if _, err := s.messageClient.Reply(ctx, &req); err != nil {
		respondError(w, "Не удалось отправить ответ")
		return
	}
	respondSuccess(w, map[string]string{"status": "ok"})
}

func (s *Server) sendHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Необходим access token", nil)
		return
	}
	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Некорректное тело запроса", nil)
		return
	}

	if _, err := s.messageClient.Send(ctx, &req); err != nil {
		respondError(w, "Не удалось отправить письмо")
		return
	}
	respondSuccess(w, map[string]string{"status": "ok"})
}

func (s *Server) getFolderHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Необходим access token", nil)
		return
	}
	ctx := s.addTokenToContext(r.Context(), accessToken)

	folderKey := r.PathValue("folder_name")
	lastMessageID := r.URL.Query().Get("last_message_id")
	lastDatetime := r.URL.Query().Get("last_datetime")
	limit := r.URL.Query().Get("limit")

	foldersResp, err := s.messageClient.GetFolders(ctx, &messagesproto.GetFoldersRequest{})
	if err != nil {
		writeGrpcAwareError(w, err, "Не удалось получить папки")
		return
	}

	folderID := resolveFolderID(foldersResp.Folders, folderKey)
	if folderID == "" {
		if strings.EqualFold(folderKey, "inbox") {
			s.handleInbox(ctx, w, r)
			return
		}
		writeResponse(w, http.StatusNotFound, "Папка не найдена", nil)
		return
	}

	s.respondFolder(ctx, w, r, folderID, lastMessageID, lastDatetime, limit, "Не удалось получить папку")
}

func (s *Server) getFoldersHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Необходим access token", nil)
		return
	}
	ctx := s.addTokenToContext(r.Context(), accessToken)

	resp, err := s.messageClient.GetFolders(ctx, &messagesproto.GetFoldersRequest{})
	if err != nil {
		writeGrpcAwareError(w, err, "Не удалось получить папки")
		return
	}
	respondSuccess(w, resp)
}

func (s *Server) inboxHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Необходим access token", nil)
		return
	}
	ctx := s.addTokenToContext(r.Context(), accessToken)
	s.handleInbox(ctx, w, r)
}

func (s *Server) handleInbox(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	foldersResp, err := s.messageClient.GetFolders(ctx, &messagesproto.GetFoldersRequest{})
	if err != nil {
		respondError(w, "Не удалось получить папки")
		return
	}

	folderID := resolveFolderID(foldersResp.Folders, "inbox")
	if folderID == "" {
		respondSuccess(w, emptyFolderResponse())
		return
	}

	s.respondFolder(ctx, w, r, folderID, r.URL.Query().Get("last_message_id"), r.URL.Query().Get("last_datetime"), r.URL.Query().Get("limit"), "Не удалось получить входящие")
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
		writeResponse(w, http.StatusUnauthorized, "Необходим access token", nil)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.RenameFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Некорректное тело запроса", nil)
		return
	}

	resp, err := s.messageClient.RenameFolder(ctx, &req)
	if err != nil {
		writeGrpcAwareError(w, err, "Не удалось переименовать папку")
		return
	}

	respondSuccess(w, resp)
}

func (s *Server) deleteFolderHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Необходим access token", nil)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	folderID := r.URL.Query().Get("folder_id")
	req := &messagesproto.DeleteFolderRequest{FolderId: folderID}

	resp, err := s.messageClient.DeleteFolder(ctx, req)
	if err != nil {
		respondError(w, "Не удалось удалить папку")
		return
	}

	respondSuccess(w, resp)
}

func (s *Server) deleteMessageFromFolderHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Необходим access token", nil)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.DeleteMessageFromFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Некорректное тело запроса", nil)
		return
	}

	resp, err := s.messageClient.DeleteMessageFromFolder(ctx, &req)
	if err != nil {
		respondError(w, "Не удалось удалить письмо из папки")
		return
	}

	respondSuccess(w, resp)
}

func (s *Server) saveDraftHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Необходим access token", nil)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.SaveDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Некорректное тело запроса", nil)
		return
	}

	resp, err := s.messageClient.SaveDraft(ctx, &req)
	if err != nil {
		respondError(w, "Не удалось сохранить черновик")
		return
	}

	respondSuccess(w, resp)
}

func (s *Server) deleteDraftHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Необходим access token", nil)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.DeleteDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Некорректное тело запроса", nil)
		return
	}

	resp, err := s.messageClient.DeleteDraft(ctx, &req)
	if err != nil {
		respondError(w, "Не удалось удалить черновик")
		return
	}

	respondSuccess(w, resp)
}

func (s *Server) sendDraftHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Необходим access token", nil)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.SendDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Некорректное тело запроса", nil)
		return
	}

	resp, err := s.messageClient.SendDraft(ctx, &req)
	if err != nil {
		respondError(w, "Не удалось отправить черновик")
		return
	}

	respondSuccess(w, resp)
}

func (s *Server) downloadFileHandler(w http.ResponseWriter, r *http.Request) {
	if s.minioClient == nil || s.attachmentsBucket == "" {
		writeResponse(w, http.StatusInternalServerError, "file storage is not configured", nil)
		return
	}

	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Необходим access token", nil)
		return
	}

	ctx := s.addTokenToContext(r.Context(), accessToken)
	rawPath := r.PathValue("file_path")
	if strings.TrimSpace(rawPath) == "" {
		writeResponse(w, http.StatusBadRequest, "file path is required", nil)
		return
	}

	objectName := path.Clean(strings.TrimPrefix(rawPath, "/"))
	if objectName == "." || objectName == "/" || objectName == "" {
		writeResponse(w, http.StatusBadRequest, "invalid file path", nil)
		return
	}

	obj, err := s.minioClient.GetObject(ctx, s.attachmentsBucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "failed to fetch file", nil)
		return
	}
	defer obj.Close()

	stat, err := obj.Stat()
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" || errResp.StatusCode == http.StatusNotFound {
			writeResponse(w, http.StatusNotFound, "file not found", nil)
			return
		}
		writeResponse(w, http.StatusInternalServerError, "failed to stat file", nil)
		return
	}

	filename := path.Base(objectName)
	if filename != "" {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, url.PathEscape(filename)))
	}
	if stat.ContentType != "" {
		w.Header().Set("Content-Type", stat.ContentType)
	}
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size))

	if _, err := io.Copy(w, obj); err != nil {
		writeResponse(w, http.StatusInternalServerError, "failed to stream file", nil)
	}
}

func (s *Server) uploadFileHandler(w http.ResponseWriter, r *http.Request) {
	if s.minioClient == nil || s.attachmentsBucket == "" {
		writeResponse(w, http.StatusInternalServerError, "file storage is not configured", nil)
		return
	}

	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "invalid access token", nil)
		return
	}
	ctx := s.addTokenToContext(r.Context(), accessToken)

	if err := r.ParseMultipartForm(12 << 20); err != nil {
		writeResponse(w, http.StatusBadRequest, "failed to parse form", nil)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "no file provided", nil)
		return
	}
	defer file.Close()

	const maxSize = 10 * 1024 * 1024
	if header.Size > maxSize {
		writeResponse(w, http.StatusBadRequest, "file too large", nil)
		return
	}

	objectName := strings.TrimSpace(r.FormValue("path"))
	if objectName == "" {
		objectName = path.Join("attachments", fmt.Sprintf("%d", time.Now().UnixNano()), sanitizeFileName(header.Filename))
	}
	objectName = strings.TrimPrefix(path.Clean(objectName), "/")
	if objectName == "." || objectName == "" {
		writeResponse(w, http.StatusBadRequest, "invalid storage path", nil)
		return
	}

	info, err := s.minioClient.PutObject(ctx, s.attachmentsBucket, objectName, file, header.Size, minio.PutObjectOptions{
		ContentType: header.Header.Get("Content-Type"),
	})
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "failed to store file", nil)
		return
	}

	respBody := map[string]interface{}{
		"storage_path": objectName,
		"name":         header.Filename,
		"size":         info.Size,
		"file_type":    header.Header.Get("Content-Type"),
	}
	respondSuccess(w, respBody)
}

func sanitizeFileName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "file.bin"
	}
	return base
}

func (s *Server) wsNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	log := logger.GetLogger(r.Context()).With(slog.String("component", "ws_notifications"))

	accessToken, err := getAccessToken(r)
	if err != nil {
		log.Warn("ws auth failed: no token")
		writeResponse(w, http.StatusUnauthorized, "authentication required", nil)
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("ws upgrade failed", slog.String("error", err.Error()))
		return
	}
	defer conn.Close()
	log.Info("ws connected")

	ctx := s.addTokenToContext(r.Context(), accessToken)
	var lastTotal, lastUnread int64

	type payload struct {
		Type   string `json:"type"`
		Total  int64  `json:"total"`
		Unread int64  `json:"unread"`
	}

	send := func(total, unread int64) error {
		return conn.WriteJSON(payload{Type: "mail_update", Total: total, Unread: unread})
	}

	fetch := func() (int64, int64, error) {
		foldersResp, err := s.messageClient.GetFolders(ctx, &messagesproto.GetFoldersRequest{})
		if err != nil {
			log.Warn("get folders failed", slog.String("error", err.Error()))
			return 0, 0, err
		}
		folderID := resolveFolderID(foldersResp.Folders, "inbox")
		if folderID == "" {
			return 0, 0, fmt.Errorf("inbox folder not found")
		}
		resp, err := s.messageClient.GetFolder(ctx, &messagesproto.GetFolderRequest{FolderId: folderID, Limit: "1"})
		if err != nil {
			log.Warn("inbox fetch failed", slog.String("error", err.Error()))
			return 0, 0, err
		}
		total, _ := strconv.ParseInt(resp.MessageTotal, 10, 64)
		unread, _ := strconv.ParseInt(resp.MessageUnread, 10, 64)
		return total, unread, nil
	}

	if total, unread, err := fetch(); err == nil {
		lastTotal, lastUnread = total, unread
		_ = send(total, unread)
	} else {
		log.Warn("initial fetch failed", slog.String("error", err.Error()))
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			total, unread, err := fetch()
			if err != nil {
				log.Warn("periodic fetch failed", slog.String("error", err.Error()))
				_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "fetch failed"))
				return
			}
			if total != lastTotal || unread != lastUnread {
				lastTotal, lastUnread = total, unread
				if err := send(total, unread); err != nil {
					log.Warn("send failed", slog.String("error", err.Error()))
					return
				}
			}
		}
	}
}

func setAuthCookies(w http.ResponseWriter, access, refresh string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    access,
		MaxAge:   15 * 60,
		HttpOnly: true,
		Secure:   false,
		Path:     "/",
		SameSite: http.SameSiteNoneMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refresh,
		MaxAge:   30 * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   false,
		Path:     "/",
		SameSite: http.SameSiteNoneMode,
	})

	if strings.TrimSpace(access) != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "ws_token",
			Value:    access,
			MaxAge:   15 * 60,
			HttpOnly: false,
			Secure:   false,
			Path:     "/",
			SameSite: http.SameSiteNoneMode,
		})
	}
}

func ensureBucket(ctx context.Context, client *minio.Client, bucket string) error {
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
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
		writeResponse(w, http.StatusUnauthorized, "Необходим access token", nil)
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
		writeGrpcAwareError(w, err, "Не удалось загрузить аватар")
		return
	}

	respondSuccess(w, resp)
}

func (s *Server) getAvatarHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Необходим access token", nil)
		return
	}
	ctx := s.addTokenToContext(r.Context(), accessToken)

	targetURL := strings.TrimSpace(r.URL.Query().Get("url"))
	if targetURL == "" {
		profileResp, err := s.profileClient.GetProfile(ctx, &profileproto.GetProfileRequest{})
		if err != nil {
			writeGrpcAwareError(w, err, "Не удалось получить профиль")
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
		writeResponse(w, http.StatusUnauthorized, "Необходим access token", nil)
		return
	}
	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.MarkAsSpamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Некорректное тело запроса", nil)
		return
	}

	resp, err := s.messageClient.MarkAsSpam(ctx, &req)
	if err != nil {
		respondError(w, "Не удалось пометить как спам")
		return
	}

	respondSuccess(w, resp)
}

func (s *Server) moveToFolderHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Необходим access token", nil)
		return
	}
	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.MoveToFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Некорректное тело запроса", nil)
		return
	}

	resp, err := s.messageClient.MoveToFolder(ctx, &req)
	if err != nil {
		respondError(w, "Не удалось переместить письмо")
		return
	}

	respondSuccess(w, resp)
}

func (s *Server) createFolderHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken(r)
	if err != nil {
		writeResponse(w, http.StatusUnauthorized, "Необходим access token", nil)
		return
	}
	ctx := s.addTokenToContext(r.Context(), accessToken)

	var req messagesproto.CreateFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, http.StatusBadRequest, "Некорректное тело запроса", nil)
		return
	}

	resp, err := s.messageClient.CreateFolder(ctx, &req)
	if err != nil {
		writeGrpcAwareError(w, err, "Не удалось создать папку")
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
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:]), nil
	}
	queryToken := strings.TrimSpace(r.URL.Query().Get("token"))
	if queryToken == "" {
		queryToken = strings.TrimSpace(r.URL.Query().Get("access_token"))
	}
	if queryToken != "" {
		return queryToken, nil
	}
	if cookie, err := r.Cookie("access_token"); err == nil {
		return cookie.Value, nil
	}
	if cookie, err := r.Cookie("ws_token"); err == nil {
		return cookie.Value, nil
	}
	return "", fmt.Errorf("access token not found")
}

func (s *Server) addTokenToContext(ctx context.Context, token string) context.Context {
	md := metadata.Pairs("authorization", "Bearer "+token)
	return metadata.NewOutgoingContext(ctx, md)
}
