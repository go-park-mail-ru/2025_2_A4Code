package response

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSendErrorResponse(t *testing.T) {
	expectedEmptyBodyValue := map[string]interface{}{}

	type args struct {
		errorMsg   string
		statusCode int
	}

	type expected struct {
		statusCode int
		body       Response
	}

	tests := []struct {
		name string
		args args
		want expected
	}{
		{
			name: "Test 400 Bad Request",
			args: args{
				errorMsg:   "Invalid request data.",
				statusCode: http.StatusBadRequest,
			},
			want: expected{
				statusCode: http.StatusBadRequest,
				body: Response{
					Status:  http.StatusBadRequest,
					Message: "Invalid request data.",
					Body:    expectedEmptyBodyValue,
				},
			},
		},
		{
			name: "Test 500 Internal Server Error",
			args: args{
				errorMsg:   "A server processing error occurred.",
				statusCode: http.StatusInternalServerError,
			},
			want: expected{
				statusCode: http.StatusInternalServerError,
				body: Response{
					Status:  http.StatusInternalServerError,
					Message: "A server processing error occurred.",
					Body:    expectedEmptyBodyValue,
				},
			},
		},
		{
			name: "Test 404 Not Found",
			args: args{
				errorMsg:   "The requested resource was not found.",
				statusCode: http.StatusNotFound, // 404
			},
			want: expected{
				statusCode: http.StatusNotFound,
				body: Response{
					Status:  http.StatusNotFound,
					Message: "The requested resource was not found.",
					Body:    expectedEmptyBodyValue,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			SendErrorResponse(rr, tt.args.errorMsg, tt.args.statusCode)

			if status := rr.Code; status != tt.want.statusCode {
				t.Errorf("SendErrorResponse returned wrong status code: got %v, want %v",
					status, tt.want.statusCode)
			}

			expectedContentType := "application/json"
			if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
				t.Errorf("SendErrorResponse did not set Content-Type header correctly: got %q, want %q",
					contentType, expectedContentType)
			}

			var actualBody Response

			bodyBytes, err := io.ReadAll(rr.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			if err := json.Unmarshal(bodyBytes, &actualBody); err != nil {
				t.Fatalf("Failed to unmarshal response body (JSON may be invalid): %v\nBody received: %s", err, bodyBytes)
			}

			if !reflect.DeepEqual(actualBody, tt.want.body) {
				expectedJSON, _ := json.MarshalIndent(tt.want.body, "", "  ")
				actualJSON, _ := json.MarshalIndent(actualBody, "", "  ")

				t.Errorf("SendErrorResponse returned unexpected body:\n--- GOT ---\n%s\n--- WANT ---\n%s",
					string(actualJSON), string(expectedJSON))
			}
		})
	}
}
