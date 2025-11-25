package messages_service

import (
	"2025_2_a4code/internal/domain"
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/mail"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"2025_2_a4code/internal/http-server/middleware/logger"
	"2025_2_a4code/internal/lib/session"
	"2025_2_a4code/internal/lib/validation"

	pb "2025_2_a4code/messages-service/pkg/messagesproto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedMessagesServiceServer
	messageUCase MessageUsecase
	avatarUCase  AvatarUsecase
	JWTSecret    []byte
}

type MessageUsecase interface {
	// базовые методы для сообщений
	FindByMessageID(ctx context.Context, messageID int64) (*domain.Message, error)
	FindFullByMessageID(ctx context.Context, messageID int64, profileID int64) (domain.FullMessage, error)
	SaveMessage(ctx context.Context, receiverProfileEmail string, senderBaseProfileID int64, topic, text string) (int64, error)
	SaveFile(ctx context.Context, messageID int64, fileName, fileType, storagePath string, size int64) (fileID int64, err error)

	// методы для тредов
	SaveThread(ctx context.Context, messageID int64) (threadID int64, err error)
	SaveThreadIdToMessage(ctx context.Context, messageID int64, threadID int64) error
	FindThreadsByProfileID(ctx context.Context, profileID int64) ([]domain.ThreadInfo, error)

	// методы для работы с сообщениями
	MarkMessageAsRead(ctx context.Context, messageID int64, profileID int64) error
	MarkMessageAsSpam(ctx context.Context, messageID int64, profileID int64) error
	IsUsersMessage(ctx context.Context, messageID int64, profileID int64) (bool, error)

	// методы для черновиков
	SaveDraft(ctx context.Context, profileID int64, draftID, receiverEmail, topic, text string) (int64, error)
	IsDraftBelongsToUser(ctx context.Context, draftID, profileID int64) (bool, error)
	DeleteDraft(ctx context.Context, draftID, profileID int64) error
	SendDraft(ctx context.Context, draftID, profileID int64) error
	GetDraft(ctx context.Context, draftID, profileID int64) (domain.FullMessage, error)

	// методы для папок
	MoveToFolder(ctx context.Context, profileID, messageID, folderID int64) error
	GetFolderByType(ctx context.Context, profileID int64, folderType string) (int64, error)
	ShouldMarkAsRead(ctx context.Context, messageID, profileID int64) (bool, error)
	CreateFolder(ctx context.Context, profileID int64, folderName string) (*domain.Folder, error)
	GetUserFolders(ctx context.Context, profileID int64) ([]domain.Folder, error)
	RenameFolder(ctx context.Context, profileID, folderID int64, newName string) (*domain.Folder, error)
	DeleteFolder(ctx context.Context, profileID, folderID int64) error
	DeleteMessageFromFolder(ctx context.Context, profileID, messageID, folderID int64) error
	GetFolderMessagesWithKeysetPagination(ctx context.Context, profileID, folderID, lastMessageID int64, lastDatetime time.Time, limit int) ([]domain.Message, error)
	GetFolderMessagesInfo(ctx context.Context, profileID, folderID int64) (domain.Messages, error)

	// методы для отправки сообщений с автоматическим распределением по папкам
	SendMessage(ctx context.Context, receiverEmail string, senderProfileID int64, topic, text string) (int64, error)
	ReplyToMessage(ctx context.Context, receiverEmail string, senderProfileID int64, threadRoot int64, topic, text string) (int64, error)
}

type AvatarUsecase interface {
	GetAvatarPresignedURL(ctx context.Context, objectName string, duration time.Duration) (*url.URL, error)
}

const (
	maxTopicLen       = 255
	maxTextLen        = 10000
	maxFileSize       = 10 * 1024 * 1024 // 10 MB
	defaultLimitFiles = 20
)

var allowedFileTypes = map[string]struct{}{
	"image/jpeg":      {},
	"image/png":       {},
	"application/pdf": {},
	"text/plain":      {},
}

func New(messageUCase MessageUsecase, avatarUCase AvatarUsecase, secret []byte) *Server {
	return &Server{
		messageUCase: messageUCase,
		avatarUCase:  avatarUCase,
		JWTSecret:    secret,
	}
}

func (s *Server) getProfileID(ctx context.Context) (int64, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, status.Error(codes.Unauthenticated, "metadata is not provided")
	}

	tokens := md.Get("authorization")
	if len(tokens) == 0 {
		return 0, status.Error(codes.Unauthenticated, "authorization token is not provided")
	}

	tokenString := strings.TrimPrefix(tokens[0], "Bearer ")
	return session.GetProfileIDFromTokenString(tokenString, s.JWTSecret, "access")
}

func (s *Server) MessagePage(ctx context.Context, req *pb.MessagePageRequest) (*pb.MessagePageResponse, error) {
	const op = "messagesservice.MessagePage"
	log := logger.GetLogger(ctx)
	log.Debug("handle messages/{message_id}")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	messageID, err := strconv.ParseInt(req.MessageId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message id")
	}

	ok, err := s.messageUCase.IsUsersMessage(ctx, messageID, profileID)
	if err != nil {
		log.Error(op + ": failed to check if it is users message: " + err.Error())
		return nil, status.Error(codes.Internal, "could not get message")
	}
	if !ok {
		log.Debug(op + ": unpermitted access to message")
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	fullMessage, err := s.messageUCase.FindFullByMessageID(ctx, messageID, profileID)
	if err != nil {
		log.Error(op + ": failed to get message: " + err.Error())
		return nil, status.Error(codes.Internal, "could not get message")
	}

	shouldMarkAsRead, err := s.messageUCase.ShouldMarkAsRead(ctx, messageID, profileID)
	if err != nil {
		log.Warn("failed to check if should mark as read: " + err.Error())
		shouldMarkAsRead = false
	}

	if shouldMarkAsRead {
		if err := s.messageUCase.MarkMessageAsRead(ctx, messageID, profileID); err != nil {
			log.Warn("failed to mark message as read: " + err.Error())
		}
	}

	if err := s.enrichSenderAvatar(ctx, &fullMessage.Sender); err != nil {
		log.Warn("failed to enrich sender avatar: " + err.Error())
	}

	pbFiles := make([]*pb.File, len(fullMessage.Files))
	for i, file := range fullMessage.Files {
		pbFiles[i] = &pb.File{
			Name:        file.Name,
			FileType:    file.FileType,
			Size:        strconv.FormatInt(file.Size, 10),
			StoragePath: file.StoragePath,
		}
	}

	return &pb.MessagePageResponse{
		Message: &pb.FullMessage{
			Topic:    fullMessage.Topic,
			Text:     fullMessage.Text,
			Datetime: fullMessage.Datetime.Format(time.RFC3339),
			ThreadId: fullMessage.ThreadRoot,
			Sender:   s.domainSenderToProto(&fullMessage.Sender),
			Files:    pbFiles,
		},
	}, nil
}

func (s *Server) Reply(ctx context.Context, req *pb.ReplyRequest) (*pb.ReplyResponse, error) {
	const op = "messagesservice.Reply"
	log := logger.GetLogger(ctx)
	log.Debug("handle messages/reply")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := s.validateReplyRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	threadRoot, err := s.resolveThreadRoot(ctx, req, profileID, log)
	if err != nil {
		return nil, err
	}

	safeTopic, safeText := sanitizeContent(req.Topic, req.Text)

	var messageID int64
	for _, receiver := range req.Receivers {
		msgID, err := s.messageUCase.ReplyToMessage(ctx, receiver.Email, profileID, threadRoot, safeTopic, safeText)
		if err != nil {
			log.Error(op + ": failed to reply to message: " + err.Error())
			return nil, status.Error(codes.Internal, "could not reply to message")
		}

		for _, file := range req.Files {
			size, _ := strconv.ParseInt(file.Size, 10, 64)
			_, err = s.messageUCase.SaveFile(ctx, msgID, file.Name, file.FileType, file.StoragePath, size)
			if err != nil {
				log.Error(op + ": failed to save file: " + err.Error())
				return nil, status.Error(codes.Internal, "could not save file")
			}
		}

		messageID = msgID
	}

	return &pb.ReplyResponse{
		MessageId: strconv.FormatInt(messageID, 10),
	}, nil
}

func (s *Server) Send(ctx context.Context, req *pb.SendRequest) (*pb.SendResponse, error) {
	const op = "messagesservice.Send"
	log := logger.GetLogger(ctx)
	log.Debug("handle messages/send")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := s.validateSendRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	safeTopic, safeText := sanitizeContent(req.Topic, req.Text)

	var messageID int64
	for _, receiver := range req.Receivers {
		msgID, err := s.messageUCase.SendMessage(ctx, receiver.Email, profileID, safeTopic, safeText)
		if err != nil {
			log.Error(op + ": failed to send message: " + err.Error())
			return nil, status.Error(codes.Internal, "could not send message")
		}

		if messageID == 0 {
			threadID, err := s.messageUCase.SaveThread(ctx, msgID)
			if err != nil {
				log.Error(op + ": failed to save thread: " + err.Error())
				return nil, status.Error(codes.Internal, "could not save thread")
			}

			if err := s.messageUCase.SaveThreadIdToMessage(ctx, msgID, threadID); err != nil {
				log.Error(op + ": failed to save thread id: " + err.Error())
				return nil, status.Error(codes.Internal, "could not save thread id")
			}
		}

		for _, file := range req.Files {
			size, _ := strconv.ParseInt(file.Size, 10, 64)
			_, err = s.messageUCase.SaveFile(ctx, msgID, file.Name, file.FileType, file.StoragePath, size)
			if err != nil {
				log.Error(op + ": failed to save file: " + err.Error())
				return nil, status.Error(codes.Internal, "could not save file")
			}
		}

		messageID = msgID
	}

	return &pb.SendResponse{
		MessageId: strconv.FormatInt(messageID, 10),
	}, nil
}

func (s *Server) domainSenderToProto(sender *domain.Sender) *pb.Sender {
	if sender == nil {
		return nil
	}
	return &pb.Sender{
		Email:    sender.Email,
		Username: sender.Username,
		Avatar:   sender.Avatar,
	}
}

func (s *Server) enrichSenderAvatar(ctx context.Context, sender *domain.Sender) error {
	if sender == nil || sender.Avatar == "" {
		return nil
	}

	objectName := sender.Avatar
	if strings.HasPrefix(objectName, "http://") || strings.HasPrefix(objectName, "https://") {
		parsed, err := url.Parse(objectName)
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

	url, err := s.avatarUCase.GetAvatarPresignedURL(ctx, objectName, 15*time.Minute)
	if err != nil {
		return err
	}

	sender.Avatar = url.String()
	return nil
}

func sanitizeContent(topic, text string) (string, string) {
	return html.EscapeString(topic), html.EscapeString(text)
}

func (s *Server) resolveThreadRoot(ctx context.Context, req *pb.ReplyRequest, profileID int64, log *slog.Logger) (int64, error) {
	rootMessageRaw := strings.TrimSpace(req.RootMessageId)
	if rootMessageRaw == "" {
		return 0, status.Error(codes.InvalidArgument, "thread root is required")
	}

	rootMessageID, err := strconv.ParseInt(rootMessageRaw, 10, 64)
	if err != nil {
		return 0, status.Error(codes.InvalidArgument, "invalid root message id")
	}

	threadRootRaw := strings.TrimSpace(req.ThreadRoot)
	if threadRootRaw != "" {
		threadRoot, err := strconv.ParseInt(threadRootRaw, 10, 64)
		if err != nil {
			log.Warn("invalid thread_root, recreating thread", "thread_root", threadRootRaw, "err", err)
		} else {
			// try to bind provided thread to root message to ensure it exists
			if err := s.messageUCase.SaveThreadIdToMessage(ctx, rootMessageID, threadRoot); err != nil {
				log.Warn("failed to attach provided thread to message, will recreate", "thread_root", threadRoot, "err", err)
			} else {
				return threadRoot, nil
			}
		}
	}

	// try to reuse existing thread if message already has it
	if full, err := s.messageUCase.FindFullByMessageID(ctx, rootMessageID, profileID); err == nil {
		if parsed, parseErr := strconv.ParseInt(strings.TrimSpace(full.ThreadRoot), 10, 64); parseErr == nil && parsed > 0 {
			return parsed, nil
		}
	} else {
		log.Warn("failed to fetch message to determine thread, will create new", "err", err)
	}

	threadRoot, err := s.messageUCase.SaveThread(ctx, rootMessageID)
	if err != nil {
		log.Error("failed to create thread for reply: " + err.Error())
		return 0, status.Error(codes.Internal, "could not create thread")
	}

	if err := s.messageUCase.SaveThreadIdToMessage(ctx, rootMessageID, threadRoot); err != nil {
		log.Error("failed to attach thread to message: " + err.Error())
		return 0, status.Error(codes.Internal, "could not attach thread to message")
	}

	return threadRoot, nil
}

func (s *Server) validateReplyRequest(req *pb.ReplyRequest) error {
	if req.Text == "" || req.Receivers == nil || len(req.Receivers) == 0 {
		return fmt.Errorf("empty request body")
	}

	if len(req.Topic) > maxTopicLen {
		return fmt.Errorf("topic too long")
	}
	if len(req.Text) > maxTextLen {
		return fmt.Errorf("text too long")
	}

	if validation.HasDangerousCharacters(req.Topic) {
		return fmt.Errorf("topic contains forbidden characters")
	}

	seen := make(map[string]struct{})
	for _, r := range req.Receivers {
		email := strings.TrimSpace(r.Email)
		if email == "" {
			return fmt.Errorf("empty receiver email")
		}
		if _, err := mail.ParseAddress(email); err != nil {
			return fmt.Errorf("invalid receiver email: %s", email)
		}
		lower := strings.ToLower(email)
		if _, ok := seen[lower]; ok {
			return fmt.Errorf("duplicate receiver: %s", email)
		}
		seen[lower] = struct{}{}

		if validation.HasDangerousCharacters(email) {
			return fmt.Errorf("receiver email contains forbidden characters: %s", email)
		}
	}

	if len(req.Files) > defaultLimitFiles {
		return fmt.Errorf("too many files")
	}
	for _, f := range req.Files {
		size, _ := strconv.ParseInt(f.Size, 10, 64)
		if size < 0 || size > maxFileSize {
			return fmt.Errorf("file size invalid or too large: %s", f.Name)
		}
		if _, ok := allowedFileTypes[f.FileType]; !ok {
			return fmt.Errorf("unsupported file type: %s", f.FileType)
		}
		base := filepath.Base(f.Name)
		if base != f.Name || strings.Contains(f.Name, "..") {
			return fmt.Errorf("invalid file name: %s", f.Name)
		}
		if validation.HasDangerousCharacters(f.StoragePath) {
			return fmt.Errorf("invalid storage path for file: %s", f.Name)
		}
		if validation.HasDangerousCharacters(f.Name) {
			return fmt.Errorf("invalid file name: %s", f.Name)
		}
	}

	return nil
}

func (s *Server) validateSendRequest(req *pb.SendRequest) error {
	if req.Text == "" || req.Receivers == nil || len(req.Receivers) == 0 {
		return fmt.Errorf("empty request body")
	}

	if len(req.Topic) > maxTopicLen {
		return fmt.Errorf("topic too long")
	}
	if len(req.Text) > maxTextLen {
		return fmt.Errorf("text too long")
	}

	if validation.HasDangerousCharacters(req.Topic) {
		return fmt.Errorf("topic contains forbidden characters")
	}

	seen := make(map[string]struct{})
	for _, r := range req.Receivers {
		email := strings.TrimSpace(r.Email)
		if email == "" {
			return fmt.Errorf("empty receiver email")
		}
		if _, err := mail.ParseAddress(email); err != nil {
			return fmt.Errorf("invalid receiver email: %s", email)
		}
		lower := strings.ToLower(email)
		if _, ok := seen[lower]; ok {
			return fmt.Errorf("duplicate receiver: %s", email)
		}
		seen[lower] = struct{}{}

		if validation.HasDangerousCharacters(email) {
			return fmt.Errorf("receiver email contains forbidden characters: %s", email)
		}
	}

	if len(req.Files) > defaultLimitFiles {
		return fmt.Errorf("too many files")
	}
	for _, f := range req.Files {
		size, _ := strconv.ParseInt(f.Size, 10, 64)
		if size < 0 || size > maxFileSize {
			return fmt.Errorf("file size invalid or too large: %s", f.Name)
		}
		if _, ok := allowedFileTypes[f.FileType]; !ok {
			return fmt.Errorf("unsupported file type: %s", f.FileType)
		}
		base := filepath.Base(f.Name)
		if base != f.Name || strings.Contains(f.Name, "..") {
			return fmt.Errorf("invalid file name: %s", f.Name)
		}
		if validation.HasDangerousCharacters(f.StoragePath) {
			return fmt.Errorf("invalid storage path for file: %s", f.Name)
		}
		if validation.HasDangerousCharacters(f.Name) {
			return fmt.Errorf("invalid file name: %s", f.Name)
		}
	}

	return nil
}

func (s *Server) MarkAsSpam(ctx context.Context, req *pb.MarkAsSpamRequest) (*pb.MarkAsSpamResponse, error) {
	const op = "messagesservice.MarkAsSpam"
	log := logger.GetLogger(ctx)
	log.Debug("handle messages/mark-as-spam")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	messageID, err := strconv.ParseInt(req.MessageId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message id")
	}

	if err := s.messageUCase.MarkMessageAsSpam(ctx, messageID, profileID); err != nil {
		log.Warn("failed to mark message as spam: " + err.Error())
		return nil, status.Error(codes.Internal, "could not mark message as spam")
	}

	return &pb.MarkAsSpamResponse{}, nil
}

func (s *Server) MoveToFolder(ctx context.Context, req *pb.MoveToFolderRequest) (*pb.MoveToFolderResponse, error) {
	const op = "messagesservice.MoveToFolder"
	log := logger.GetLogger(ctx)
	log.Debug("handle messages/move-to-folder")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	messageID, err := strconv.ParseInt(req.MessageId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message id")
	}

	folderID, err := strconv.ParseInt(req.FolderId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid folder id")
	}

	if err := s.messageUCase.MoveToFolder(ctx, profileID, messageID, folderID); err != nil {
		log.Error(op + ": failed to move message to folder: " + err.Error())
		return nil, status.Error(codes.Internal, "could not move message to folder")
	}

	return &pb.MoveToFolderResponse{}, nil
}

func (s *Server) CreateFolder(ctx context.Context, req *pb.CreateFolderRequest) (*pb.CreateFolderResponse, error) {
	const op = "messagesservice.CreateFolder"
	log := logger.GetLogger(ctx)
	log.Debug("handle messages/create-folder")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := s.validateFolderName(req.FolderName); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	folder, err := s.messageUCase.CreateFolder(ctx, profileID, req.FolderName)
	if err != nil {
		if errors.Is(err, domain.ErrFolderExists) {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		log.Error(op + ": failed to create folder: " + err.Error())
		return nil, status.Error(codes.Internal, "could not create folder")
	}

	return &pb.CreateFolderResponse{
		FolderId:   strconv.FormatInt(folder.ID, 10),
		FolderName: folder.Name,
		FolderType: string(folder.Type),
	}, nil
}

func (s *Server) validateFolderName(folderName string) error {
	if folderName == "" {
		return fmt.Errorf("folder name cannot be empty")
	}

	if len(folderName) > 50 {
		return fmt.Errorf("folder name too long (max 50 characters)")
	}

	if len(folderName) < 1 {
		return fmt.Errorf("folder name too short (min 1 character)")
	}

	if validation.HasDangerousCharacters(folderName) {
		return fmt.Errorf("folder name contains forbidden characters")
	}

	systemFolders := map[string]struct{}{
		"inbox":  {},
		"sent":   {},
		"draft":  {},
		"spam":   {},
		"trash":  {},
		"custom": {},
	}

	lowerName := strings.ToLower(folderName)
	if _, exists := systemFolders[lowerName]; exists {
		return fmt.Errorf("folder name '%s' is reserved for system folders", folderName)
	}

	return nil
}

func (s *Server) GetFolder(ctx context.Context, req *pb.GetFolderRequest) (*pb.GetFolderResponse, error) {
	const op = "messagesservice.GetFolder"
	log := logger.GetLogger(ctx)
	log.Debug("handle messages/get-folder")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	folderID, err := strconv.ParseInt(req.FolderId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid folder id")
	}

	var lastMessageID int64
	var lastDatetime time.Time
	limit := 20

	if req.LastMessageId != "" {
		if id, err := strconv.ParseInt(req.LastMessageId, 10, 64); err == nil {
			lastMessageID = id
		}
	}

	if req.LastDatetime != "" {
		if dt, err := time.Parse(time.RFC3339, req.LastDatetime); err == nil {
			lastDatetime = dt
		}
	}

	if req.Limit != "" {
		if l, err := strconv.Atoi(req.Limit); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	messages, err := s.messageUCase.GetFolderMessagesWithKeysetPagination(ctx, profileID, folderID, lastMessageID, lastDatetime, limit)
	if err != nil {
		log.Error(op + ": failed to get folder messages: " + err.Error())
		return nil, status.Error(codes.Internal, "could not get folder messages")
	}

	messagesInfo, err := s.messageUCase.GetFolderMessagesInfo(ctx, profileID, folderID)
	if err != nil {
		log.Error(op + ": failed to get folder messages info: " + err.Error())
		return nil, status.Error(codes.Internal, "could not get folder messages info")
	}

	pbMessages := make([]*pb.Message, 0, len(messages))
	var nextLastMessageID int64
	var nextLastDatetime time.Time

	for _, m := range messages {
		messageID, _ := strconv.ParseInt(m.ID, 10, 64)
		if err := s.enrichSenderAvatar(ctx, &m.Sender); err != nil {
			log.Warn("failed to enrich sender avatar: " + err.Error())
		}

		pbMessages = append(pbMessages, &pb.Message{
			Id:       m.ID,
			Sender:   s.domainSenderToProto(&m.Sender),
			Topic:    m.Topic,
			Snippet:  m.Snippet,
			Datetime: m.Datetime.Format(time.RFC3339),
			IsRead:   strconv.FormatBool(m.IsRead),
		})

		nextLastMessageID = messageID
		nextLastDatetime = m.Datetime
	}

	return &pb.GetFolderResponse{
		MessageTotal:  strconv.Itoa(messagesInfo.MessageTotal),
		MessageUnread: strconv.Itoa(messagesInfo.MessageUnread),
		Messages:      pbMessages,
		Pagination: &pb.PaginationInfo{
			HasNext:           strconv.FormatBool(len(messages) == limit),
			NextLastMessageId: strconv.FormatInt(nextLastMessageID, 10),
			NextLastDatetime:  nextLastDatetime.Format(time.RFC3339),
		},
	}, nil
}

func (s *Server) GetFolders(ctx context.Context, req *pb.GetFoldersRequest) (*pb.GetFoldersResponse, error) {
	const op = "messagesservice.GetFolders"
	log := logger.GetLogger(ctx)
	log.Debug("handle messages/get-folders")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	folders, err := s.messageUCase.GetUserFolders(ctx, profileID)
	if err != nil {
		log.Error(op + ": failed to get user folders: " + err.Error())
		return nil, status.Error(codes.Internal, "could not get folders")
	}

	// конвертируем domain папки в proto папки
	pbFolders := make([]*pb.Folder, 0, len(folders))
	for _, folder := range folders {
		pbFolders = append(pbFolders, &pb.Folder{
			FolderId:   strconv.FormatInt(folder.ID, 10),
			FolderName: folder.Name,
			FolderType: string(folder.Type),
		})
	}

	return &pb.GetFoldersResponse{
		Folders: pbFolders,
	}, nil
}

func (s *Server) RenameFolder(ctx context.Context, req *pb.RenameFolderRequest) (*pb.RenameFolderResponse, error) {
	const op = "messagesservice.RenameFolder"
	log := logger.GetLogger(ctx)
	log.Debug("handle messages/rename-folder")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	folderID, err := strconv.ParseInt(req.FolderId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid folder id")
	}

	if err := s.validateFolderName(req.NewFolderName); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	folder, err := s.messageUCase.RenameFolder(ctx, profileID, folderID, req.NewFolderName)
	if err != nil {
		if errors.Is(err, domain.ErrFolderExists) {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		log.Error(op + ": failed to rename folder: " + err.Error())
		return nil, status.Error(codes.Internal, "could not rename folder")
	}

	return &pb.RenameFolderResponse{
		FolderId:   strconv.FormatInt(folder.ID, 10),
		FolderName: folder.Name,
		FolderType: string(folder.Type),
	}, nil
}

func (s *Server) DeleteFolder(ctx context.Context, req *pb.DeleteFolderRequest) (*pb.DeleteFolderResponse, error) {
	const op = "messagesservice.DeleteFolder"

	log := logger.GetLogger(ctx)
	log.Debug("handle messages/delete-folder")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	folderID, err := strconv.ParseInt(req.FolderId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid folder id")
	}

	if err := s.messageUCase.DeleteFolder(ctx, profileID, folderID); err != nil {
		if errors.Is(err, domain.ErrFolderNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		if errors.Is(err, domain.ErrFolderSystem) {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		log.Error(op + ": failed to delete folder: " + err.Error())
		return nil, status.Error(codes.Internal, "could not delete folder")
	}

	return &pb.DeleteFolderResponse{}, nil
}

func (s *Server) DeleteMessageFromFolder(ctx context.Context, req *pb.DeleteMessageFromFolderRequest) (*pb.DeleteMessageFromFolderResponse, error) {
	const op = "messagesservice.DeleteMessageFromFolder"

	log := logger.GetLogger(ctx)
	log.Debug("handle messages/delete-message-from-folder")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	messageID, err := strconv.ParseInt(req.MessageId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message id")
	}

	folderID, err := strconv.ParseInt(req.FolderId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid folder id")
	}

	if err := s.messageUCase.DeleteMessageFromFolder(ctx, profileID, messageID, folderID); err != nil {
		log.Error(op + ": failed to delete message from folder: " + err.Error())
		return nil, status.Error(codes.Internal, "could not delete message from folder")
	}

	return &pb.DeleteMessageFromFolderResponse{}, nil
}

func (s *Server) SaveDraft(ctx context.Context, req *pb.SaveDraftRequest) (*pb.SaveDraftResponse, error) {
	const op = "messagesservice.SaveDraft"
	log := logger.GetLogger(ctx)
	log.Debug("handle messages/save-draft")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := s.validateDraftRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	receivers := req.Receivers
	if len(receivers) == 0 {
		receivers = []*pb.Receiver{{Email: ""}}
	}

	safeTopic, safeText := sanitizeContent(req.Topic, req.Text)

	var draftID int64

	for _, receiver := range receivers {
		msgID, err := s.messageUCase.SaveDraft(ctx, profileID, req.DraftId, receiver.Email, safeTopic, safeText)
		if err != nil {
			log.Error(op + ": failed to save draft: " + err.Error())
			return nil, status.Error(codes.Internal, "could not save draft")
		}
		draftID = msgID

		if req.DraftId == "" {
			var threadID int64
			if req.ThreadId != "" {
				threadID, err = strconv.ParseInt(req.ThreadId, 10, 64)
				if err != nil {
					log.Error(op + ": invalid thread id: " + err.Error())
					return nil, status.Error(codes.InvalidArgument, "invalid thread id")
				}
			} else {
				threadID, err = s.messageUCase.SaveThread(ctx, draftID)
				if err != nil {
					log.Error(op + ": failed to save thread: " + err.Error())
					return nil, status.Error(codes.Internal, "could not save thread")
				}
			}
			if err := s.messageUCase.SaveThreadIdToMessage(ctx, draftID, threadID); err != nil {
				log.Error(op + ": failed to save thread id: " + err.Error())
				return nil, status.Error(codes.Internal, "could not save thread id")
			}

			draftFolderID, err := s.messageUCase.GetFolderByType(ctx, profileID, "draft")
			if err != nil {
				log.Error(op + ": failed to get draft folder: " + err.Error())
				return nil, status.Error(codes.Internal, "could not get draft folder")
			}

			if err := s.messageUCase.MoveToFolder(ctx, profileID, draftID, draftFolderID); err != nil {
				log.Error(op + ": failed to put draft to folder: " + err.Error())
				return nil, status.Error(codes.Internal, "could not save draft to folder")
			}
		}

		for _, file := range req.Files {
			size, _ := strconv.ParseInt(file.Size, 10, 64)
			_, err = s.messageUCase.SaveFile(ctx, draftID, file.Name, file.FileType, file.StoragePath, size)
			if err != nil {
				log.Error(op + ": failed to save file: " + err.Error())
				return nil, status.Error(codes.Internal, "could not save file")
			}
		}
	}

	return &pb.SaveDraftResponse{
		DraftId: strconv.FormatInt(draftID, 10),
	}, nil
}

func (s *Server) validateDraftRequest(req *pb.SaveDraftRequest) error {
	if len(req.Topic) > maxTopicLen {
		return fmt.Errorf("topic too long")
	}
	if len(req.Text) > maxTextLen {
		return fmt.Errorf("text too long")
	}

	if validation.HasDangerousCharacters(req.Topic) {
		return fmt.Errorf("topic contains forbidden characters")
	}

	if len(req.Receivers) > 0 {
		for _, r := range req.Receivers {
			email := strings.TrimSpace(r.Email)
			if email != "" {
				if validation.HasDangerousCharacters(email) {
					return fmt.Errorf("receiver email contains forbidden characters: %s", email)
				}
			}
		}
	}

	if len(req.Files) > defaultLimitFiles {
		return fmt.Errorf("too many files")
	}
	for _, f := range req.Files {
		size, _ := strconv.ParseInt(f.Size, 10, 64)
		if size < 0 || size > maxFileSize {
			return fmt.Errorf("file size invalid or too large: %s", f.Name)
		}
		if _, ok := allowedFileTypes[f.FileType]; !ok {
			return fmt.Errorf("unsupported file type: %s", f.FileType)
		}
		base := filepath.Base(f.Name)
		if base != f.Name || strings.Contains(f.Name, "..") {
			return fmt.Errorf("invalid file name: %s", f.Name)
		}
		if validation.HasDangerousCharacters(f.StoragePath) {
			return fmt.Errorf("invalid storage path for file: %s", f.Name)
		}
		if validation.HasDangerousCharacters(f.Name) {
			return fmt.Errorf("invalid file name: %s", f.Name)
		}
	}

	return nil
}

func (s *Server) DeleteDraft(ctx context.Context, req *pb.DeleteDraftRequest) (*pb.DeleteDraftResponse, error) {
	const op = "messagesservice.DeleteDraft"
	log := logger.GetLogger(ctx)
	log.Debug("handle messages/delete-draft")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if req.DraftId == "" {
		return nil, status.Error(codes.InvalidArgument, "draft id is required")
	}

	draftID, err := strconv.ParseInt(req.DraftId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid draft id format")
	}

	belongs, err := s.messageUCase.IsDraftBelongsToUser(ctx, draftID, profileID)
	if err != nil {
		log.Error(op + ": failed to check draft ownership: " + err.Error())
		return nil, status.Error(codes.Internal, "could not verify draft ownership")
	}

	if !belongs {
		return nil, status.Error(codes.PermissionDenied, "draft not found or access denied")
	}

	err = s.messageUCase.DeleteDraft(ctx, draftID, profileID)
	if err != nil {
		log.Error(op + ": failed to delete draft: " + err.Error())
		return nil, status.Error(codes.Internal, "could not delete draft")
	}

	log.Debug("draft deleted successfully", "draft_id", draftID)
	return &pb.DeleteDraftResponse{
		Success: true,
	}, nil
}

func (s *Server) SendDraft(ctx context.Context, req *pb.SendDraftRequest) (*pb.SendDraftResponse, error) {
	const op = "messagesservice.SendDraft"
	log := logger.GetLogger(ctx)
	log.Debug("handle messages/send-draft")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	draftID, err := strconv.ParseInt(req.DraftId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid draft id")
	}

	belongs, err := s.messageUCase.IsDraftBelongsToUser(ctx, draftID, profileID)
	if err != nil {
		log.Error(op + ": failed to check draft ownership: " + err.Error())
		return nil, status.Error(codes.Internal, "could not verify draft ownership")
	}

	if !belongs {
		return nil, status.Error(codes.PermissionDenied, "draft not found or access denied")
	}

	err = s.messageUCase.SendDraft(ctx, draftID, profileID)
	if err != nil {
		log.Error(op + ": failed to send draft: " + err.Error())
		return nil, status.Error(codes.Internal, "could not send draft")
	}

	return &pb.SendDraftResponse{
		Success:   true,
		MessageId: req.DraftId,
	}, nil
}
