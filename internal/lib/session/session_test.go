package session

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var testSecret = []byte("secret")
var wrongSecret = []byte("wrongSecret")

func generateToken(claims jwt.MapClaims, secret []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func createRequestWithCookie(cookieName, token string) *http.Request {
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  cookieName,
		Value: token,
	})
	return req
}

func createClaims(userID int64, tokenType string, expTime time.Time) jwt.MapClaims {
	claims := jwt.MapClaims{
		"user_id": float64(userID),
		"type":    tokenType,
	}
	if !expTime.IsZero() {
		claims["exp"] = float64(expTime.Unix())
	}
	return claims
}

func TestCheckSessionWithToken(t *testing.T) {
	const cookieName = "test_cookie"
	const expectedType = "access"
	testUserID := int64(123)

	validTime := time.Now().Add(time.Hour)

	validClaims := createClaims(testUserID, expectedType, validTime)
	wrongTypeClaims := createClaims(testUserID, "refresh", validTime)
	noTypeClaims := createClaims(testUserID, "", validTime)
	delete(noTypeClaims, "type")

	validToken, _ := generateToken(validClaims, testSecret)
	wrongTypeToken, _ := generateToken(wrongTypeClaims, testSecret)
	noTypeToken, _ := generateToken(noTypeClaims, testSecret)
	wrongSecretToken, _ := generateToken(validClaims, wrongSecret)

	type args struct {
		r            *http.Request
		SECRET       []byte
		cookieName   string
		expectedType string
	}
	tests := []struct {
		name    string
		args    args
		want    jwt.MapClaims
		wantErr error
	}{
		{
			name: "Success: Valid Access Token",
			args: args{
				r:            createRequestWithCookie(cookieName, validToken),
				SECRET:       testSecret,
				cookieName:   cookieName,
				expectedType: expectedType,
			},
			want:    validClaims,
			wantErr: nil,
		},
		{
			name: "Failure: No Cookie",
			args: args{
				r:            httptest.NewRequest("GET", "/", nil),
				SECRET:       testSecret,
				cookieName:   cookieName,
				expectedType: expectedType,
			},
			want:    jwt.MapClaims{},
			wantErr: ErrorSessionNotFound,
		},
		{
			name: "Failure: Wrong Token Type",
			args: args{
				r:            createRequestWithCookie(cookieName, wrongTypeToken),
				SECRET:       testSecret,
				cookieName:   cookieName,
				expectedType: expectedType,
			},
			want:    jwt.MapClaims{},
			wantErr: ErrorWrongTokenType,
		},
		{
			name: "Failure: Token Missing Type (when expected)",
			args: args{
				r:            createRequestWithCookie(cookieName, noTypeToken),
				SECRET:       testSecret,
				cookieName:   cookieName,
				expectedType: expectedType,
			},
			want:    jwt.MapClaims{},
			wantErr: ErrorWrongTokenType,
		},
		{
			name: "Failure: Invalid Signature/Wrong Secret",
			args: args{
				r:            createRequestWithCookie(cookieName, wrongSecretToken),
				SECRET:       testSecret,
				cookieName:   cookieName,
				expectedType: expectedType,
			},
			want:    jwt.MapClaims{},
			wantErr: ErrorInvalidToken,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckSessionWithToken(tt.args.r, tt.args.SECRET, tt.args.cookieName, tt.args.expectedType)

			if (err != nil && tt.wantErr == nil) || (err == nil && tt.wantErr != nil) || (err != nil && tt.wantErr != nil && err.Error() != tt.wantErr.Error()) {
				t.Errorf("CheckSessionWithToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want != nil && tt.want["exp"] != nil {
				tt.want["exp"] = float64(int64(tt.want["exp"].(float64)))
			}
			if got != nil && got["exp"] != nil {
				got["exp"] = float64(int64(got["exp"].(float64)))
			}

			delete(got, "iat")
			delete(tt.want, "iat")

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckSessionWithToken() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckSession(t *testing.T) {
	testUserID := int64(456)
	validTime := time.Now().Add(time.Hour)

	validClaims := createClaims(testUserID, "access", validTime)
	wrongTypeClaims := createClaims(testUserID, "refresh", validTime)

	validToken, _ := generateToken(validClaims, testSecret)
	wrongTypeToken, _ := generateToken(wrongTypeClaims, testSecret)

	type args struct {
		r      *http.Request
		SECRET []byte
	}
	tests := []struct {
		name    string
		args    args
		want    jwt.MapClaims
		wantErr error
	}{
		{
			name: "Success: Valid Access Token",
			args: args{
				r:      createRequestWithCookie("access_token", validToken),
				SECRET: testSecret,
			},
			want:    validClaims,
			wantErr: nil,
		},
		{
			name: "Failure: No access_token Cookie",
			args: args{
				r:      httptest.NewRequest("GET", "/", nil),
				SECRET: testSecret,
			},
			want:    jwt.MapClaims{},
			wantErr: ErrorSessionNotFound,
		},
		{
			name: "Failure: Wrong Token Type (Refresh token used as Access)",
			args: args{
				r:      createRequestWithCookie("access_token", wrongTypeToken),
				SECRET: testSecret,
			},
			want:    jwt.MapClaims{},
			wantErr: ErrorWrongTokenType,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckSession(tt.args.r, tt.args.SECRET)

			if (err != nil && tt.wantErr == nil) || (err == nil && tt.wantErr != nil) || (err != nil && tt.wantErr != nil && err.Error() != tt.wantErr.Error()) {
				t.Errorf("CheckSession() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want != nil && tt.want["exp"] != nil {
				tt.want["exp"] = float64(int64(tt.want["exp"].(float64)))
			}
			if got != nil && got["exp"] != nil {
				got["exp"] = float64(int64(got["exp"].(float64)))
			}
			delete(got, "iat")
			delete(tt.want, "iat")

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckSession() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckSessionWithRefreshToken(t *testing.T) {
	testUserID := int64(789)
	validTime := time.Now().Add(time.Hour)

	validClaims := createClaims(testUserID, "refresh", validTime)
	wrongTypeClaims := createClaims(testUserID, "access", validTime)

	validToken, _ := generateToken(validClaims, testSecret)
	wrongTypeToken, _ := generateToken(wrongTypeClaims, testSecret)

	type args struct {
		r      *http.Request
		SECRET []byte
	}
	tests := []struct {
		name    string
		args    args
		want    jwt.MapClaims
		wantErr error
	}{
		{
			name: "Success: Valid Refresh Token",
			args: args{
				r:      createRequestWithCookie("refresh_token", validToken),
				SECRET: testSecret,
			},
			want:    validClaims,
			wantErr: nil,
		},
		{
			name: "Failure: Wrong Token Type (Access token used as Refresh)",
			args: args{
				r:      createRequestWithCookie("refresh_token", wrongTypeToken),
				SECRET: testSecret,
			},
			want:    jwt.MapClaims{},
			wantErr: ErrorWrongTokenType,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckSessionWithRefreshToken(tt.args.r, tt.args.SECRET)

			// Check error
			if (err != nil && tt.wantErr == nil) || (err == nil && tt.wantErr != nil) || (err != nil && tt.wantErr != nil && err.Error() != tt.wantErr.Error()) {
				t.Errorf("CheckSessionWithRefreshToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want != nil && tt.want["exp"] != nil {
				tt.want["exp"] = float64(int64(tt.want["exp"].(float64)))
			}
			if got != nil && got["exp"] != nil {
				got["exp"] = float64(int64(got["exp"].(float64)))
			}
			delete(got, "iat")
			delete(tt.want, "iat")

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckSessionWithRefreshToken() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetProfileIDFromToken(t *testing.T) {
	const cookieName = "test_cookie"
	const expectedType = "access"
	testUserID := int64(123)

	validTime := time.Now().Add(time.Hour)

	validClaims := createClaims(testUserID, expectedType, validTime)
	validToken, _ := generateToken(validClaims, testSecret)

	missingIDClaims := createClaims(testUserID, expectedType, validTime)
	delete(missingIDClaims, "user_id")
	missingIDToken, _ := generateToken(missingIDClaims, testSecret)

	type args struct {
		r            *http.Request
		SECRET       []byte
		cookieName   string
		expectedType string
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantErr error
	}{
		{
			name: "Success: Valid Token and ID",
			args: args{
				r:            createRequestWithCookie(cookieName, validToken),
				SECRET:       testSecret,
				cookieName:   cookieName,
				expectedType: expectedType,
			},
			want:    testUserID,
			wantErr: nil,
		},
		{
			name: "Failure: ID is Missing",
			args: args{
				r:            createRequestWithCookie(cookieName, missingIDToken),
				SECRET:       testSecret,
				cookieName:   cookieName,
				expectedType: expectedType,
			},
			want:    -1,
			wantErr: ErrorIdNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetProfileIDFromToken(tt.args.r, tt.args.SECRET, tt.args.cookieName, tt.args.expectedType)

			if (err != nil && tt.wantErr == nil) || (err == nil && tt.wantErr != nil) || (err != nil && tt.wantErr != nil && err.Error() != tt.wantErr.Error()) {
				t.Errorf("GetProfileIDFromToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("GetProfileIDFromToken() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetProfileID(t *testing.T) {
	testUserID := int64(456)
	validTime := time.Now().Add(time.Hour)

	validClaims := createClaims(testUserID, "access", validTime)
	validToken, _ := generateToken(validClaims, testSecret)

	type args struct {
		r      *http.Request
		SECRET []byte
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantErr error
	}{
		{
			name: "Success: Valid Access Token and ID",
			args: args{
				r:      createRequestWithCookie("access_token", validToken),
				SECRET: testSecret,
			},
			want:    testUserID,
			wantErr: nil,
		},
		{
			name: "Failure: No access_token Cookie",
			args: args{
				r:      httptest.NewRequest("GET", "/", nil),
				SECRET: testSecret,
			},
			want:    -1,
			wantErr: ErrorSessionNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetProfileID(tt.args.r, tt.args.SECRET)

			if (err != nil && tt.wantErr == nil) || (err == nil && tt.wantErr != nil) || (err != nil && tt.wantErr != nil && err.Error() != tt.wantErr.Error()) {
				t.Errorf("GetProfileID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("GetProfileID() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetProfileIDFromRefresh(t *testing.T) {
	testUserID := int64(789)
	validTime := time.Now().Add(time.Hour)

	validClaims := createClaims(testUserID, "refresh", validTime)
	validToken, _ := generateToken(validClaims, testSecret)

	type args struct {
		r      *http.Request
		SECRET []byte
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantErr error
	}{
		{
			name: "Success: Valid Refresh Token and ID",
			args: args{
				r:      createRequestWithCookie("refresh_token", validToken),
				SECRET: testSecret,
			},
			want:    testUserID,
			wantErr: nil,
		},
		{
			name: "Failure: No refresh_token Cookie",
			args: args{
				r:      httptest.NewRequest("GET", "/", nil),
				SECRET: testSecret,
			},
			want:    -1,
			wantErr: ErrorSessionNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetProfileIDFromRefresh(tt.args.r, tt.args.SECRET)

			if (err != nil && tt.wantErr == nil) || (err == nil && tt.wantErr != nil) || (err != nil && tt.wantErr != nil && err.Error() != tt.wantErr.Error()) {
				t.Errorf("GetProfileIDFromRefresh() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("GetProfileIDFromRefresh() got = %v, want %v", got, tt.want)
			}
		})
	}
}
