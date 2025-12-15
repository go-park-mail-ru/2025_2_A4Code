package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/emersion/go-smtp"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type config struct {
	DBDSN      string
	Minio      minioConfig
	LMTPAddr   string
	HTTPAddr   string
	Domain     string
	BucketName string
}

type minioConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

func loadConfig() config {
	return config{
		DBDSN: envOr("MAIL_DB_DSN", "postgres://postgres:postgresql@mail-postgres:5432/maildb?sslmode=disable"),
		Minio: minioConfig{
			Endpoint:  envOr("MAIL_MINIO_ENDPOINT", "minio:9000"),
			AccessKey: envOr("MAIL_MINIO_USER", "minio"),
			SecretKey: envOr("MAIL_MINIO_PASSWORD", "miniominio"),
			UseSSL:    envOr("MAIL_MINIO_USE_SSL", "") == "true",
		},
		LMTPAddr:   envOr("MAIL_LMTP_ADDR", "0.0.0.0:2525"),
		HTTPAddr:   envOr("MAIL_HTTP_ADDR", ":8085"),
		Domain:     envOr("MAIL_DOMAIN", "mail.local"),
		BucketName: envOr("MAIL_MINIO_BUCKET", "mail-ingest"),
	}
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

type store struct {
	db     *sql.DB
	minio  *minio.Client
	bucket string
	log    *slog.Logger
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

func (s *store) saveMessage(ctx context.Context, from string, rcpts []string, raw []byte) error {
	subject := extractSubject(raw)
	objectName := fmt.Sprintf("incoming/%d.eml", time.Now().UnixNano())

	_, err := s.minio.PutObject(ctx, s.bucket, objectName, bytes.NewReader(raw), int64(len(raw)), minio.PutObjectOptions{
		ContentType: "message/rfc822",
	})
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}

	const query = `
		INSERT INTO ingest_messages (mail_from, rcpt_to, subject, raw_path, received_at)
		VALUES ($1, $2, $3, $4, NOW())`

	rcptJoined := strings.Join(rcpts, ",")
	if _, err := s.db.ExecContext(ctx, query, from, rcptJoined, subject, objectName); err != nil {
		return fmt.Errorf("insert message: %w", err)
	}

	s.log.Info("stored incoming message", "from", from, "rcpt", rcptJoined, "object", objectName, "subject", subject)
	return nil
}

func extractSubject(raw []byte) string {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return ""
	}
	return msg.Header.Get("Subject")
}

type ingestBackend struct {
	store *store
}

func (b *ingestBackend) NewSession(_ *smtp.Conn) (smtp.Session, error) {
	return &ingestSession{store: b.store}, nil
}

type ingestSession struct {
	store *store
	from  string
	rcpts []string
}

func (s *ingestSession) AuthPlain(_, _ string) error {
	return smtp.ErrAuthUnsupported
}

func (s *ingestSession) Mail(from string, _ *smtp.MailOptions) error {
	s.from = from
	return nil
}

func (s *ingestSession) Rcpt(to string, _ *smtp.RcptOptions) error {
	s.rcpts = append(s.rcpts, to)
	return nil
}

func (s *ingestSession) Data(r io.Reader) error {
	raw, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.store.saveMessage(ctx, s.from, s.rcpts, raw)
}

func (s *ingestSession) Reset() {}

func (s *ingestSession) Logout() error { return nil }

func ensureBucket(ctx context.Context, client *minio.Client, bucket string, logger *slog.Logger) error {
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if !exists {
		logger.Info("creating bucket for mail ingest", "bucket", bucket)
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return err
		}
	}
	return nil
}

type messageDTO struct {
	ID         int64     `json:"id"`
	From       string    `json:"from"`
	To         string    `json:"to"`
	Subject    string    `json:"subject"`
	RawPath    string    `json:"raw_path"`
	ReceivedAt time.Time `json:"received_at"`
}

func handleMessages(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		rows, err := db.QueryContext(ctx, `
			SELECT id, mail_from, rcpt_to, subject, raw_path, received_at
			FROM ingest_messages
			ORDER BY id DESC
			LIMIT 50`)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var messages []messageDTO
		for rows.Next() {
			var m messageDTO
			if err := rows.Scan(&m.ID, &m.From, &m.To, &m.Subject, &m.RawPath, &m.ReceivedAt); err != nil {
				http.Error(w, "scan error", http.StatusInternalServerError)
				return
			}
			messages = append(messages, m)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(messages)
	}
}

func main() {
	cfg := loadConfig()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	log.Info("starting mail-ingest-service",
		"lmtp", cfg.LMTPAddr,
		"http", cfg.HTTPAddr,
		"bucket", cfg.BucketName,
		"domain", cfg.Domain)

	db, err := sql.Open("pgx", cfg.DBDSN)
	if err != nil {
		log.Error("failed to connect to ingest database", "err", err)
		os.Exit(1)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	if err := ensureIngestTable(db); err != nil {
		log.Error("failed to ensure ingest table", "err", err)
		os.Exit(1)
	}

	minioClient, err := minio.New(cfg.Minio.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Minio.AccessKey, cfg.Minio.SecretKey, ""),
		Secure: cfg.Minio.UseSSL,
	})
	if err != nil {
		log.Error("failed to init minio client", "err", err)
		os.Exit(1)
	}
	if err := ensureBucket(context.Background(), minioClient, cfg.BucketName, log); err != nil {
		log.Error("failed to ensure bucket", "err", err)
		os.Exit(1)
	}

	st := &store{db: db, minio: minioClient, bucket: cfg.BucketName, log: log}
	backend := &ingestBackend{store: st}

	s := smtp.NewServer(backend)
	s.Network = "tcp"
	s.Addr = cfg.LMTPAddr
	s.Domain = cfg.Domain
	s.LMTP = true
	s.ReadTimeout = 10 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.MaxMessageBytes = 80_000_000 // allow ~80MB to handle overhead over 40MB decimal limit

	go func() {
		log.Info("LMTP server listening", "addr", s.Addr)
		if err := s.ListenAndServe(); err != nil {
			log.Error("LMTP server stopped", "err", err)
			os.Exit(1)
		}
	}()

	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	http.Handle("/messages", handleMessages(db))

	log.Info("HTTP server listening", "addr", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, nil); err != nil {
		log.Error("HTTP server stopped", "err", err)
		os.Exit(1)
	}
}
