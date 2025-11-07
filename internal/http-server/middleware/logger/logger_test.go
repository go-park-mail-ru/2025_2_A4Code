package logger

import (
	"context"
	"log/slog"
	"testing"
)

func TestGetLogger(t *testing.T) {
	defaultLogger := slog.Default()

	customLogger, _ := newMockLogger()
	ctxWithLogger := context.WithValue(context.Background(), loggerKey, customLogger)

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want *slog.Logger
	}{
		{
			name: "Returns default logger when context has no logger",
			args: args{ctx: context.Background()},
			want: defaultLogger,
		},
		{
			name: "Returns custom logger when context has one",
			args: args{ctx: ctxWithLogger},
			want: customLogger,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetLogger(tt.args.ctx); got != tt.want {
				t.Errorf("GetLogger() got logger instance %p, want %p", got, tt.want)
			}
		})
	}
}

type mockHandler struct {
	attrs map[string]interface{}
}

func (h *mockHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *mockHandler) Handle(ctx context.Context, r slog.Record) error {
	r.Attrs(func(a slog.Attr) bool {
		h.attrs[a.Key] = a.Value.Any()
		return true
	})
	h.attrs["msg"] = r.Message
	h.attrs["level"] = r.Level.String()
	return nil
}

func (h *mockHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make(map[string]interface{}, len(h.attrs)+len(attrs))
	for k, v := range h.attrs {
		newAttrs[k] = v
	}
	for _, attr := range attrs {
		newAttrs[attr.Key] = attr.Value.Any()
	}
	return &mockHandler{attrs: newAttrs}
}

func (h *mockHandler) WithGroup(name string) slog.Handler {
	return h
}

func newMockLogger() (*slog.Logger, *mockHandler) {
	mockH := &mockHandler{attrs: make(map[string]interface{})}
	return slog.New(mockH), mockH
}
