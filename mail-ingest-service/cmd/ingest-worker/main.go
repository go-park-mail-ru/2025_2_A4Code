package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	pb "2025_2_a4code/messages-service/pkg/messagesproto"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type config struct {
	MailDBDSN         string
	Minio             minioConfig
	BucketName        string
	AttachmentsBucket string
	DomainFilter      string
	MessagesGRPCAddr  string
	IngestSecret      string
}

type minioConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func loadConfig() config {
	return config{
		// Локальный запуск по умолчанию смотрит на проброшенные порты docker-compose.
		MailDBDSN: envOr("MAIL_DB_DSN", "postgres://postgres:postgresql@localhost:8017/maildb?sslmode=disable"),
		Minio: minioConfig{
			Endpoint:  envOr("MAIL_MINIO_ENDPOINT", "localhost:8005"),
			AccessKey: envOr("MAIL_MINIO_USER", "minio"),
			SecretKey: envOr("MAIL_MINIO_PASSWORD", "miniominio"),
			UseSSL:    envOr("MAIL_MINIO_USE_SSL", "") == "true",
		},
		BucketName:        envOr("MAIL_MINIO_BUCKET", "mail-ingest"),
		AttachmentsBucket: envOr("MAIL_MINIO_ATTACHMENTS_BUCKET", "attachments"),
		DomainFilter:      envOr("INGEST_DOMAIN", "flintmail.ru"),
		MessagesGRPCAddr:  envOr("MESSAGES_GRPC_ADDR", "localhost:8002"),
		IngestSecret:      envOr("INGEST_SECRET", "secret"),
	}
}

type ingestMessage struct {
	ID      int64
	From    string
	RcptTo  string
	Subject string
	Path    string
}

type ingestPayload struct {
	SenderEmail string        `json:"sender_email"`
	SenderName  string        `json:"sender_name"`
	Receivers   []string      `json:"receivers"`
	Subject     string        `json:"subject"`
	Text        string        `json:"text"`
	Files       []ingestFile  `json:"files"`
}

type ingestFile struct {
	Name        string `json:"name"`
	FileType    string `json:"file_type"`
	Size        int64  `json:"size"`
	StoragePath string `json:"storage_path"`
}

func main() {
	cfg := loadConfig()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)
	ctx := context.Background()

	mailDB, err := sql.Open("pgx", cfg.MailDBDSN)
	if err != nil {
		log.Error("failed to connect mail db", "err", err)
		os.Exit(1)
	}
	defer mailDB.Close()
	if err := ensureIngestTable(mailDB); err != nil {
		log.Error("failed to ensure ingest table", "err", err)
		os.Exit(1)
	}

	minioClient, err := minio.New(cfg.Minio.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Minio.AccessKey, cfg.Minio.SecretKey, ""),
		Secure: cfg.Minio.UseSSL,
	})
	if err != nil {
		log.Error("failed to init minio", "err", err)
		os.Exit(1)
	}
	if err := ensureBucket(ctx, minioClient, cfg.AttachmentsBucket); err != nil {
		log.Error("failed to ensure attachments bucket", "err", err)
		os.Exit(1)
	}

	conn, err := grpc.Dial(cfg.MessagesGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.WaitForReady(true)))
	if err != nil {
		log.Error("failed to dial messages-service", "err", err)
		os.Exit(1)
	}
	defer conn.Close()

	client := pb.NewMessagesServiceClient(conn)

	for {
		if err := processBatch(ctx, cfg, mailDB, minioClient, client, log); err != nil {
			log.Error("batch error", "err", err)
		}
		time.Sleep(2 * time.Second)
	}
}

func processBatch(ctx context.Context, cfg config, mailDB *sql.DB, minioClient *minio.Client, client pb.MessagesServiceClient, log *slog.Logger) error {
	rows, err := mailDB.QueryContext(ctx, `
		SELECT id, mail_from, rcpt_to, subject, raw_path
		FROM ingest_messages
		ORDER BY id
		LIMIT 5`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var msgs []ingestMessage
	for rows.Next() {
		var m ingestMessage
		if err := rows.Scan(&m.ID, &m.From, &m.RcptTo, &m.Subject, &m.Path); err != nil {
			return err
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(msgs) == 0 {
		return nil
	}

	for _, m := range msgs {
		if err := handleMessage(ctx, cfg, minioClient, client, m, log); err != nil {
			log.Error("failed to handle message", "id", m.ID, "err", err)
			continue
		}
		if _, err := mailDB.ExecContext(ctx, `DELETE FROM ingest_messages WHERE id = $1`, m.ID); err != nil {
			log.Error("failed to delete processed message", "id", m.ID, "err", err)
		}
	}
	return nil
}

func handleMessage(ctx context.Context, cfg config, minioClient *minio.Client, client pb.MessagesServiceClient, m ingestMessage, log *slog.Logger) error {
	raw, err := downloadRaw(ctx, minioClient, cfg.BucketName, m.Path)
	if err != nil {
		return fmt.Errorf("download raw: %w", err)
	}

	rawBytes := []byte(raw)
	parsed, err := mail.ReadMessage(bytes.NewReader(rawBytes))
	if err != nil {
		return fmt.Errorf("parse raw: %w", err)
	}

	bodyBytes, err := io.ReadAll(parsed.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	subject := decodeHeader(parsed.Header.Get("Subject"))
	if subject == "" {
		subject = decodeHeader(m.Subject)
	}

	textBody, attachments := parseMime(parsed.Header.Get("Content-Type"), bodyBytes, log)
	if textBody == "" {
		textBody = extractBodyFromSingle(bodyBytes, parsed.Header.Get("Content-Transfer-Encoding"))
	}

	fromHdr := parsed.Header.Get("From")
	addr, err := mail.ParseAddress(fromHdr)
	if err != nil && strings.TrimSpace(m.From) != "" {
		addr, _ = mail.ParseAddress(m.From)
	}
	if addr == nil {
		return fmt.Errorf("cannot parse from address")
	}

	senderDisplayName := strings.TrimSpace(addr.Name)
	senderLocal, senderDomain := splitEmail(addr.Address)

	payload := ingestPayload{
		SenderEmail: fmt.Sprintf("%s@%s", senderLocal, senderDomain),
		SenderName:  senderDisplayName,
		Subject:     subject,
		Text:        textBody,
	}

	rcpts := splitRcpts(m.RcptTo)
	for _, rcpt := range rcpts {
		rcptLocal, rcptDomain := splitEmail(rcpt)
		if strings.ToLower(rcptDomain) != strings.ToLower(cfg.DomainFilter) {
			continue
		}
		if rcptLocal == "" {
			continue
		}
		receiverEmail := fmt.Sprintf("%s@%s", rcptLocal, rcptDomain)
		payload.Receivers = append(payload.Receivers, receiverEmail)
	}

	for _, att := range attachments {
		storagePath, size, ct, err := storeAttachment(ctx, minioClient, cfg.AttachmentsBucket, att)
		if err != nil {
			log.Warn("failed to store attachment", "err", err)
			continue
		}
		payload.Files = append(payload.Files, ingestFile{
			Name:        att.Name,
			FileType:    ct,
			Size:        size,
			StoragePath: storagePath,
		})
	}

	if len(payload.Receivers) == 0 {
		log.Warn("no receivers matched domain, skipping message", "from", payload.SenderEmail)
		return nil
	}

	if err := sendToMessagesService(ctx, cfg, client, payload, log); err != nil {
		return fmt.Errorf("send to messages-service: %w", err)
	}

	return nil
}

func splitRcpts(rcpt string) []string {
	parts := strings.Split(rcpt, ",")
	var res []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			res = append(res, p)
		}
	}
	return res
}

func splitEmail(email string) (string, string) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return strings.TrimSpace(email), ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func downloadRaw(ctx context.Context, client *minio.Client, bucket, path string) (string, error) {
	obj, err := client.GetObject(ctx, bucket, strings.TrimLeft(path, "/"), minio.GetObjectOptions{})
	if err != nil {
		return "", err
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type inboundAttachment struct {
	Name        string
	ContentType string
	Data        []byte
}

func parseMime(ctHeader string, body []byte, log *slog.Logger) (string, []inboundAttachment) {
	plain, html, atts := walkMime(ctHeader, body, log)
	if plain == "" {
		plain = html
	}
	return plain, atts
}

func walkMime(ctHeader string, body []byte, log *slog.Logger) (plain string, html string, attachments []inboundAttachment) {
	mediaType, params, err := mime.ParseMediaType(ctHeader)
	if err != nil {
		mediaType = ""
	}
	if strings.HasPrefix(strings.ToLower(mediaType), "multipart/") {
		boundary := params["boundary"]
		mr := multipart.NewReader(bytes.NewReader(body), boundary)
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Warn("failed to read multipart part", "err", err)
				break
			}
			partCT := part.Header.Get("Content-Type")
			ctLower := strings.ToLower(strings.TrimSpace(strings.Split(partCT, ";")[0]))

			if strings.HasPrefix(ctLower, "multipart/") {
				partBytes, _ := io.ReadAll(part)
				p, h, at := walkMime(partCT, partBytes, log)
				if plain == "" {
					plain = p
				}
				if html == "" {
					html = h
				}
				attachments = append(attachments, at...)
				continue
			}

			data, err := readPartBytes(part)
			if err != nil {
				log.Warn("failed to read part body", "err", err)
				continue
			}

			filename := decodeHeader(part.FileName())
			if filename == "" {
				if _, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition")); err == nil {
					filename = decodeHeader(params["filename"])
				}
			}

			if strings.HasPrefix(ctLower, "text/plain") && plain == "" {
				plain = strings.TrimSpace(string(data))
				continue
			}
			if strings.HasPrefix(ctLower, "text/html") && html == "" {
				html = strings.TrimSpace(string(data))
				continue
			}
			if len(data) > 0 && filename != "" {
				ct := partCT
				if strings.TrimSpace(ct) == "" {
					ct = "application/octet-stream"
				}
				attachments = append(attachments, inboundAttachment{
					Name:        filename,
					ContentType: ct,
					Data:        data,
				})
			}
		}
		return
	}

	ctLower := strings.ToLower(strings.Split(ctHeader, ";")[0])
	if strings.HasPrefix(ctLower, "text/plain") {
		plain = strings.TrimSpace(extractBodyFromSingle(body, ""))
		return
	}
	if strings.HasPrefix(ctLower, "text/html") {
		html = strings.TrimSpace(string(body))
		return
	}
	if len(body) > 0 {
		attachments = append(attachments, inboundAttachment{
			Name:        "",
			ContentType: ctHeader,
			Data:        body,
		})
	}
	return
}

// extractBody tries to return text/plain part, or text/html as fallback, decoding transfer encodings.
func extractBody(msg *mail.Message, log *slog.Logger) string {
	ctHeader := msg.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(ctHeader)
	if err == nil && strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		mr := multipart.NewReader(msg.Body, boundary)
		var plain, html string
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Warn("failed to read multipart", "err", err)
				break
			}
			pt := part.Header.Get("Content-Type")
			body := decodePartBody(part)
			if strings.HasPrefix(strings.ToLower(pt), "text/plain") && plain == "" {
				plain = body
			}
			if strings.HasPrefix(strings.ToLower(pt), "text/html") && html == "" {
				html = body
			}
		}
		if strings.TrimSpace(plain) != "" {
			return strings.TrimSpace(plain)
		}
		if strings.TrimSpace(html) != "" {
			return strings.TrimSpace(html)
		}
	}

	// singlepart
	body := decodeSingleBody(msg.Body, msg.Header.Get("Content-Transfer-Encoding"))
	return strings.TrimSpace(body)
}

func decodeSingleBody(r io.Reader, cte string) string {
	cte = strings.ToLower(strings.TrimSpace(cte))
	var reader io.Reader = r
	switch cte {
	case "quoted-printable":
		reader = quotedprintable.NewReader(r)
	case "base64":
		reader = base64.NewDecoder(base64.StdEncoding, r)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return ""
	}
	return string(data)
}

func extractBodyFromSingle(body []byte, cte string) string {
	return strings.TrimSpace(decodeSingleBody(bytes.NewReader(body), cte))
}

func decodePartBody(part *multipart.Part) string {
	cte := strings.ToLower(strings.TrimSpace(part.Header.Get("Content-Transfer-Encoding")))
	var reader io.Reader = part
	switch cte {
	case "quoted-printable":
		reader = quotedprintable.NewReader(part)
	case "base64":
		reader = base64.NewDecoder(base64.StdEncoding, part)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return ""
	}
	return string(data)
}

func readPartBytes(part *multipart.Part) ([]byte, error) {
	cte := strings.ToLower(strings.TrimSpace(part.Header.Get("Content-Transfer-Encoding")))
	var reader io.Reader = part
	switch cte {
	case "quoted-printable":
		reader = quotedprintable.NewReader(part)
	case "base64":
		reader = base64.NewDecoder(base64.StdEncoding, part)
	}
	return io.ReadAll(reader)
}

// decodeHeader tries to decode MIME encoded-words (e.g. =?UTF-8?B?...?=) and falls back to the raw value.
func decodeHeader(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	dec := new(mime.WordDecoder)
	if decoded, err := dec.DecodeHeader(v); err == nil {
		return decoded
	}
	return v
}

func storeAttachment(ctx context.Context, client *minio.Client, bucket string, att inboundAttachment) (string, int64, string, error) {
	name := strings.TrimSpace(att.Name)
	if name == "" {
		name = "file.bin"
	}
	objectName := sanitizeStoragePath(name)
	ct := strings.TrimSpace(att.ContentType)
	if parsed, _, err := mime.ParseMediaType(ct); err == nil && strings.TrimSpace(parsed) != "" {
		ct = strings.TrimSpace(parsed)
	}
	if ct == "" {
		ct = "application/octet-stream"
	}
	if len(ct) > 100 {
		ct = ct[:100]
	}
	reader := bytes.NewReader(att.Data)
	info, err := client.PutObject(ctx, bucket, objectName, reader, int64(len(att.Data)), minio.PutObjectOptions{
		ContentType: ct,
	})
	if err != nil {
		return "", 0, ct, err
	}
	return objectName, info.Size, ct, nil
}

func sendToMessagesService(ctx context.Context, cfg config, client pb.MessagesServiceClient, payload ingestPayload, log *slog.Logger) error {
	safe := func(s string) string {
		return strings.ToValidUTF8(s, "")
	}

	var files []*pb.File
	for _, f := range payload.Files {
		files = append(files, &pb.File{
			Name:        safe(f.Name),
			FileType:    safe(f.FileType),
			Size:        strconv.FormatInt(f.Size, 10),
			StoragePath: safe(f.StoragePath),
		})
	}

	req := &pb.IngestRequest{
		SenderEmail: safe(payload.SenderEmail),
		SenderName:  safe(payload.SenderName),
		Receivers:   nil,
		Subject:     safe(payload.Subject),
		Text:        safe(payload.Text),
		Files:       files,
	}
	for _, r := range payload.Receivers {
		req.Receivers = append(req.Receivers, safe(r))
	}

	md := metadata.New(nil)
	if strings.TrimSpace(cfg.IngestSecret) != "" {
		md.Set("ingest-secret", cfg.IngestSecret)
	}
	ctx = metadata.NewOutgoingContext(ctx, md)

	_, err := client.Ingest(ctx, req, grpc.WaitForReady(true))
	return err
}

func sanitizeFileName(name string) string {
	name = path.Base(name)
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, "/", "-")
	// оставляем только разрешённые символы
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	clean := strings.Trim(b.String(), "._-")
	if clean == "" {
		clean = "file"
	}
	// гарантируем расширение
	if !strings.Contains(clean, ".") {
		clean += ".bin"
	}
	return clean
}

// sanitizeStoragePath приводит путь к виду, проходящему CHECK file_storage_path_check.
func sanitizeStoragePath(name string) string {
	safeName := sanitizeFileName(name)
	ts := fmt.Sprintf("%d", time.Now().UnixNano())
	obj := path.Join("attachments", ts, safeName)
	// ограничиваем длину 200 символов (constraint)
	if len(obj) > 200 {
		ext := path.Ext(safeName)
		base := strings.TrimSuffix(safeName, ext)
		maxBase := 200 - len("attachments/"+ts+"/") - len(ext)
		if maxBase < 1 {
			maxBase = 1
		}
		if len(base) > maxBase {
			base = base[:maxBase]
		}
		obj = path.Join("attachments", ts, base+ext)
	}
	return obj
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

func ensureIngestTable(db *sql.DB) error {
	const query = `
CREATE TABLE IF NOT EXISTS ingest_messages (
    id BIGSERIAL PRIMARY KEY,
    mail_from   TEXT,
    rcpt_to     TEXT,
    subject     TEXT,
    raw_path    TEXT NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`
	_, err := db.Exec(query)
	return err
}
