// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"net/http"
	"testing"
	"time"
)

func TestValidateStudioIdentityHeadersAcceptsValidSignedIdentity(t *testing.T) {
	t.Setenv("YNET_LOOP_INTERNAL_AUTH_SECRET", "shared-secret")
	now := time.Unix(1782864000, 0)
	headers := signedStudioIdentityHeaders(StudioIdentity{
		UserID:    "402087139",
		SpaceID:   "7652614054615187456",
		Email:     "402087139@qq.com",
		Name:      "Lu",
		Timestamp: now.Unix(),
	}, "shared-secret")

	got, present, err := ValidateStudioIdentityHeaders(headers.Get, now)
	if err != nil {
		t.Fatalf("expected valid identity, got error: %v", err)
	}
	if !present {
		t.Fatal("expected studio identity headers to be present")
	}
	if got.UserID != "402087139" || got.SpaceID != "7652614054615187456" {
		t.Fatalf("identity mismatch: got %+v", got)
	}
}

func TestValidateStudioIdentityHeadersRejectsInvalidSignature(t *testing.T) {
	t.Setenv("YNET_LOOP_INTERNAL_AUTH_SECRET", "shared-secret")
	now := time.Unix(1782864000, 0)
	headers := signedStudioIdentityHeaders(StudioIdentity{
		UserID:    "402087139",
		SpaceID:   "7652614054615187456",
		Timestamp: now.Unix(),
	}, "shared-secret")
	headers.Set(StudioHeaderSignature, "bad-signature")

	if _, present, err := ValidateStudioIdentityHeaders(headers.Get, now); !present || err == nil {
		t.Fatalf("expected invalid signature error, present=%v err=%v", present, err)
	}
}

func TestValidateStudioIdentityHeadersRejectsExpiredTimestamp(t *testing.T) {
	t.Setenv("YNET_LOOP_INTERNAL_AUTH_SECRET", "shared-secret")
	issuedAt := time.Unix(1782864000, 0)
	headers := signedStudioIdentityHeaders(StudioIdentity{
		UserID:    "402087139",
		SpaceID:   "7652614054615187456",
		Timestamp: issuedAt.Unix(),
	}, "shared-secret")

	if _, present, err := ValidateStudioIdentityHeaders(headers.Get, issuedAt.Add(10*time.Minute)); !present || err == nil {
		t.Fatalf("expected expired timestamp error, present=%v err=%v", present, err)
	}
}

func signedStudioIdentityHeaders(identity StudioIdentity, secret string) http.Header {
	headers := http.Header{}
	headers.Set(StudioHeaderUserID, identity.UserID)
	headers.Set(StudioHeaderSpaceID, identity.SpaceID)
	headers.Set(StudioHeaderEmail, identity.Email)
	headers.Set(StudioHeaderName, identity.Name)
	headers.Set(StudioHeaderTimestamp, formatStudioIdentityTimestamp(identity.Timestamp))
	headers.Set(StudioHeaderSignature, SignStudioIdentity(identity, secret))
	return headers
}
