// THIS CODE HAS A LOT OF COMMENTS BECAUSE IT'S EDUCATIONAL.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// HashPassword creates a secure hash of a plaintext password using Argon2id.
// This is a one-way operation - you can't reverse the hash back to the password.
// Use this when storing passwords in the database.
func HashPassword(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return password, err
	}
	return hash, nil
}

// CheckPasswordHash compares a plaintext password against a stored hash.
// Returns true if they match, false otherwise.
// Use this during login to verify the user's password.
func CheckPasswordHash(password, hash string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, err
	}
	return match, nil
}

// MakeJWT creates a signed JSON Web Token containing the user's ID.
//
// JWT Flow: After login succeeds, create a JWT and send it to the client.
// The client stores it and includes it in subsequent requests.
//
// Parameters:
//   - userID: The user's unique identifier (stored in the "Subject" claim)
//   - tokenSecret: Server's secret key - keep this safe! Anyone with this can forge tokens
//   - expiresIn: How long the token is valid (e.g., 1 hour, 24 hours)
//
// How it works:
//  1. Creates claims (data payload): who issued it, when, when it expires, user ID
//  2. Signs the token with HS256 (HMAC-SHA256) using the secret key
//  3. Returns a string like "eyJhbGc...header.payload.signature"
//
// The signature ensures the token can't be tampered with - if someone changes
// the userID, the signature won't match and validation will fail.
func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	// Create token with claims (the data inside the JWT)
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256, // Symmetric signing: same key signs and validates
		jwt.RegisteredClaims{
			Issuer:    "chirpy",                                      // Who created this token
			IssuedAt:  jwt.NewNumericDate(time.Now()),                // When it was created
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)), // When it expires
			Subject:   userID.String(),                               // Who this token is for (the user)
		},
	)

	// Sign the token with our secret key (converts to []byte for HS256)
	// This creates the signature that proves authenticity
	signedJWT, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}
	return signedJWT, nil
}

// ValidateJWT verifies a JWT's signature and extracts the user ID from it.
//
// Use this on every authenticated request to identify who's making the request.
//
// What it checks:
//  1. Signature is valid (token wasn't tampered with)
//  2. Token hasn't expired
//  3. Token structure is correct
//
// Returns the user's UUID if valid, or an error if:
//   - Signature doesn't match (wrong secret or tampered token)
//   - Token has expired
//   - Token is malformed
func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	// Parse the token string and validate its signature
	// The empty &jwt.RegisteredClaims{} will be filled with the token's data
	token, err := jwt.ParseWithClaims(
		tokenString,
		&jwt.RegisteredClaims{}, // Pointer so ParseWithClaims can fill it in
		func(t *jwt.Token) (any, error) {
			// This function is called during validation to get the secret key
			// For HS256: return []byte of the secret
			// For ES256/ES512: return *ecdsa.PublicKey
			return []byte(tokenSecret), nil
		},
	)
	// token is invalid (bad signature, expired, or malformed)
	if err != nil {
		return uuid.Nil, err
	}

	// Type assertion: convert the interface{} to *jwt.RegisteredClaims
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return uuid.Nil, errors.New("invalid token claims")
	}

	// Extract user ID from the Subject field (we put it there in MakeJWT)
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	header := strings.Fields(headers.Get("Authorization"))
	if len(header) < 2 {
		return "", errors.New("Bearer token not found!")
	}
	// header[0] should be "Bearer"
	// Authorization: Bearer ${jwtToken}
	bearerToken := header[1]
	return bearerToken, nil
}

func MakeRefreshToken() (string, error) {
	token := make([]byte, 32)
	_, err := rand.Read(token)
	if err != nil {
		return "", fmt.Errorf("Read from rand failed: %v", err)
	}
	return hex.EncodeToString(token), nil
}

func GetAPIKey(headers http.Header) (string, error) {
	header := headers.Get("Authorization")
	apiKey := strings.Fields(header)
	if len(apiKey) < 2 {
		return "", errors.New("Api key not found!")
	}
	return strings.TrimSpace(apiKey[1]), nil
}
