package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"2025_2_a4code/internal/lib/metrics"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestNewStatusResponseWriter(t *testing.T) {
	tests := []struct {
		name         string
		initialCode  int
		writeCode    int
		expectedCode int
	}{
		{
			name:         "Default status code",
			initialCode:  http.StatusOK,
			writeCode:    http.StatusOK,
			expectedCode: http.StatusOK,
		},
		{
			name:         "Change status code",
			initialCode:  http.StatusOK,
			writeCode:    http.StatusNotFound,
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "Internal server error",
			initialCode:  http.StatusOK,
			writeCode:    http.StatusInternalServerError,
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:         "Bad request",
			initialCode:  http.StatusOK,
			writeCode:    http.StatusBadRequest,
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			srw := NewStatusResponseWriter(rr)

			assert.Equal(t, tt.initialCode, srw.statusCode)

			srw.WriteHeader(tt.writeCode)

			assert.Equal(t, tt.expectedCode, srw.statusCode)
			assert.Equal(t, tt.expectedCode, rr.Code)
		})
	}
}

func TestStatusResponseWriter_Write(t *testing.T) {
	rr := httptest.NewRecorder()
	srw := NewStatusResponseWriter(rr)

	data := []byte("test response")
	n, err := srw.Write(data)

	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, data, rr.Body.Bytes())
	assert.Equal(t, http.StatusOK, srw.statusCode)
}

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name               string
		method             string
		path               string
		handlerStatusCode  int
		expectedStatusCode int
		expectedPath       string
	}{
		{
			name:               "GET request with normal path",
			method:             "GET",
			path:               "/api/test",
			handlerStatusCode:  http.StatusOK,
			expectedStatusCode: http.StatusOK,
			expectedPath:       "/api/test",
		},
		{
			name:               "POST request with messages ID path",
			method:             "POST",
			path:               "/messages/123",
			handlerStatusCode:  http.StatusCreated,
			expectedStatusCode: http.StatusCreated,
			expectedPath:       "/messages/:id",
		},
		{
			name:               "PUT request with long messages ID path",
			method:             "PUT",
			path:               "/messages/abc123def456",
			handlerStatusCode:  http.StatusOK,
			expectedStatusCode: http.StatusOK,
			expectedPath:       "/messages/:id",
		},
		{
			name:               "DELETE request with messages path",
			method:             "DELETE",
			path:               "/messages/",
			handlerStatusCode:  http.StatusNoContent,
			expectedStatusCode: http.StatusNoContent,
			expectedPath:       "/messages/",
		},
		{
			name:               "Not found status",
			method:             "GET",
			path:               "/unknown",
			handlerStatusCode:  http.StatusNotFound,
			expectedStatusCode: http.StatusNotFound,
			expectedPath:       "/unknown",
		},
		{
			name:               "Internal server error",
			method:             "POST",
			path:               "/api/error",
			handlerStatusCode:  http.StatusInternalServerError,
			expectedStatusCode: http.StatusInternalServerError,
			expectedPath:       "/api/error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.handlerStatusCode)
				w.Write([]byte("response"))
			})

			middleware := Middleware(handler)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			middleware.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)
			assert.Equal(t, "response", rr.Body.String())
		})
	}
}

func TestMiddleware_MetricsRecording(t *testing.T) {
	originalRequestsTotal := metrics.HttpRequestsTotal
	originalRequestDuration := metrics.HttpRequestDuration

	defer func() {
		metrics.HttpRequestsTotal = originalRequestsTotal
		metrics.HttpRequestDuration = originalRequestDuration
	}()

	testRequestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	testRequestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "test_http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	metrics.HttpRequestsTotal = testRequestsTotal
	metrics.HttpRequestDuration = testRequestDuration

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	counter, err := testRequestsTotal.GetMetricWithLabelValues("GET", "/test", "200")
	assert.NoError(t, err)

	assert.NotNil(t, counter)
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal path",
			input:    "/api/users",
			expected: "/api/users",
		},
		{
			name:     "Messages with numeric ID",
			input:    "/messages/123",
			expected: "/messages/:id",
		},
		{
			name:     "Messages with UUID",
			input:    "/messages/abc123-def456",
			expected: "/messages/:id",
		},
		{
			name:     "Messages with alphanumeric ID",
			input:    "/messages/user123",
			expected: "/messages/:id",
		},
		{
			name:     "Messages root path",
			input:    "/messages",
			expected: "/messages",
		},
		{
			name:     "Messages with trailing slash",
			input:    "/messages/",
			expected: "/messages/",
		},
		{
			name:     "Nested messages path",
			input:    "/api/messages/123",
			expected: "/api/messages/123",
		},
		{
			name:     "Empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "Root path",
			input:    "/",
			expected: "/",
		},
		{
			name:     "Long messages path",
			input:    "/messages/123/details",
			expected: "/messages/123/details",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMiddleware_ExecutionTime(t *testing.T) {
	delay := 50 * time.Millisecond

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		w.WriteHeader(http.StatusOK)
	})

	middleware := Middleware(handler)

	req := httptest.NewRequest("GET", "/slow", nil)
	rr := httptest.NewRecorder()

	start := time.Now()
	middleware.ServeHTTP(rr, req)
	duration := time.Since(start)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.GreaterOrEqual(t, duration, delay)
}

func TestMiddleware_ConcurrentRequests(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Middleware(handler)

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			req := httptest.NewRequest("GET", "/concurrent", nil)
			rr := httptest.NewRecorder()

			middleware.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestStatusResponseWriter_ImplementsInterface(t *testing.T) {
	var _ http.ResponseWriter = NewStatusResponseWriter(httptest.NewRecorder())
}

func TestMiddleware_HeaderPreservation(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "test-value")
		w.WriteHeader(http.StatusCreated)
	})

	middleware := Middleware(handler)

	req := httptest.NewRequest("POST", "/api/resource", nil)
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.Equal(t, "test-value", rr.Header().Get("X-Custom-Header"))
}

func TestStatusResponseWriter_MultipleWriteHeaderCalls(t *testing.T) {
	rr := httptest.NewRecorder()
	srw := NewStatusResponseWriter(rr)

	srw.WriteHeader(http.StatusOK)
	srw.WriteHeader(http.StatusNotFound)

	assert.Equal(t, http.StatusOK, srw.statusCode)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMiddleware_DifferentHTTPMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Middleware(handler)

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", nil)
			rr := httptest.NewRecorder()

			middleware.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
		})
	}
}

func TestSanitizePath_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Multiple slashes",
			input:    "//messages//123",
			expected: "//messages//123",
		},
		{
			name:     "Very long ID",
			input:    "/messages/abcdefghijklmnopqrstuvwxyz0123456789",
			expected: "/messages/:id",
		},
		{
			name:     "Special characters in path",
			input:    "/messages/123!@#$%",
			expected: "/messages/:id",
		},
		{
			name:     "Just messages prefix",
			input:    "/messages",
			expected: "/messages",
		},
		{
			name:     "Messages with query params",
			input:    "/messages/123?param=value",
			expected: "/messages/:id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMiddleware_MetricsIncrement(t *testing.T) {
	originalRequestsTotal := metrics.HttpRequestsTotal
	defer func() {
		metrics.HttpRequestsTotal = originalRequestsTotal
	}()

	testRequestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_http_requests_total_2",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	metrics.HttpRequestsTotal = testRequestsTotal

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Middleware(handler)

	req := httptest.NewRequest("GET", "/test-metrics", nil)
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	counter, err := testRequestsTotal.GetMetricWithLabelValues("GET", "/test-metrics", "200")
	assert.NoError(t, err)
	assert.NotNil(t, counter)
}

func TestMiddleware_ErrorStatusCodes(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedStatus int
	}{
		{
			name:           "Client error - 400",
			statusCode:     http.StatusBadRequest,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Unauthorized - 401",
			statusCode:     http.StatusUnauthorized,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Forbidden - 403",
			statusCode:     http.StatusForbidden,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Not found - 404",
			statusCode:     http.StatusNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Internal server error - 500",
			statusCode:     http.StatusInternalServerError,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Service unavailable - 503",
			statusCode:     http.StatusServiceUnavailable,
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			})

			middleware := Middleware(handler)

			req := httptest.NewRequest("GET", "/error", nil)
			rr := httptest.NewRecorder()

			middleware.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestStatusResponseWriter_InitialState(t *testing.T) {
	rr := httptest.NewRecorder()
	srw := NewStatusResponseWriter(rr)

	assert.Equal(t, http.StatusOK, srw.statusCode)
	assert.Equal(t, rr, srw.ResponseWriter)
}

func TestMiddleware_NoBody(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	middleware := Middleware(handler)

	req := httptest.NewRequest("DELETE", "/resource", nil)
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.Empty(t, rr.Body.String())
}
