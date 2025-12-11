package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/mail"
	"os"
	"strings"
	"time"

	messageRepository "2025_2_a4code/internal/storage/postgres/message-repository"
	"2025_2_a4code/internal/usecase/message"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type config struct {
	MailDBDSN    string
	MainDBDSN    string
	Minio        minioConfig
	BucketName   string
	DomainFilter string
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
		MailDBDSN: envOr("MAIL_DB_DSN", "postgres://postgres:postgresql@mail-postgres:5432/maildb?sslmode=disable"),
		MainDBDSN: envOr("MAIN_DB_DSN", "postgres://postgres:postgresql@postgres:5432/a4code_db?sslmode=disable"),
		Minio: minioConfig{
			Endpoint:  envOr("MAIL_MINIO_ENDPOINT", "minio:9000"),
			AccessKey: envOr("MAIL_MINIO_USER", "minio"),
			SecretKey: envOr("MAIL_MINIO_PASSWORD", "miniominio"),
			UseSSL:    envOr("MAIL_MINIO_USE_SSL", "") == "true",
		},
		BucketName:   envOr("MAIL_MINIO_BUCKET", "mail-ingest"),
		DomainFilter: envOr("INGEST_DOMAIN", "flintmail.ru"),
	}
}

type ingestMessage struct {
	ID      int64
	From    string
	RcptTo  string
	Subject string
	Path    string
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

	mainDB, err := sql.Open("pgx", cfg.MainDBDSN)
	if err != nil {
		log.Error("failed to connect main db", "err", err)
		os.Exit(1)
	}
	defer mainDB.Close()

	minioClient, err := minio.New(cfg.Minio.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Minio.AccessKey, cfg.Minio.SecretKey, ""),
		Secure: cfg.Minio.UseSSL,
	})
	if err != nil {
		log.Error("failed to init minio", "err", err)
		os.Exit(1)
	}

	msgRepo := messageRepository.New(mainDB)
	msgUcase := message.New(msgRepo)

	for {
		if err := processBatch(ctx, cfg, mailDB, minioClient, msgUcase, log); err != nil {
			log.Error("batch error", "err", err)
		}
		time.Sleep(2 * time.Second)
	}
}

func processBatch(ctx context.Context, cfg config, mailDB *sql.DB, minioClient *minio.Client, msgUcase *message.MessageUcase, log *slog.Logger) error {
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
		if err := handleMessage(ctx, cfg, minioClient, msgUcase, m, log); err != nil {
			log.Error("failed to handle message", "id", m.ID, "err", err)
			continue
		}
		if _, err := mailDB.ExecContext(ctx, `DELETE FROM ingest_messages WHERE id = $1`, m.ID); err != nil {
			log.Error("failed to delete processed message", "id", m.ID, "err", err)
		}
	}
	return nil
}

func handleMessage(ctx context.Context, cfg config, minioClient *minio.Client, msgUcase *message.MessageUcase, m ingestMessage, log *slog.Logger) error {
	raw, err := downloadRaw(ctx, minioClient, cfg.BucketName, m.Path)
	if err != nil {
		return fmt.Errorf("download raw: %w", err)
	}

	parsed, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		return fmt.Errorf("parse raw: %w", err)
	}

	subject := parsed.Header.Get("Subject")
	if subject == "" {
		subject = m.Subject
	}
	bodyBytes, err := io.ReadAll(bufio.NewReader(parsed.Body))
	if err != nil {
		bodyBytes = []byte{}
	}
	body := strings.TrimSpace(string(bodyBytes))

	fromHdr := parsed.Header.Get("From")
	addr, err := mail.ParseAddress(fromHdr)
	if err != nil && strings.TrimSpace(m.From) != "" {
		addr, _ = mail.ParseAddress(m.From)
	}
	if addr == nil {
		return fmt.Errorf("cannot parse from address")
	}

	senderLocal, senderDomain := splitEmail(addr.Address)
	senderBaseID, err := msgUcase.EnsureBaseProfile(ctx, senderLocal, senderDomain)
	if err != nil {
		return fmt.Errorf("ensure sender base profile: %w", err)
	}

	rcpts := splitRcpts(m.RcptTo)
	for _, rcpt := range rcpts {
		rcptLocal, rcptDomain := splitEmail(rcpt)
		if strings.ToLower(rcptDomain) != strings.ToLower(cfg.DomainFilter) {
			continue
		}
		receiverEmail := fmt.Sprintf("%s@%s", rcptLocal, rcptDomain)
		if _, err := msgUcase.SaveMessage(ctx, receiverEmail, senderBaseID, subject, body); err != nil {
			return fmt.Errorf("save message for %s: %w", receiverEmail, err)
		}
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
