package main

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"mime/multipart"
	"os"
	"strings"
	"testing"
	"time"

	"log/slog"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Минимальные обертки для замены методов в тестах

type testMinioClient struct {
	*minio.Client
	mockGetObject    func(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)
	mockPutObject    func(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	mockBucketExists func(ctx context.Context, bucketName string) (bool, error)
}

func (t *testMinioClient) GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error) {
	if t.mockGetObject != nil {
		return t.mockGetObject(ctx, bucketName, objectName, opts)
	}
	return t.Client.GetObject(ctx, bucketName, objectName, opts)
}

func (t *testMinioClient) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	if t.mockPutObject != nil {
		return t.mockPutObject(ctx, bucketName, objectName, reader, objectSize, opts)
	}
	return t.Client.PutObject(ctx, bucketName, objectName, reader, objectSize, opts)
}

func (t *testMinioClient) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	if t.mockBucketExists != nil {
		return t.mockBucketExists(ctx, bucketName)
	}
	return t.Client.BucketExists(ctx, bucketName)
}

type testSQLDB struct {
	*sql.DB
	mockQueryContext       func(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	mockExecContext        func(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	mockQueryRowContext    func(ctx context.Context, query string, args ...interface{}) *sql.Row
	mockBeginTx            func(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
	mockPingContext        func(ctx context.Context) error
	mockClose              func() error
	mockPrepareContext     func(ctx context.Context, query string) (*sql.Stmt, error)
	mockSetConnMaxLifetime func(d time.Duration)
	mockSetMaxIdleConns    func(n int)
	mockSetMaxOpenConns    func(n int)
	mockStats              func() sql.DBStats
	mockDriver             func() driver.Driver
}

func (t *testSQLDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if t.mockQueryContext != nil {
		return t.mockQueryContext(ctx, query, args...)
	}
	return t.DB.QueryContext(ctx, query, args...)
}

func (t *testSQLDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if t.mockExecContext != nil {
		return t.mockExecContext(ctx, query, args...)
	}
	return t.DB.ExecContext(ctx, query, args...)
}

func (t *testSQLDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if t.mockQueryRowContext != nil {
		return t.mockQueryRowContext(ctx, query, args...)
	}
	return t.DB.QueryRowContext(ctx, query, args...)
}

func (t *testSQLDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if t.mockBeginTx != nil {
		return t.mockBeginTx(ctx, opts)
	}
	return t.DB.BeginTx(ctx, opts)
}

func (t *testSQLDB) PingContext(ctx context.Context) error {
	if t.mockPingContext != nil {
		return t.mockPingContext(ctx)
	}
	return t.DB.PingContext(ctx)
}

func (t *testSQLDB) Close() error {
	if t.mockClose != nil {
		return t.mockClose()
	}
	return t.DB.Close()
}

func (t *testSQLDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	if t.mockPrepareContext != nil {
		return t.mockPrepareContext(ctx, query)
	}
	return t.DB.PrepareContext(ctx, query)
}

func (t *testSQLDB) SetConnMaxLifetime(d time.Duration) {
	if t.mockSetConnMaxLifetime != nil {
		t.mockSetConnMaxLifetime(d)
		return
	}
	t.DB.SetConnMaxLifetime(d)
}

func (t *testSQLDB) SetMaxIdleConns(n int) {
	if t.mockSetMaxIdleConns != nil {
		t.mockSetMaxIdleConns(n)
		return
	}
	t.DB.SetMaxIdleConns(n)
}

func (t *testSQLDB) SetMaxOpenConns(n int) {
	if t.mockSetMaxOpenConns != nil {
		t.mockSetMaxOpenConns(n)
		return
	}
	t.DB.SetMaxOpenConns(n)
}

func (t *testSQLDB) Stats() sql.DBStats {
	if t.mockStats != nil {
		return t.mockStats()
	}
	return t.DB.Stats()
}

// Моки

type MockMinioObject struct {
	mock.Mock
	io.ReadCloser
}

func (m *MockMinioObject) Read(p []byte) (int, error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

func (m *MockMinioObject) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockSQLRows struct {
	mock.Mock
}

func (m *MockSQLRows) Next() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockSQLRows) Scan(dest ...interface{}) error {
	args := m.Called(dest...)
	return args.Error(0)
}

func (m *MockSQLRows) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSQLRows) Err() error {
	args := m.Called()
	return args.Error(0)
}

type MockSQLRow struct {
	mock.Mock
}

func (m *MockSQLRow) Scan(dest ...interface{}) error {
	args := m.Called(dest...)
	return args.Error(0)
}

type MockSQLResult struct {
	mock.Mock
}

func (m *MockSQLResult) LastInsertId() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSQLResult) RowsAffected() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

type MockMessageUcase struct {
	mock.Mock
}

func (m *MockMessageUcase) EnsureBaseProfile(ctx context.Context, username, domain string) (int64, error) {
	args := m.Called(ctx, username, domain)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMessageUcase) EnsureProfileForBase(ctx context.Context, baseID int64, displayName string) error {
	args := m.Called(ctx, baseID, displayName)
	return args.Error(0)
}

func (m *MockMessageUcase) SaveMessage(ctx context.Context, receiverEmail string, senderProfileID int64, topic, text string) (int64, error) {
	args := m.Called(ctx, receiverEmail, senderProfileID, topic, text)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMessageUcase) SaveFile(ctx context.Context, messageID int64, fileName, fileType, storagePath string, size int64) (int64, error) {
	args := m.Called(ctx, messageID, fileName, fileType, storagePath, size)
	return args.Get(0).(int64), args.Error(1)
}

// Тесты

func TestSplitRcpts(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Single recipient",
			input:    "user@example.com",
			expected: []string{"user@example.com"},
		},
		{
			name:     "Multiple recipients",
			input:    "user1@example.com, user2@example.com, user3@example.com",
			expected: []string{"user1@example.com", "user2@example.com", "user3@example.com"},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "With spaces",
			input:    "  user1@example.com  ,  user2@example.com  ",
			expected: []string{"user1@example.com", "user2@example.com"},
		},
		{
			name:     "Multiple commas",
			input:    "user1@example.com,,user2@example.com",
			expected: []string{"user1@example.com", "user2@example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitRcpts(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitEmail(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedLocal  string
		expectedDomain string
	}{
		{
			name:           "Valid email",
			input:          "user@example.com",
			expectedLocal:  "user",
			expectedDomain: "example.com",
		},
		{
			name:           "Email with spaces",
			input:          "  user  @  example.com  ",
			expectedLocal:  "user",
			expectedDomain: "example.com",
		},
		{
			name:           "No domain",
			input:          "user",
			expectedLocal:  "user",
			expectedDomain: "",
		},
		{
			name:           "Multiple @ symbols",
			input:          "user@name@example.com",
			expectedLocal:  "user@name",
			expectedDomain: "example.com",
		},
		{
			name:           "Empty string",
			input:          "",
			expectedLocal:  "",
			expectedDomain: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			local, domain := splitEmail(tt.input)
			assert.Equal(t, tt.expectedLocal, local)
			assert.Equal(t, tt.expectedDomain, domain)
		})
	}
}

func TestDecodeHeader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Plain text",
			input:    "Test Subject",
			expected: "Test Subject",
		},
		{
			name:     "Base64 encoded",
			input:    "=?UTF-8?B?VGVzdCBTdWJqZWN0?=",
			expected: "Test Subject",
		},
		{
			name:     "Quoted printable",
			input:    "=?UTF-8?Q?Test_Subject?=",
			expected: "Test Subject",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "With spaces",
			input:    "  Test Subject  ",
			expected: "Test Subject",
		},
		{
			name:     "Invalid encoding",
			input:    "=?UTF-8?X?Invalid?=",
			expected: "=?UTF-8?X?Invalid?=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decodeHeader(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractBodyFromSingle(t *testing.T) {
	tests := []struct {
		name     string
		body     []byte
		cte      string
		expected string
	}{
		{
			name:     "Plain text",
			body:     []byte("Hello World"),
			cte:      "",
			expected: "Hello World",
		},
		{
			name:     "Base64 encoded",
			body:     []byte("SGVsbG8gV29ybGQ="),
			cte:      "base64",
			expected: "Hello World",
		},
		{
			name:     "Quoted printable",
			body:     []byte("Hello=20World"),
			cte:      "quoted-printable",
			expected: "Hello World",
		},
		{
			name:     "Invalid base64",
			body:     []byte("Not base64"),
			cte:      "base64",
			expected: "",
		},
		{
			name:     "Empty body",
			body:     []byte(""),
			cte:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBodyFromSingle(tt.body, tt.cte)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Valid filename",
			input:    "document.pdf",
			expected: "document.pdf",
		},
		{
			name:     "With path",
			input:    "/path/to/document.pdf",
			expected: "document.pdf",
		},
		{
			name:     "With backslashes",
			input:    "folder\\document.pdf",
			expected: "document.pdf",
		},
		{
			name:     "With slashes",
			input:    "folder/document.pdf",
			expected: "document.pdf",
		},
		{
			name:     "Empty filename",
			input:    "",
			expected: "file.bin",
		},
		{
			name:     "Only slashes",
			input:    "///",
			expected: "file.bin",
		},
		{
			name:     "With special characters",
			input:    "doc<ment>.pdf",
			expected: "doc<ment>.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFileName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseMime(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	tests := []struct {
		name         string
		ctHeader     string
		body         []byte
		expectedText string
		expectedAtts int
	}{
		{
			name:         "Plain text",
			ctHeader:     "text/plain; charset=utf-8",
			body:         []byte("Hello World"),
			expectedText: "Hello World",
			expectedAtts: 0,
		},
		{
			name:         "HTML only",
			ctHeader:     "text/html; charset=utf-8",
			body:         []byte("<html><body>Hello</body></html>"),
			expectedText: "<html><body>Hello</body></html>",
			expectedAtts: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, atts := parseMime(tt.ctHeader, tt.body, log)
			assert.Equal(t, tt.expectedText, strings.TrimSpace(text))
			assert.Len(t, atts, tt.expectedAtts)
		})
	}
}

func TestParseMime_Multipart(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	textHeader := make(map[string][]string)
	textHeader["Content-Type"] = []string{"text/plain; charset=utf-8"}
	textPart, _ := writer.CreatePart(textHeader)
	textPart.Write([]byte("Hello World"))

	htmlHeader := make(map[string][]string)
	htmlHeader["Content-Type"] = []string{"text/html; charset=utf-8"}
	htmlPart, _ := writer.CreatePart(htmlHeader)
	htmlPart.Write([]byte("<html><body>Hello</body></html>"))

	writer.Close()

	ctHeader := "multipart/alternative; boundary=" + writer.Boundary()
	text, atts := parseMime(ctHeader, buf.Bytes(), log)

	assert.Equal(t, "Hello World", strings.TrimSpace(text))
	assert.Len(t, atts, 0)
}

func TestParseMime_MultipartWithAttachment(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	textHeader := make(map[string][]string)
	textHeader["Content-Type"] = []string{"text/plain; charset=utf-8"}
	textPart, _ := writer.CreatePart(textHeader)
	textPart.Write([]byte("Hello World"))

	attachmentHeader := make(map[string][]string)
	attachmentHeader["Content-Type"] = []string{"application/pdf"}
	attachmentHeader["Content-Disposition"] = []string{`attachment; filename="test.pdf"`}
	attachmentPart, _ := writer.CreatePart(attachmentHeader)
	attachmentPart.Write([]byte("PDF content"))

	writer.Close()

	ctHeader := "multipart/mixed; boundary=" + writer.Boundary()
	text, atts := parseMime(ctHeader, buf.Bytes(), log)

	assert.Equal(t, "Hello World", strings.TrimSpace(text))
	assert.Len(t, atts, 1)
	assert.Equal(t, "test.pdf", atts[0].Name)
	assert.Equal(t, "application/pdf", atts[0].ContentType)
}

func TestWalkMime(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	textHeader := make(map[string][]string)
	textHeader["Content-Type"] = []string{"text/plain; charset=utf-8"}
	textPart, _ := writer.CreatePart(textHeader)
	textPart.Write([]byte("Plain text"))

	htmlHeader := make(map[string][]string)
	htmlHeader["Content-Type"] = []string{"text/html; charset=utf-8"}
	htmlPart, _ := writer.CreatePart(htmlHeader)
	htmlPart.Write([]byte("<html><body>HTML</body></html>"))

	writer.Close()

	ctHeader := "multipart/alternative; boundary=" + writer.Boundary()
	plain, html, atts := walkMime(ctHeader, buf.Bytes(), log)

	assert.Equal(t, "Plain text", strings.TrimSpace(plain))
	assert.Equal(t, "<html><body>HTML</body></html>", strings.TrimSpace(html))
	assert.Len(t, atts, 0)
}

func TestReadPartBytes(t *testing.T) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	header := make(map[string][]string)
	header["Content-Type"] = []string{"text/plain; charset=utf-8"}
	part, _ := writer.CreatePart(header)
	part.Write([]byte("Hello World"))

	writer.Close()

	reader := multipart.NewReader(&buf, writer.Boundary())
	readPart, _ := reader.NextPart()

	data, err := readPartBytes(readPart)
	assert.NoError(t, err)
	assert.Equal(t, []byte("Hello World"), data)
}

func TestReadPartBytes_Base64(t *testing.T) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	header := make(map[string][]string)
	header["Content-Type"] = []string{"text/plain; charset=utf-8"}
	header["Content-Transfer-Encoding"] = []string{"base64"}
	part, _ := writer.CreatePart(header)
	part.Write([]byte("SGVsbG8gV29ybGQ="))

	writer.Close()

	reader := multipart.NewReader(&buf, writer.Boundary())
	readPart, _ := reader.NextPart()

	data, err := readPartBytes(readPart)
	assert.NoError(t, err)
	assert.Equal(t, []byte("Hello World"), data)
}

func TestEnvOr(t *testing.T) {
	t.Setenv("TEST_KEY", "test-value")
	t.Setenv("EMPTY_KEY", "")

	assert.Equal(t, "test-value", envOr("TEST_KEY", "default"))
	assert.Equal(t, "default", envOr("EMPTY_KEY", "default"))
	assert.Equal(t, "default", envOr("NONEXISTENT_KEY", "default"))
	assert.Equal(t, "default", envOr("", "default"))
}

func BenchmarkSplitRcpts(b *testing.B) {
	input := "user1@example.com, user2@example.com, user3@example.com, user4@example.com, user5@example.com"
	for i := 0; i < b.N; i++ {
		splitRcpts(input)
	}
}

func BenchmarkDecodeHeader(b *testing.B) {
	header := "=?UTF-8?B?VGVzdCBTdWJqZWN0IHdpdGggZW5jb2Rpbmc=?="
	for i := 0; i < b.N; i++ {
		decodeHeader(header)
	}
}

func BenchmarkExtractBodyFromSingle(b *testing.B) {
	body := []byte("SGVsbG8gV29ybGQgdGhpcyBpcyBhIHRlc3QgbWVzc2FnZSB3aXRoIGJhc2U2NCBlbmNvZGluZw==")
	cte := "base64"
	for i := 0; i < b.N; i++ {
		extractBodyFromSingle(body, cte)
	}
}

func BenchmarkSanitizeFileName(b *testing.B) {
	filename := "/path/to/../../document with spaces.pdf"
	for i := 0; i < b.N; i++ {
		sanitizeFileName(filename)
	}
}

// Вспомогательная функция для тестирования parseMime без внешних зависимостей
func TestParseMime_HelperFunctions(t *testing.T) {

	// Тестируем decodeSingleBody
	body := "Hello World"
	cte := ""
	result := decodeSingleBody(strings.NewReader(body), cte)
	assert.Equal(t, body, result)

	// Тестируем decodePartBody
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	header := make(map[string][]string)
	header["Content-Type"] = []string{"text/plain; charset=utf-8"}
	part, _ := writer.CreatePart(header)
	part.Write([]byte(body))
	writer.Close()

	reader := multipart.NewReader(&buf, writer.Boundary())
	readPart, _ := reader.NextPart()
	result2 := decodePartBody(readPart)
	assert.Equal(t, body, result2)
}
