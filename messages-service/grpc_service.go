package messages_service

import (
	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/http-server/middleware/logger"
	"2025_2_a4code/internal/lib/session"
	"2025_2_a4code/internal/lib/validation"
	"context"
	"fmt"
	"net/mail"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	pb "2025_2_a4code/pkg/messagesproto"

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
	FindByProfileIDWithKeysetPagination(ctx context.Context, profileID, lastMessageID int64, lastDatetime time.Time, limit int) ([]domain.Message, error)
	GetMessagesInfoWithPagination(ctx context.Context, profileID int64) (domain.Messages, error)
	FindSentMessagesByProfileIDWithKeysetPagination(ctx context.Context, profileID int64, lastMessageID int64, lastDatetime time.Time, limit int) ([]domain.Message, error)
	GetSentMessagesInfoWithPagination(ctx context.Context, profileID int64) (domain.Messages, error)
	FindFullByMessageID(ctx context.Context, messageID int64, profileID int64) (domain.FullMessage, error)
	MarkMessageAsRead(ctx context.Context, messageID int64, profileID int64) error
	SaveMessage(ctx context.Context, receiverProfileEmail string, senderBaseProfileID int64, topic, text string) (int64, error)
	SaveFile(ctx context.Context, messageID int64, fileName, fileType, storagePath string, size int64) (fileID int64, err error)
	SaveThreadIdToMessage(ctx context.Context, messageID int64, threadID int64) error
	SaveThread(ctx context.Context, messageID int64) (threadID int64, err error)
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

func (s *Server) Inbox(ctx context.Context, req *pb.InboxRequest) (*pb.InboxResponse, error) {
	const op = "messagesservice.Inbox"
	log := logger.GetLogger(ctx)
	log.Debug("handle messages/inbox")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
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

	messages, err := s.messageUCase.FindByProfileIDWithKeysetPagination(ctx, profileID, lastMessageID, lastDatetime, limit)
	if err != nil {
		log.Error(op+": failed to get messages: ", err.Error())
		return nil, status.Error(codes.Internal, "could not get messages")
	}

	messagesInfo, err := s.messageUCase.GetMessagesInfoWithPagination(ctx, profileID)
	if err != nil {
		log.Error(op+": failed to get messages info: ", err.Error())
		return nil, status.Error(codes.Internal, "could not get messages info")
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

	return &pb.InboxResponse{
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

	fullMessage, err := s.messageUCase.FindFullByMessageID(ctx, messageID, profileID)
	if err != nil {
		log.Error(op+": failed to get message: ", err.Error())
		return nil, status.Error(codes.Internal, "could not get message")
	}

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

	threadRoot, err := strconv.ParseInt(req.ThreadRoot, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid thread root")
	}

	var messageID int64
	for _, receiver := range req.Receivers {
		msgID, err := s.messageUCase.SaveMessage(ctx, receiver.Email, profileID, req.Topic, req.Text)
		if err != nil {
			log.Error(op+": failed to save message: ", err.Error())
			return nil, status.Error(codes.Internal, "could not save message")
		}

		if err := s.messageUCase.SaveThreadIdToMessage(ctx, msgID, threadRoot); err != nil {
			log.Error(op+": failed to save thread id: ", err.Error())
			return nil, status.Error(codes.Internal, "could not save thread id")
		}

		for _, file := range req.Files {
			size, _ := strconv.ParseInt(file.Size, 10, 64)
			_, err = s.messageUCase.SaveFile(ctx, msgID, file.Name, file.FileType, file.StoragePath, size)
			if err != nil {
				log.Error(op+": failed to save file: ", err.Error())
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

	var messageID int64
	for _, receiver := range req.Receivers {
		msgID, err := s.messageUCase.SaveMessage(ctx, receiver.Email, profileID, req.Topic, req.Text)
		if err != nil {
			log.Error(op+": failed to save message: ", err.Error())
			return nil, status.Error(codes.Internal, "could not save message")
		}

		threadID, err := s.messageUCase.SaveThread(ctx, msgID)
		if err != nil {
			log.Error(op+": failed to save thread: ", err.Error())
			return nil, status.Error(codes.Internal, "could not save thread")
		}

		if err := s.messageUCase.SaveThreadIdToMessage(ctx, msgID, threadID); err != nil {
			log.Error(op+": failed to save thread id: ", err.Error())
			return nil, status.Error(codes.Internal, "could not save thread id")
		}

		for _, file := range req.Files {
			size, _ := strconv.ParseInt(file.Size, 10, 64)
			_, err = s.messageUCase.SaveFile(ctx, msgID, file.Name, file.FileType, file.StoragePath, size)
			if err != nil {
				log.Error(op+": failed to save file: ", err.Error())
				return nil, status.Error(codes.Internal, "could not save file")
			}
		}

		messageID = msgID
	}

	return &pb.SendResponse{
		MessageId: strconv.FormatInt(messageID, 10),
	}, nil
}

func (s *Server) Sent(ctx context.Context, req *pb.SentRequest) (*pb.SentResponse, error) {
	const op = "messagesservice.Sent"
	log := logger.GetLogger(ctx)
	log.Debug("handle messages/sent")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
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

	messages, err := s.messageUCase.FindSentMessagesByProfileIDWithKeysetPagination(ctx, profileID, lastMessageID, lastDatetime, limit)
	if err != nil {
		log.Error(op+": failed to get sent messages: ", err.Error())
		return nil, status.Error(codes.Internal, "could not get sent messages")
	}

	messagesInfo, err := s.messageUCase.GetSentMessagesInfoWithPagination(ctx, profileID)
	if err != nil {
		log.Error(op+": failed to get sent messages info: ", err.Error())
		return nil, status.Error(codes.Internal, "could not get sent messages info")
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

	return &pb.SentResponse{
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
	if validation.HasDangerousCharacters(req.Text) {
		return fmt.Errorf("text contains forbidden characters")
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
	if validation.HasDangerousCharacters(req.Text) {
		return fmt.Errorf("text contains forbidden characters")
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
