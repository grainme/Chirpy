package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCheckPasswordHash(t *testing.T) {
	// First, we need to create some hashed passwords for testing
	password1 := "correctPassword123!"
	password2 := "anotherPassword456!"
	hash1, _ := HashPassword(password1)
	hash2, _ := HashPassword(password2)

	tests := []struct {
		name          string
		password      string
		hash          string
		wantErr       bool
		matchPassword bool
	}{
		{
			name:          "Correct password",
			password:      password1,
			hash:          hash1,
			wantErr:       false,
			matchPassword: true,
		},
		{
			name:          "Incorrect password",
			password:      "wrongPassword",
			hash:          hash1,
			wantErr:       false,
			matchPassword: false,
		},
		{
			name:          "Password doesn't match different hash",
			password:      password1,
			hash:          hash2,
			wantErr:       false,
			matchPassword: false,
		},
		{
			name:          "Empty password",
			password:      "",
			hash:          hash1,
			wantErr:       false,
			matchPassword: false,
		},
		{
			name:          "Invalid hash",
			password:      password1,
			hash:          "invalidhash",
			wantErr:       true,
			matchPassword: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, err := CheckPasswordHash(tt.password, tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckPasswordHash() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && match != tt.matchPassword {
				t.Errorf("CheckPasswordHash() expects %v, got %v", tt.matchPassword, match)
			}
		})
	}
}

func TestCreateAndValidateJWT(t *testing.T) {
	expiration, _ := time.ParseDuration("5m")
	tests := []struct {
		name             string
		userID           uuid.UUID
		tokenSecret      string
		expiresIn        time.Duration
		wantErr          bool
		matchID          bool
		validationSecret string
	}{
		{
			name:             "happy path",
			userID:           uuid.New(),
			tokenSecret:      "some-super-secret-string",
			expiresIn:        expiration,
			wantErr:          false,
			matchID:          true,
			validationSecret: "some-super-secret-string",
		},
		{
			name:             "Expired Token",
			userID:           uuid.New(),
			tokenSecret:      "some-super-secret-string",
			expiresIn:        -time.Second,
			wantErr:          true,
			matchID:          false,
			validationSecret: "some-super-secret-string",
		},
		{
			name:             "Wrong secret",
			userID:           uuid.New(),
			tokenSecret:      "some-super-secret-string",
			expiresIn:        expiration,
			wantErr:          true,
			matchID:          false,
			validationSecret: "secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// creating the token
			signedJWT, err := MakeJWT(tt.userID, tt.tokenSecret, tt.expiresIn)
			if !tt.wantErr && err != nil {
				t.Errorf("MakeJWT() error %v, wantErr %v", err, tt.wantErr)
				return
			}

			userID, err := ValidateJWT(signedJWT, tt.validationSecret)
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateJWT() error %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.matchID && userID != tt.userID {
				t.Errorf("ValidateJWT() expected %v, got %v", tt.userID, userID)
				return
			}
		})

	}
}

func TestBearerToken(t *testing.T) {
	headers := make(http.Header)
	headers.Set("Authorization", "Bearer some-secret-JWTtoken")

	tests := []struct {
		name    string
		wantErr bool
		want    string
	}{
		{
			name:    "happy path",
			wantErr: false,
			want:    "some-secret-JWTtoken",
		},
		{
			name:    "wrong secret",
			wantErr: true,
			want:    "BEARER some-secret-JWTtoken",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GetBearerToken(headers)
			if !tt.wantErr && err != nil {
				t.Errorf("GetBearerToken() error %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && token != tt.want {
				t.Errorf("GetBearerToken() expected %v, got %v", tt.want, token)
				return
			}
		})
	}
}
