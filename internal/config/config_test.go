package config

import (
	"errors"
	"os"
	"reflect"
	"testing"
)

func TestGetConfig(t *testing.T) {
	originalConfigPath := os.Getenv("CONFIG_PATH")

	validConfig := Config{
		AppConfig: &AppConfig{
			Host:   "127.0.0.1",
			Port:   "8080",
			Secret: "test-secret",
		},
		DBConfig: &DBConfig{
			Host:     "db_host",
			Port:     "5432",
			User:     "db_user",
			Password: "db_password",
			Name:     "test_db",
			SSLMode:  "disable",
		},
		MinioConfig: &MinioConfig{
			Host:       "minio_host",
			Port:       "9000",
			User:       "minio_user",
			Password:   "minio_password",
			BucketName: "test-bucket",
			Endpoint:   "http://minio_endpoint",
			UseSSL:     false,
		},
	}

	tests := []struct {
		name         string
		configPath   string
		want         Config
		wantErr      bool
		errCheckFunc func(error) bool
	}{
		{
			name:       "Success: Load valid config file",
			configPath: "test_valid.yml",
			want:       validConfig,
			wantErr:    false,
		},
		{
			name:         "Failure: Config file not found",
			configPath:   "non_existent_config.yml",
			want:         Config{},
			wantErr:      true,
			errCheckFunc: func(err error) bool { return errors.Is(err, os.ErrNotExist) },
		},
		{
			name:         "Failure: Invalid YAML format",
			configPath:   "test_invalid.yml",
			want:         Config{},
			wantErr:      true,
			errCheckFunc: func(err error) bool { return errors.Is(err, errors.New("Can not parse config file")) },
		},
		{
			name:         "Failure: Default path not found",
			configPath:   "",
			want:         Config{},
			wantErr:      true,
			errCheckFunc: func(err error) bool { return errors.Is(err, os.ErrNotExist) },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.configPath == "" {
				os.Unsetenv("CONFIG_PATH")
			} else {
				os.Setenv("CONFIG_PATH", tt.configPath)
			}

			got, err := GetConfig()

			if (err != nil) != tt.wantErr {
				t.Fatalf("GetConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.errCheckFunc != nil {
				if !tt.errCheckFunc(err) {
					t.Errorf("GetConfig() returned wrong error: got %v", err)
				}
			}

			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetConfig() got = %+v, want %+v", got, tt.want)
			}
		})
	}

	if originalConfigPath == "" {
		os.Unsetenv("CONFIG_PATH")
	} else {
		os.Setenv("CONFIG_PATH", originalConfigPath)
	}
}
