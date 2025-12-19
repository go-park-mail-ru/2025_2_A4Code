package messages_service

import (
	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/lib/metrics"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"2025_2_a4code/internal/http-server/middleware/logger"
	"2025_2_a4code/internal/lib/session"
	"2025_2_a4code/internal/lib/validation"

	pb "2025_2_a4code/messages-service/pkg/messagesproto"

	"github.com/minio/minio-go/v7"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedMessagesServiceServer
	messageUCase      MessageUsecase
	avatarUCase       AvatarUsecase
	JWTSecret         []byte
	ingestSecret      string
	smtpAddr          string
	localDomain       string
	smtpSkipTLS       bool
	minioClient       *minio.Client
	attachmentsBucket string
}

type MessageUsecase interface {
	// базовые методы для сообщений
	FindByMessageID(ctx context.Context, messageID int64) (*domain.Message, error)
	FindFullByMessageID(ctx context.Context, messageID int64, profileID int64) (domain.FullMessage, error)
	SaveMessage(ctx context.Context, receiverProfileEmail string, senderProfileID int64, topic, text string) (int64, error)
	EnsureBaseProfile(ctx context.Context, username, domain string) (int64, error)
	EnsureProfileForBase(ctx context.Context, baseProfileID int64, displayName string) error
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
	SaveOutgoingExternalMessage(ctx context.Context, senderProfileID int64, topic, text string, receivers []string) (int64, error)
	GetProfileEmail(ctx context.Context, profileID int64) (string, error)

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
	maxFileSize       = 40_000_000 // 40 MB (decimal)
	maxTotalFilesSize = 40_000_000 // 40 MB (decimal)
	defaultLimitFiles = 20
)

var allowedFileTypes = map[string]struct{}{
	"image/jpeg":      {},
	"image/png":       {},
	"application/pdf": {},
	"text/plain":      {},
}

func New(messageUCase MessageUsecase, avatarUCase AvatarUsecase, secret []byte, minioClient *minio.Client, attachmentsBucket string, ingestSecret string) *Server {
	if strings.TrimSpace(attachmentsBucket) == "" {
		attachmentsBucket = "attachments"
	}
	return &Server{
		messageUCase:      messageUCase,
		avatarUCase:       avatarUCase,
		JWTSecret:         secret,
		ingestSecret:      strings.TrimSpace(ingestSecret),
		smtpAddr:          smtpAddrFromEnv(),
		localDomain:       localDomainFromEnv(),
		smtpSkipTLS:       smtpSkipTLSFromEnv(),
		minioClient:       minioClient,
		attachmentsBucket: attachmentsBucket,
	}
}

func (s *Server) getProfileID(ctx context.Context) (int64, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, status.Error(codes.Unauthenticated, "Метаданные отсутствуют")
	}

	tokens := md.Get("authorization")
	if len(tokens) == 0 {
		return 0, status.Error(codes.Unauthenticated, "Отсутствует токен авторизации")
	}

	tokenString := strings.TrimPrefix(tokens[0], "Bearer ")
	return session.GetProfileIDFromTokenString(tokenString, s.JWTSecret, "access")
}

func (s *Server) MessagePage(ctx context.Context, req *pb.MessagePageRequest) (*pb.MessagePageResponse, error) {
	const op = "messagesservice.MessagePage"

	timer := prometheus.NewTimer(metrics.MessagesOperationsDuration.WithLabelValues("messages", "get_message"))
	defer timer.ObserveDuration()

	log := logger.GetLogger(ctx)
	log.Debug("handle messages/{message_id}")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "get_message", "error").Inc()
		return nil, status.Error(codes.Unauthenticated, "Не авторизован")
	}

	messageID, err := strconv.ParseInt(req.MessageId, 10, 64)
	if err != nil {
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "get_message", "error").Inc()
		return nil, status.Error(codes.InvalidArgument, "Некорректный идентификатор сообщения")
	}

	ok, err := s.messageUCase.IsUsersMessage(ctx, messageID, profileID)
	if err != nil {
		log.Error(op + ": failed to check if it is users message: " + err.Error())
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "get_message", "error").Inc()
		return nil, status.Error(codes.Internal, "Не удалось получить письмо")
	}
	if !ok {
		log.Debug(op + ": unpermitted access to message")
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "get_message", "error").Inc()
		return nil, status.Error(codes.PermissionDenied, "Доступ запрещен")
	}

	fullMessage, err := s.messageUCase.FindFullByMessageID(ctx, messageID, profileID)
	if err != nil {
		log.Error(op + ": failed to get message: " + err.Error())
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "get_message", "error").Inc()
		return nil, status.Error(codes.Internal, "Не удалось получить письмо")
	}

	// Помечаем как прочитанное без дополнительной проверки, чтобы не пропускать обновление статуса
	if err := s.messageUCase.MarkMessageAsRead(ctx, messageID, profileID); err != nil {
		log.Warn("failed to mark message as read: " + err.Error())
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
	var pbReceivers []*pb.Receiver
	for _, r := range fullMessage.Receivers {
		pbReceivers = append(pbReceivers, &pb.Receiver{Email: r})
	}

	metrics.MessagesOperationsTotal.WithLabelValues("messages", "get_message", "ok").Inc()

	return &pb.MessagePageResponse{
		Message: &pb.FullMessage{
			Topic:     fullMessage.Topic,
			Text:      fullMessage.Text,
			Datetime:  fullMessage.Datetime.Format(time.RFC3339),
			ThreadId:  fullMessage.ThreadRoot,
			Sender:    s.domainSenderToProto(&fullMessage.Sender),
			Files:     pbFiles,
			Receivers: pbReceivers,
		},
	}, nil
}

func (s *Server) Reply(ctx context.Context, req *pb.ReplyRequest) (*pb.ReplyResponse, error) {
	const op = "messagesservice.Reply"

	timer := prometheus.NewTimer(metrics.MessagesOperationsDuration.WithLabelValues("messages", "reply"))
	defer timer.ObserveDuration()

	log := logger.GetLogger(ctx)
	log.Debug("handle messages/reply")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "reply", "error").Inc()
		return nil, status.Error(codes.Unauthenticated, "Не авторизован")
	}

	if err := s.validateReplyRequest(req); err != nil {
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "reply", "error").Inc()
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	threadRoot, err := s.resolveThreadRoot(ctx, req, profileID, log)
	if err != nil {
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "reply", "error").Inc()
		return nil, err
	}

	safeTopic, safeText := sanitizeContent(req.Topic, req.Text)

	var messageID int64
	senderEmail, err := s.messageUCase.GetProfileEmail(ctx, profileID)
	if err != nil {
		log.Error(op + ": failed to resolve sender email: " + err.Error())
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "reply", "error").Inc()
		return nil, status.Error(codes.Internal, "Не удалось определить отправителя")
	}

	for _, receiver := range req.Receivers {
		email := strings.TrimSpace(receiver.Email)
		if s.isLocalDomain(email) {
			msgID, err := s.messageUCase.ReplyToMessage(ctx, email, profileID, threadRoot, safeTopic, safeText)
			if err != nil {
				log.Error(op + ": failed to reply to message: " + err.Error())
				metrics.MessagesOperationsTotal.WithLabelValues("messages", "reply", "error").Inc()
				return nil, status.Error(codes.Internal, "Не удалось отправить ответ")
			}

			for _, file := range req.Files {
				size, _ := strconv.ParseInt(file.Size, 10, 64)
				_, err = s.messageUCase.SaveFile(ctx, msgID, file.Name, file.FileType, file.StoragePath, size)
				if err != nil {
					log.Error(op + ": failed to save file: " + err.Error())
					metrics.MessagesOperationsTotal.WithLabelValues("messages", "reply", "error").Inc()
					return nil, status.Error(codes.Internal, "Не удалось сохранить файл")
				}

				metrics.FileSize.WithLabelValues("messages", file.FileType).Observe(float64(size))
				metrics.FileOperations.WithLabelValues("messages", "upload", "ok").Inc()
			}

			metrics.MessagesSentTotal.WithLabelValues("reply").Inc()
			messageID = msgID
		} else {
			if err := s.sendExternalMail(senderEmail, email, safeTopic, safeText, req.Files); err != nil {
				log.Error(op + ": failed to send external reply: " + err.Error())
				metrics.MessagesOperationsTotal.WithLabelValues("messages", "reply", "error").Inc()
				return nil, status.Error(codes.Internal, "Не удалось отправить внешний ответ")
			}
			msgID, err := s.messageUCase.SaveOutgoingExternalMessage(ctx, profileID, safeTopic, safeText, []string{email})
			if err != nil {
				log.Error(op + ": failed to save external outgoing reply: " + err.Error())
				metrics.MessagesOperationsTotal.WithLabelValues("messages", "reply", "error").Inc()
				return nil, status.Error(codes.Internal, "Не удалось сохранить внешний ответ")
			}

			for _, file := range req.Files {
				size, _ := strconv.ParseInt(file.Size, 10, 64)
				if _, err := s.messageUCase.SaveFile(ctx, msgID, file.Name, file.FileType, file.StoragePath, size); err != nil {
					log.Error(op + ": failed to save external reply file: " + err.Error())
					metrics.MessagesOperationsTotal.WithLabelValues("messages", "reply", "error").Inc()
					return nil, status.Error(codes.Internal, "Не удалось сохранить файл внешнего ответа")
				}
				metrics.FileSize.WithLabelValues("messages", file.FileType).Observe(float64(size))
				metrics.FileOperations.WithLabelValues("messages", "upload", "ok").Inc()
			}

			metrics.MessagesSentTotal.WithLabelValues("reply_external").Inc()
			messageID = msgID
		}
	}

	metrics.MessagesOperationsTotal.WithLabelValues("messages", "reply", "ok").Inc()

	return &pb.ReplyResponse{
		MessageId: strconv.FormatInt(messageID, 10),
	}, nil
}

func (s *Server) Send(ctx context.Context, req *pb.SendRequest) (*pb.SendResponse, error) {
	const op = "messagesservice.Send"

	timer := prometheus.NewTimer(metrics.MessagesOperationsDuration.WithLabelValues("messages", "send"))
	defer timer.ObserveDuration()

	log := logger.GetLogger(ctx)
	log.Debug("handle messages/send")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "send", "error").Inc()
		return nil, status.Error(codes.Unauthenticated, "Не авторизован")
	}

	if err := s.validateSendRequest(req); err != nil {
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "send", "error").Inc()
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	safeTopic, safeText := sanitizeContent(req.Topic, req.Text)

	var messageID int64
	senderEmail, err := s.messageUCase.GetProfileEmail(ctx, profileID)
	if err != nil {
		log.Error(op + ": failed to resolve sender email: " + err.Error())
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "send", "error").Inc()
		return nil, status.Error(codes.Internal, "Не удалось определить отправителя")
	}

	for _, receiver := range req.Receivers {
		email := strings.TrimSpace(receiver.Email)
		if s.isLocalDomain(email) {
			msgID, err := s.messageUCase.SendMessage(ctx, email, profileID, safeTopic, safeText)
			if err != nil {
				log.Error(op + ": failed to send message: " + err.Error())
				metrics.MessagesOperationsTotal.WithLabelValues("messages", "send", "error").Inc()
				return nil, status.Error(codes.Internal, "Не удалось отправить письмо")
			}

			if messageID == 0 {
				threadID, err := s.messageUCase.SaveThread(ctx, msgID)
				if err != nil {
					log.Error(op + ": failed to save thread: " + err.Error())
					metrics.MessagesOperationsTotal.WithLabelValues("messages", "send", "error").Inc()
					return nil, status.Error(codes.Internal, "Не удалось создать цепочку")
				}

				if err := s.messageUCase.SaveThreadIdToMessage(ctx, msgID, threadID); err != nil {
					log.Error(op + ": failed to save thread id: " + err.Error())
					metrics.MessagesOperationsTotal.WithLabelValues("messages", "send", "error").Inc()
					return nil, status.Error(codes.Internal, "Не удалось привязать цепочку")
				}
			}

			for _, file := range req.Files {
				size, _ := strconv.ParseInt(file.Size, 10, 64)
				_, err = s.messageUCase.SaveFile(ctx, msgID, file.Name, file.FileType, file.StoragePath, size)
				if err != nil {
					log.Error(op + ": failed to save file: " + err.Error())
					metrics.MessagesOperationsTotal.WithLabelValues("messages", "send", "error").Inc()
					return nil, status.Error(codes.Internal, "Не удалось сохранить файл")
				}

				metrics.FileSize.WithLabelValues("messages", file.FileType).Observe(float64(size))
				metrics.FileOperations.WithLabelValues("messages", "upload", "ok").Inc()
			}

			metrics.MessagesSentTotal.WithLabelValues("send").Inc()
			messageID = msgID
		} else {
			if err := s.sendExternalMail(senderEmail, email, safeTopic, safeText, req.Files); err != nil {
				log.Error(op + ": failed to send external message: " + err.Error())
				metrics.MessagesOperationsTotal.WithLabelValues("messages", "send", "error").Inc()
				return nil, status.Error(codes.Internal, "Не удалось отправить внешнее письмо")
			}

			msgID, err := s.messageUCase.SaveOutgoingExternalMessage(ctx, profileID, safeTopic, safeText, []string{email})
			if err != nil {
				log.Error(op + ": failed to save external outgoing message: " + err.Error())
				metrics.MessagesOperationsTotal.WithLabelValues("messages", "send", "error").Inc()
				return nil, status.Error(codes.Internal, "Не удалось сохранить внешнее письмо")
			}

			for _, file := range req.Files {
				size, _ := strconv.ParseInt(file.Size, 10, 64)
				if _, err := s.messageUCase.SaveFile(ctx, msgID, file.Name, file.FileType, file.StoragePath, size); err != nil {
					log.Error(op + ": failed to save external file: " + err.Error())
					metrics.MessagesOperationsTotal.WithLabelValues("messages", "send", "error").Inc()
					return nil, status.Error(codes.Internal, "Не удалось сохранить внешний файл")
				}
				metrics.FileSize.WithLabelValues("messages", file.FileType).Observe(float64(size))
				metrics.FileOperations.WithLabelValues("messages", "upload", "ok").Inc()
			}

			metrics.MessagesSentTotal.WithLabelValues("send_external").Inc()
			messageID = msgID
		}
	}

	metrics.MessagesOperationsTotal.WithLabelValues("messages", "send", "ok").Inc()

	return &pb.SendResponse{
		MessageId: strconv.FormatInt(messageID, 10),
	}, nil
}

func (s *Server) Ingest(ctx context.Context, req *pb.IngestRequest) (*pb.IngestResponse, error) {
	const op = "messagesservice.Ingest"
	log := logger.GetLogger(ctx)

	if s.ingestSecret != "" {
		md, _ := metadata.FromIncomingContext(ctx)
		secret := ""
		if md != nil {
			vals := md.Get("ingest-secret")
			if len(vals) > 0 {
				secret = strings.TrimSpace(vals[0])
			}
		}
		if secret == "" || secret != s.ingestSecret {
			return nil, status.Error(codes.Unauthenticated, "Не авторизован")
		}
	}

	senderEmail := strings.TrimSpace(req.SenderEmail)
	if senderEmail == "" || len(req.Receivers) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Требуются отправитель и получатели")
	}

	senderUser, senderDomain, err := splitEmailParts(senderEmail)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Некорректный email отправителя")
	}

	senderProfileID, err := s.messageUCase.EnsureBaseProfile(ctx, senderUser, senderDomain)
	if err != nil {
		log.Error(op + ": failed to ensure sender profile: " + err.Error())
		return nil, status.Error(codes.Internal, "Не удалось сохранить отправителя")
	}
	if err := s.messageUCase.EnsureProfileForBase(ctx, senderProfileID, req.SenderName); err != nil {
		log.Warn(op + ": failed to update sender name: " + err.Error())
	}

	safeSubject, safeText := sanitizeContent(req.Subject, req.Text)

	seen := make(map[string]struct{})
	var messageIDs []string
	for _, rcpt := range req.Receivers {
		email := strings.TrimSpace(rcpt)
		if email == "" {
			continue
		}
		key := strings.ToLower(email)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		msgID, err := s.messageUCase.SaveMessage(ctx, email, senderProfileID, safeSubject, safeText)
		if err != nil {
			log.Warn(op+": failed to save message: "+err.Error(), slog.String("receiver", email))
			continue
		}

		if threadID, err := s.messageUCase.SaveThread(ctx, msgID); err == nil {
			if err := s.messageUCase.SaveThreadIdToMessage(ctx, msgID, threadID); err != nil {
				log.Warn(op + ": failed to attach thread id: " + err.Error())
			}
		} else {
			log.Warn(op + ": failed to create thread: " + err.Error())
		}

		for _, f := range req.Files {
			size, _ := strconv.ParseInt(f.Size, 10, 64)
			if _, err := s.messageUCase.SaveFile(ctx, msgID, f.Name, f.FileType, f.StoragePath, size); err != nil {
				log.Warn(op + ": failed to save file: " + err.Error())
				continue
			}
			metrics.FileSize.WithLabelValues("messages", f.FileType).Observe(float64(size))
			metrics.FileOperations.WithLabelValues("messages", "upload", "ok").Inc()
		}

		messageIDs = append(messageIDs, strconv.FormatInt(msgID, 10))
	}

	return &pb.IngestResponse{MessageIds: messageIDs}, nil
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
		return 0, status.Error(codes.InvalidArgument, "Не указан корень цепочки")
	}

	rootMessageID, err := strconv.ParseInt(rootMessageRaw, 10, 64)
	if err != nil {
		return 0, status.Error(codes.InvalidArgument, "Некорректный идентификатор корневого письма")
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
		return 0, status.Error(codes.Internal, "Не удалось создать цепочку")
	}

	if err := s.messageUCase.SaveThreadIdToMessage(ctx, rootMessageID, threadRoot); err != nil {
		log.Error("failed to attach thread to message: " + err.Error())
		return 0, status.Error(codes.Internal, "Не удалось привязать цепочку к письму")
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
	var totalSize int64
	for _, f := range req.Files {
		size, _ := strconv.ParseInt(f.Size, 10, 64)
		if size < 0 || size > maxFileSize {
			return fmt.Errorf("file size invalid or too large: %s", f.Name)
		}
		totalSize += size
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
	if totalSize > maxTotalFilesSize {
		return fmt.Errorf("total attachments size exceeds %d MB", maxTotalFilesSize/(1024*1024))
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
		return nil, status.Error(codes.Unauthenticated, "Не авторизован")
	}

	messageID, err := strconv.ParseInt(req.MessageId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Некорректный идентификатор сообщения")
	}

	if err := s.messageUCase.MarkMessageAsSpam(ctx, messageID, profileID); err != nil {
		log.Warn("failed to mark message as spam: " + err.Error())
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "mark_spam", "error").Inc()
		return nil, status.Error(codes.Internal, "Не удалось пометить письмо как спам")
	}

	metrics.MessagesOperationsTotal.WithLabelValues("messages", "mark_spam", "ok").Inc()
	return &pb.MarkAsSpamResponse{}, nil
}

func (s *Server) MoveToFolder(ctx context.Context, req *pb.MoveToFolderRequest) (*pb.MoveToFolderResponse, error) {
	const op = "messagesservice.MoveToFolder"
	log := logger.GetLogger(ctx)
	log.Debug("handle messages/move-to-folder")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Не авторизован")
	}

	messageID, err := strconv.ParseInt(req.MessageId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Некорректный идентификатор сообщения")
	}

	folderID, err := strconv.ParseInt(req.FolderId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Некорректный идентификатор папки")
	}

	if err := s.messageUCase.MoveToFolder(ctx, profileID, messageID, folderID); err != nil {
		log.Error(op + ": failed to move message to folder: " + err.Error())
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "move_to_folder", "error").Inc()
		return nil, status.Error(codes.Internal, "Не удалось переместить письмо")
	}

	metrics.MessagesOperationsTotal.WithLabelValues("messages", "move_to_folder", "ok").Inc()
	return &pb.MoveToFolderResponse{}, nil
}

func (s *Server) CreateFolder(ctx context.Context, req *pb.CreateFolderRequest) (*pb.CreateFolderResponse, error) {
	const op = "messagesservice.CreateFolder"
	log := logger.GetLogger(ctx)
	log.Debug("handle messages/create-folder")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Не авторизован")
	}

	if err := s.validateFolderName(req.FolderName); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	folder, err := s.messageUCase.CreateFolder(ctx, profileID, req.FolderName)
	if err != nil {
		if errors.Is(err, domain.ErrFolderExists) {
			return nil, status.Error(codes.AlreadyExists, "Папка с таким именем уже существует")
		}
		log.Error(op + ": failed to create folder: " + err.Error())
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "create_folder", "error").Inc()
		return nil, status.Error(codes.Internal, "Не удалось создать папку")
	}

	metrics.MessagesOperationsTotal.WithLabelValues("messages", "create_folder", "ok").Inc()
	return &pb.CreateFolderResponse{
		FolderId:   strconv.FormatInt(folder.ID, 10),
		FolderName: folder.Name,
		FolderType: string(folder.Type),
	}, nil
}

func (s *Server) validateFolderName(folderName string) error {
	if folderName == "" {
		return fmt.Errorf("Название папки не может быть пустым")
	}

	if len(folderName) > 50 {
		return fmt.Errorf("Название папки слишком длинное (максимум 50 символов)")
	}

	if len(folderName) < 1 {
		return fmt.Errorf("Название папки слишком короткое")
	}

	if validation.HasDangerousCharacters(folderName) {
		return fmt.Errorf("Название папки содержит недопустимые символы")
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
		return fmt.Errorf("Название папки '%s' зарезервировано для системных папок", folderName)
	}

	return nil
}

func (s *Server) GetFolder(ctx context.Context, req *pb.GetFolderRequest) (*pb.GetFolderResponse, error) {
	const op = "messagesservice.GetFolder"
	log := logger.GetLogger(ctx)
	log.Debug("handle messages/get-folder")

	timer := prometheus.NewTimer(metrics.MessagesOperationsDuration.WithLabelValues("messages", "get_folder_messages"))
	defer timer.ObserveDuration()

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "get_folder_messages", "error").Inc()
		return nil, status.Error(codes.Unauthenticated, "Не авторизован")
	}

	folderID, err := strconv.ParseInt(req.FolderId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Некорректный идентификатор папки")
	}

	var lastMessageID int64
	var lastDatetime time.Time
	limit := 100

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
		if l, err := strconv.Atoi(req.Limit); err == nil && l > 0 {
			if l > 200 {
				limit = 200
			} else {
				limit = l
			}
		}
	}

	messages, err := s.messageUCase.GetFolderMessagesWithKeysetPagination(ctx, profileID, folderID, lastMessageID, lastDatetime, limit)
	if err != nil {
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "get_folder_messages", "error").Inc()
		log.Error(op + ": failed to get folder messages: " + err.Error())
		return nil, status.Error(codes.Internal, "Не удалось получить письма папки")
	}

	messagesInfo, err := s.messageUCase.GetFolderMessagesInfo(ctx, profileID, folderID)
	if err != nil {
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "get_folder_messages", "error").Inc()
		log.Error(op + ": failed to get folder messages info: " + err.Error())
		return nil, status.Error(codes.Internal, "Не удалось получить данные папки")
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

	metrics.MessagesOperationsTotal.WithLabelValues("messages", "get_folder_messages", "ok").Inc()
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

	timer := prometheus.NewTimer(metrics.MessagesOperationsDuration.WithLabelValues("messages", "get_folders"))
	defer timer.ObserveDuration()

	log := logger.GetLogger(ctx)
	log.Debug("handle messages/get-folders")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "get_folders", "error").Inc()
		return nil, status.Error(codes.Unauthenticated, "Не авторизован")
	}

	folders, err := s.messageUCase.GetUserFolders(ctx, profileID)
	if err != nil {
		log.Error(op + ": failed to get user folders: " + err.Error())
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "get_folders", "error").Inc()
		return nil, status.Error(codes.Internal, "Не удалось получить список папок")
	}

	pbFolders := make([]*pb.Folder, 0, len(folders))
	for _, folder := range folders {
		pbFolders = append(pbFolders, &pb.Folder{
			FolderId:   strconv.FormatInt(folder.ID, 10),
			FolderName: folder.Name,
			FolderType: string(folder.Type),
		})
	}

	metrics.MessagesOperationsTotal.WithLabelValues("messages", "get_folders", "ok").Inc()
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
		return nil, status.Error(codes.Unauthenticated, "Не авторизован")
	}

	folderID, err := strconv.ParseInt(req.FolderId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Некорректный идентификатор папки")
	}

	if err := s.validateFolderName(req.NewFolderName); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	folder, err := s.messageUCase.RenameFolder(ctx, profileID, folderID, req.NewFolderName)
	if err != nil {
		if errors.Is(err, domain.ErrFolderExists) {
			return nil, status.Error(codes.AlreadyExists, "Папка с таким именем уже существует")
		}
		log.Error(op + ": failed to rename folder: " + err.Error())
		return nil, status.Error(codes.Internal, "Не удалось переименовать папку")
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
		return nil, status.Error(codes.Unauthenticated, "Не авторизован")
	}

	folderID, err := strconv.ParseInt(req.FolderId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Некорректный идентификатор папки")
	}

	if err := s.messageUCase.DeleteFolder(ctx, profileID, folderID); err != nil {
		if errors.Is(err, domain.ErrFolderNotFound) {
			return nil, status.Error(codes.NotFound, "Папка не найдена")
		}
		if errors.Is(err, domain.ErrFolderSystem) {
			return nil, status.Error(codes.PermissionDenied, "Системную папку нельзя удалить")
		}
		log.Error(op + ": failed to delete folder: " + err.Error())
		return nil, status.Error(codes.Internal, "Не удалось удалить папку")
	}

	return &pb.DeleteFolderResponse{}, nil
}

func (s *Server) DeleteMessageFromFolder(ctx context.Context, req *pb.DeleteMessageFromFolderRequest) (*pb.DeleteMessageFromFolderResponse, error) {
	const op = "messagesservice.DeleteMessageFromFolder"

	log := logger.GetLogger(ctx)
	log.Debug("handle messages/delete-message-from-folder")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Не авторизован")
	}

	messageID, err := strconv.ParseInt(req.MessageId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Некорректный идентификатор сообщения")
	}

	folderID, err := strconv.ParseInt(req.FolderId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Некорректный идентификатор папки")
	}

	if err := s.messageUCase.DeleteMessageFromFolder(ctx, profileID, messageID, folderID); err != nil {
		log.Error(op + ": failed to delete message from folder: " + err.Error())
		return nil, status.Error(codes.Internal, "Не удалось удалить письмо из папки")
	}

	return &pb.DeleteMessageFromFolderResponse{}, nil
}

func (s *Server) SaveDraft(ctx context.Context, req *pb.SaveDraftRequest) (*pb.SaveDraftResponse, error) {
	const op = "messagesservice.SaveDraft"

	timer := prometheus.NewTimer(metrics.MessagesOperationsDuration.WithLabelValues("messages", "save_draft"))
	defer timer.ObserveDuration()

	log := logger.GetLogger(ctx)
	log.Debug("handle messages/save-draft")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "save_draft", "error").Inc()
		return nil, status.Error(codes.Unauthenticated, "Не авторизован")
	}

	if err := s.validateDraftRequest(req); err != nil {
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "save_draft", "error").Inc()
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
			metrics.MessagesOperationsTotal.WithLabelValues("messages", "save_draft", "error").Inc()
			return nil, status.Error(codes.Internal, "Не удалось сохранить черновик")
		}
		draftID = msgID

		if req.DraftId == "" {
			var threadID int64
			if req.ThreadId != "" {
				threadID, err = strconv.ParseInt(req.ThreadId, 10, 64)
				if err != nil {
					log.Error(op + ": invalid thread id: " + err.Error())
					return nil, status.Error(codes.InvalidArgument, "Некорректный идентификатор цепочки")
				}
			} else {
				threadID, err = s.messageUCase.SaveThread(ctx, draftID)
				if err != nil {
					log.Error(op + ": failed to save thread: " + err.Error())
					return nil, status.Error(codes.Internal, "Не удалось создать цепочку")
				}
			}
			if err := s.messageUCase.SaveThreadIdToMessage(ctx, draftID, threadID); err != nil {
				log.Error(op + ": failed to save thread id: " + err.Error())
				return nil, status.Error(codes.Internal, "Не удалось привязать цепочку")
			}

			draftFolderID, err := s.messageUCase.GetFolderByType(ctx, profileID, "draft")
			if err != nil {
				log.Error(op + ": failed to get draft folder: " + err.Error())
				return nil, status.Error(codes.Internal, "Не удалось получить папку черновиков")
			}

			if err := s.messageUCase.MoveToFolder(ctx, profileID, draftID, draftFolderID); err != nil {
				log.Error(op + ": failed to put draft to folder: " + err.Error())
				return nil, status.Error(codes.Internal, "Не удалось сохранить черновик в папку")
			}
		}

		for _, file := range req.Files {
			size, _ := strconv.ParseInt(file.Size, 10, 64)
			_, err = s.messageUCase.SaveFile(ctx, draftID, file.Name, file.FileType, file.StoragePath, size)
			if err != nil {
				log.Error(op + ": failed to save file: " + err.Error())
				metrics.MessagesOperationsTotal.WithLabelValues("messages", "save_draft", "error").Inc()
				return nil, status.Error(codes.Internal, "Не удалось сохранить файл")
			}

			metrics.FileSize.WithLabelValues("messages", file.FileType).Observe(float64(size))
			metrics.FileOperations.WithLabelValues("messages", "upload_draft", "ok").Inc()
		}
	}

	metrics.MessagesOperationsTotal.WithLabelValues("messages", "save_draft", "ok").Inc()

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
	var totalSize int64
	for _, f := range req.Files {
		size, _ := strconv.ParseInt(f.Size, 10, 64)
		if size < 0 || size > maxFileSize {
			return fmt.Errorf("file size invalid or too large: %s", f.Name)
		}
		totalSize += size
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
	if totalSize > maxTotalFilesSize {
		return fmt.Errorf("total attachments size exceeds %d MB", maxTotalFilesSize/(1024*1024))
	}

	return nil
}

func (s *Server) DeleteDraft(ctx context.Context, req *pb.DeleteDraftRequest) (*pb.DeleteDraftResponse, error) {
	const op = "messagesservice.DeleteDraft"
	log := logger.GetLogger(ctx)
	log.Debug("handle messages/delete-draft")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Не авторизован")
	}

	if req.DraftId == "" {
		return nil, status.Error(codes.InvalidArgument, "Не указан идентификатор черновика")
	}

	draftID, err := strconv.ParseInt(req.DraftId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Некорректный формат идентификатора черновика")
	}

	belongs, err := s.messageUCase.IsDraftBelongsToUser(ctx, draftID, profileID)
	if err != nil {
		log.Error(op + ": failed to check draft ownership: " + err.Error())
		return nil, status.Error(codes.Internal, "Не удалось подтвердить права на черновик")
	}

	if !belongs {
		return nil, status.Error(codes.PermissionDenied, "Черновик не найден или доступ запрещен")
	}

	err = s.messageUCase.DeleteDraft(ctx, draftID, profileID)
	if err != nil {
		log.Error(op + ": failed to delete draft: " + err.Error())
		return nil, status.Error(codes.Internal, "Не удалось удалить черновик")
	}

	log.Debug("draft deleted successfully", "draft_id", draftID)
	return &pb.DeleteDraftResponse{
		Success: true,
	}, nil
}

func (s *Server) SendDraft(ctx context.Context, req *pb.SendDraftRequest) (*pb.SendDraftResponse, error) {
	const op = "messagesservice.SendDraft"

	timer := prometheus.NewTimer(metrics.MessagesOperationsDuration.WithLabelValues("messages", "send_draft"))
	defer timer.ObserveDuration()

	log := logger.GetLogger(ctx)
	log.Debug("handle messages/send-draft")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "send_draft", "error").Inc()
		return nil, status.Error(codes.Unauthenticated, "Не авторизован")
	}

	draftID, err := strconv.ParseInt(req.DraftId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Некорректный идентификатор черновика")
	}

	belongs, err := s.messageUCase.IsDraftBelongsToUser(ctx, draftID, profileID)
	if err != nil {
		log.Error(op + ": failed to check draft ownership: " + err.Error())
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "send_draft", "error").Inc()
		return nil, status.Error(codes.Internal, "Не удалось подтвердить права на черновик")
	}

	if !belongs {
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "send_draft", "error").Inc()
		return nil, status.Error(codes.PermissionDenied, "Черновик не найден или доступ запрещен")
	}

	err = s.messageUCase.SendDraft(ctx, draftID, profileID)
	if err != nil {
		log.Error(op + ": failed to send draft: " + err.Error())
		metrics.MessagesOperationsTotal.WithLabelValues("messages", "send_draft", "error").Inc()
		return nil, status.Error(codes.Internal, "Не удалось отправить черновик")
	}

	metrics.MessagesSentTotal.WithLabelValues("draft_send").Inc()
	metrics.MessagesOperationsTotal.WithLabelValues("messages", "send_draft", "ok").Inc()

	return &pb.SendDraftResponse{
		Success:   true,
		MessageId: req.DraftId,
	}, nil
}

func localDomainFromEnv() string {
	if v := strings.TrimSpace(os.Getenv("LOCAL_DOMAIN")); v != "" {
		return v
	}
	return "flintmail.ru"
}

func smtpAddrFromEnv() string {
	if v := strings.TrimSpace(os.Getenv("SMTP_ADDR")); v != "" {
		return v
	}
	return "exim:25"
}

func smtpSkipTLSFromEnv() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("SMTP_SKIP_TLS")))
	return v == "1" || v == "true" || v == "yes"
}

func (s *Server) isLocalDomain(email string) bool {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(parts[1]), s.localDomain)
}

func splitEmailParts(email string) (string, string, error) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid email")
	}
	user := strings.TrimSpace(parts[0])
	domain := strings.TrimSpace(parts[1])
	if user == "" || domain == "" {
		return "", "", fmt.Errorf("invalid email")
	}
	return user, domain, nil
}

func (s *Server) sendExternalMail(from, to, subject, body string, files []*pb.File) error {
	if strings.TrimSpace(from) == "" {
		from = fmt.Sprintf("no-reply@%s", s.localDomain)
	}
	var msgBytes []byte
	if len(files) == 0 {
		msgBytes = []byte(fmt.Sprintf("From: %s\r\nReply-To: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", from, from, to, subject, body))
	} else {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		boundary := writer.Boundary()
		buf.WriteString(fmt.Sprintf("From: %s\r\nReply-To: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=%s\r\n\r\n", from, from, to, subject, boundary))

		textHeader := textproto.MIMEHeader{}
		textHeader.Set("Content-Type", "text/plain; charset=UTF-8")
		textHeader.Set("Content-Transfer-Encoding", "quoted-printable")
		textPart, _ := writer.CreatePart(textHeader)
		qp := quotedprintable.NewWriter(textPart)
		_, _ = qp.Write([]byte(body))
		_ = qp.Close()

		for _, file := range files {
			data, ct, name, err := s.getAttachmentData(file)
			if err != nil {
				return err
			}
			h := textproto.MIMEHeader{}
			if ct == "" {
				ct = "application/octet-stream"
			}
			h.Set("Content-Type", ct)
			h.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
			h.Set("Content-Transfer-Encoding", "base64")
			part, _ := writer.CreatePart(h)
			writeBase64WithCRLF(part, data)
		}
		writer.Close()
		msgBytes = buf.Bytes()
	}

	host, _, err := net.SplitHostPort(s.smtpAddr)
	if err != nil {
		host = s.smtpAddr
	}

	c, err := smtp.Dial(s.smtpAddr)
	if err != nil {
		return err
	}
	defer c.Close()

	if !s.smtpSkipTLS {
		if ok, _ := c.Extension("STARTTLS"); ok {
			cfg := &tls.Config{ServerName: host, InsecureSkipVerify: s.smtpSkipTLS}
			if err := c.StartTLS(cfg); err != nil {
				return err
			}
		}
	}

	if err := c.Mail(from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}

	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msgBytes); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

func (s *Server) getAttachmentData(file *pb.File) ([]byte, string, string, error) {
	if s.minioClient == nil {
		return nil, "", "", fmt.Errorf("minio client is not configured")
	}
	objectName := strings.TrimLeft(file.GetStoragePath(), "/")
	obj, err := s.minioClient.GetObject(context.Background(), s.attachmentsBucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", "", err
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, "", "", err
	}
	name := file.GetName()
	if strings.TrimSpace(name) == "" {
		name = filepath.Base(objectName)
	}
	return data, file.GetFileType(), name, nil
}

// writeBase64WithCRLF writes base64-encoded data with CRLF line wrapping at 76 chars to satisfy SMTP line length limits.
func writeBase64WithCRLF(w io.Writer, data []byte) {
	const lineLen = 76
	enc := base64.StdEncoding
	encoded := enc.EncodeToString(data)
	for len(encoded) > 0 {
		chunk := encoded
		if len(chunk) > lineLen {
			chunk = encoded[:lineLen]
			encoded = encoded[lineLen:]
		} else {
			encoded = ""
		}
		_, _ = fmt.Fprintf(w, "%s\r\n", chunk)
	}
}
