package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockHandler struct {
	called bool
}

func (h *mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.called = true
	w.WriteHeader(http.StatusOK)
}

func TestNew(t *testing.T) {
	middleware := New()
	allowedOrigin := "http://localhost:8080"
	disallowedOrigin := "http://malicious.com"

	tests := []struct {
		name            string
		method          string
		origin          string
		wantStatus      int
		wantCalled      bool
		wantAllowOrigin bool
	}{
		{
			name:            "OPTIONS: Allowed Origin (Preflight)",
			method:          "OPTIONS",
			origin:          allowedOrigin,
			wantStatus:      http.StatusOK,
			wantCalled:      false,
			wantAllowOrigin: true,
		},
		{
			name:            "OPTIONS: Disallowed Origin (Preflight)",
			method:          "OPTIONS",
			origin:          disallowedOrigin,
			wantStatus:      http.StatusOK,
			wantCalled:      false,
			wantAllowOrigin: false,
		},
		{
			name:            "GET: Allowed Origin (Actual Request)",
			method:          "GET",
			origin:          allowedOrigin,
			wantStatus:      http.StatusOK,
			wantCalled:      true,
			wantAllowOrigin: true,
		},
		{
			name:            "GET: Disallowed Origin (Actual Request)",
			method:          "GET",
			origin:          disallowedOrigin,
			wantStatus:      http.StatusOK,
			wantCalled:      true,
			wantAllowOrigin: false,
		},
		{
			name:            "GET: No Origin Header",
			method:          "GET",
			origin:          "",
			wantStatus:      http.StatusOK,
			wantCalled:      true,
			wantAllowOrigin: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockNext := &mockHandler{}
			handlerToTest := middleware(mockNext)

			req := httptest.NewRequest(tt.method, "http://example.com/foo", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			w := httptest.NewRecorder()
			handlerToTest.ServeHTTP(w, req)

			if mockNext.called != tt.wantCalled {
				t.Errorf("Next handler called mismatch. Got: %v, Want: %v", mockNext.called, tt.wantCalled)
			}

			if tt.method == "OPTIONS" && w.Code != tt.wantStatus {
				t.Errorf("Handler returned wrong status code for OPTIONS. Got: %v, Want: %v", w.Code, tt.wantStatus)
			}

			if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
				t.Errorf("Missing or wrong Access-Control-Allow-Credentials header. Got: %q", got)
			}
			if got := w.Header().Get("Access-Control-Allow-Methods"); got != "POST, GET, OPTIONS, PUT, DELETE" {
				t.Errorf("Missing or wrong Access-Control-Allow-Methods header. Got: %q", got)
			}
			if got := w.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type, Authorization, Accept" {
				t.Errorf("Missing or wrong Access-Control-Allow-Headers header. Got: %q", got)
			}

			if tt.wantAllowOrigin {
				if got := w.Header().Get("Access-Control-Allow-Origin"); got != tt.origin {
					t.Errorf("Expected ACAO header to be %q, got %q", tt.origin, got)
				}
			} else {
				if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
					t.Errorf("Expected ACAO header to be empty, got %q", got)
				}
			}
		})
	}
}

func Test_allowOrigin(t *testing.T) {
	type args struct {
		origin string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "Allowed: localhost:8080", args: args{origin: "http://localhost:8080"}, want: true},
		{name: "Allowed: 127.0.0.1:8080", args: args{origin: "http://127.0.0.1:8080"}, want: true},
		{name: "Allowed: specific IP:80", args: args{origin: "http://217.16.16.26:80"}, want: true},

		{name: "Disallowed: different port", args: args{origin: "http://localhost:3000"}, want: false},
		{name: "Disallowed: different IP", args: args{origin: "http://10.0.0.1:8080"}, want: false},
		{name: "Disallowed: HTTPS scheme", args: args{origin: "https://localhost:8080"}, want: false},
		{name: "Disallowed: Missing Scheme", args: args{origin: "localhost:8080"}, want: false},
		{name: "Disallowed: Malicious Domain", args: args{origin: "http://attacker.com"}, want: false},
		{name: "Disallowed: Empty string", args: args{origin: ""}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := allowOrigin(tt.args.origin); got != tt.want {
				t.Errorf("allowOrigin() = %v, want %v", got, tt.want)
			}
		})
	}
}
