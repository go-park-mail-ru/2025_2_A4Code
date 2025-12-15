package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvOr(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		defValue string
		setEnv   bool
		envValue string
		expected string
	}{
		{
			name:     "Environment variable exists",
			key:      "TEST_KEY",
			defValue: "default",
			setEnv:   true,
			envValue: "env-value",
			expected: "env-value",
		},
		{
			name:     "Environment variable doesn't exist",
			key:      "NONEXISTENT_KEY",
			defValue: "default",
			setEnv:   false,
			expected: "default",
		},
		{
			name:     "Environment variable is empty",
			key:      "EMPTY_KEY",
			defValue: "default",
			setEnv:   true,
			envValue: "",
			expected: "default",
		},
		{
			name:     "Environment variable has spaces",
			key:      "SPACES_KEY",
			defValue: "default",
			setEnv:   true,
			envValue: "  value  ",
			expected: "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(tt.key, tt.envValue)
			}
			result := envOr(tt.key, tt.defValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractSubject(t *testing.T) {
	tests := []struct {
		name     string
		rawEmail string
		expected string
	}{
		{
			name: "Plain subject",
			rawEmail: `From: sender@example.com
To: receiver@example.com
Subject: Test Subject
Content-Type: text/plain

Hello World`,
			expected: "Test Subject",
		},
		{
			name: "Subject with encoding",
			rawEmail: `From: sender@example.com
To: receiver@example.com
Subject: =?UTF-8?B?VGVzdCBTdWJqZWN0?=
Content-Type: text/plain

Hello World`,
			expected: "=?UTF-8?B?VGVzdCBTdWJqZWN0?=",
		},
		{
			name: "No subject header",
			rawEmail: `From: sender@example.com
To: receiver@example.com
Content-Type: text/plain

Hello World`,
			expected: "",
		},
		{
			name: "Empty subject",
			rawEmail: `From: sender@example.com
To: receiver@example.com
Subject:
Content-Type: text/plain

Hello World`,
			expected: "",
		},
		{
			name:     "Invalid email format",
			rawEmail: `Invalid email content`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := []byte(tt.rawEmail)
			result := extractSubject(raw)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name           string
		envVars        map[string]string
		expectedConfig config
	}{
		{
			name:    "Default values",
			envVars: map[string]string{},
			expectedConfig: config{
				DBDSN:      "postgres://postgres:postgresql@mail-postgres:5432/maildb?sslmode=disable",
				LMTPAddr:   "0.0.0.0:2525",
				HTTPAddr:   ":8085",
				Domain:     "mail.local",
				BucketName: "mail-ingest",
				Minio: minioConfig{
					Endpoint:  "minio:9000",
					AccessKey: "minio",
					SecretKey: "miniominio",
					UseSSL:    false,
				},
			},
		},
		{
			name: "Custom values",
			envVars: map[string]string{
				"MAIL_DB_DSN":         "postgres://user:pass@localhost:5432/db",
				"MAIL_LMTP_ADDR":      ":2526",
				"MAIL_HTTP_ADDR":      ":8086",
				"MAIL_DOMAIN":         "example.com",
				"MAIL_MINIO_BUCKET":   "custom-bucket",
				"MAIL_MINIO_ENDPOINT": "localhost:9001",
				"MAIL_MINIO_USER":     "user",
				"MAIL_MINIO_PASSWORD": "password",
				"MAIL_MINIO_USE_SSL":  "true",
			},
			expectedConfig: config{
				DBDSN:      "postgres://user:pass@localhost:5432/db",
				LMTPAddr:   ":2526",
				HTTPAddr:   ":8086",
				Domain:     "example.com",
				BucketName: "custom-bucket",
				Minio: minioConfig{
					Endpoint:  "localhost:9001",
					AccessKey: "user",
					SecretKey: "password",
					UseSSL:    true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем новые значения
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			// Загружаем конфиг
			result := loadConfig()

			// Проверяем
			assert.Equal(t, tt.expectedConfig.DBDSN, result.DBDSN)
			assert.Equal(t, tt.expectedConfig.LMTPAddr, result.LMTPAddr)
			assert.Equal(t, tt.expectedConfig.HTTPAddr, result.HTTPAddr)
			assert.Equal(t, tt.expectedConfig.Domain, result.Domain)
			assert.Equal(t, tt.expectedConfig.BucketName, result.BucketName)
			assert.Equal(t, tt.expectedConfig.Minio.Endpoint, result.Minio.Endpoint)
			assert.Equal(t, tt.expectedConfig.Minio.AccessKey, result.Minio.AccessKey)
			assert.Equal(t, tt.expectedConfig.Minio.SecretKey, result.Minio.SecretKey)
			assert.Equal(t, tt.expectedConfig.Minio.UseSSL, result.Minio.UseSSL)
		})
	}
}

// Бенчмарк-тесты
func BenchmarkEnvOr(b *testing.B) {
	b.Setenv("BENCH_KEY", "bench-value")

	for i := 0; i < b.N; i++ {
		envOr("BENCH_KEY", "default")
	}
}

func BenchmarkExtractSubject(b *testing.B) {
	rawEmail := []byte(`From: sender@example.com
To: receiver@example.com
Subject: Benchmark Test Subject
Content-Type: text/plain

Hello World`)

	for i := 0; i < b.N; i++ {
		extractSubject(rawEmail)
	}
}

// Вспомогательные функции для тестирования
func TestSplitRcptsHelper(t *testing.T) {
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
			input:    "user1@example.com,user2@example.com",
			expected: []string{"user1@example.com", "user2@example.com"},
		},
		{
			name:     "With spaces",
			input:    "user1@example.com, user2@example.com",
			expected: []string{"user1@example.com", "user2@example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Это вспомогательная функция, которая не существует в основном коде
			// но может быть полезна для тестирования
			rcpts := strings.Split(strings.ReplaceAll(tt.input, " ", ""), ",")
			if tt.input == "" {
				rcpts = []string{}
			}
			assert.Equal(t, tt.expected, rcpts)
		})
	}
}
