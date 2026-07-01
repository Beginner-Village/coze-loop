package session

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	stdjson "encoding/json"
	"strings"
	"testing"
	"time"
)

func TestValidateSessionAcceptsStudioCookieWhenSecretsMatch(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	token := signTestSession(t, studioSession{
		ID:        7657251115750129664,
		CreatedAt: now.Add(-time.Minute),
		ExpiresAt: now.Add(time.Hour),
	}, studioHMACBytes)

	got, err := NewSessionService().ValidateSession(context.Background(), token)
	if err != nil {
		t.Fatalf("ValidateSession returned error: %v", err)
	}
	if got.UserID != "7657251115750129664" {
		t.Fatalf("UserID mismatch: got %q", got.UserID)
	}
}

func TestValidateSessionRejectsExpiredLoopCookie(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	token := signTestSession(t, Session{
		UserID:    "4",
		SessionID: 4,
		CreatedAt: now.Add(-2 * time.Hour),
		ExpiresAt: now.Add(-time.Hour),
	}, hmacSecret)

	_, err := NewSessionService().ValidateSession(context.Background(), token)
	if err == nil || !strings.Contains(err.Error(), "session expired") {
		t.Fatalf("expected session expired error, got %v", err)
	}
}

func signTestSession(t *testing.T, payload any, secret []byte) string {
	t.Helper()
	sessionData, err := stdjson.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	h := hmac.New(sha256.New, secret)
	h.Write(sessionData)
	return base64.RawURLEncoding.EncodeToString(append(sessionData, h.Sum(nil)...))
}
