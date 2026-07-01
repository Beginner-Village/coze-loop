// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	StudioHeaderUserID    = "X-Ynet-Studio-User-Id"
	StudioHeaderSpaceID   = "X-Ynet-Studio-Space-Id"
	StudioHeaderEmail     = "X-Ynet-Studio-Email"
	StudioHeaderName      = "X-Ynet-Studio-Name"
	StudioHeaderTimestamp = "X-Ynet-Studio-Timestamp"
	StudioHeaderSignature = "X-Ynet-Studio-Signature"

	envStudioInternalAuthSecret = "YNET_LOOP_INTERNAL_AUTH_SECRET"
	envYnetLoopStudioHMACKey    = "YNET_LOOP_STUDIO_HMAC_KEY"

	studioIdentityMaxSkew = 5 * time.Minute
)

type StudioIdentity struct {
	UserID    string
	SpaceID   string
	Email     string
	Name      string
	Timestamp int64
}

func ValidateStudioIdentityHeaders(getHeader func(string) string, now time.Time) (*StudioIdentity, bool, error) {
	if getHeader == nil || !hasAnyStudioIdentityHeader(getHeader) {
		return nil, false, nil
	}

	identity := StudioIdentity{
		UserID:  strings.TrimSpace(getHeader(StudioHeaderUserID)),
		SpaceID: strings.TrimSpace(getHeader(StudioHeaderSpaceID)),
		Email:   strings.TrimSpace(getHeader(StudioHeaderEmail)),
		Name:    strings.TrimSpace(getHeader(StudioHeaderName)),
	}
	if identity.UserID == "" {
		return nil, true, fmt.Errorf("missing %s", StudioHeaderUserID)
	}
	if !isNumericStudioIdentityID(identity.UserID) {
		return nil, true, fmt.Errorf("invalid studio user id")
	}
	if identity.SpaceID != "" && !isNumericStudioIdentityID(identity.SpaceID) {
		return nil, true, fmt.Errorf("invalid studio space id")
	}

	timestamp, err := strconv.ParseInt(strings.TrimSpace(getHeader(StudioHeaderTimestamp)), 10, 64)
	if err != nil || timestamp <= 0 {
		return nil, true, fmt.Errorf("invalid %s", StudioHeaderTimestamp)
	}
	identity.Timestamp = timestamp
	issuedAt := time.Unix(timestamp, 0)
	if now.IsZero() {
		now = time.Now()
	}
	if issuedAt.After(now.Add(studioIdentityMaxSkew)) || issuedAt.Before(now.Add(-studioIdentityMaxSkew)) {
		return nil, true, fmt.Errorf("studio identity timestamp outside allowed window")
	}

	secret := studioInternalAuthSecret()
	if secret == "" {
		return nil, true, fmt.Errorf("studio internal auth secret is empty")
	}
	want := SignStudioIdentity(identity, secret)
	got := strings.TrimSpace(getHeader(StudioHeaderSignature))
	if got == "" {
		return nil, true, fmt.Errorf("missing %s", StudioHeaderSignature)
	}
	wantBytes, err := hex.DecodeString(want)
	if err != nil {
		return nil, true, fmt.Errorf("invalid expected signature")
	}
	gotBytes, err := hex.DecodeString(got)
	if err != nil || !hmac.Equal(gotBytes, wantBytes) {
		return nil, true, fmt.Errorf("invalid studio identity signature")
	}

	return &identity, true, nil
}

func SignStudioIdentity(identity StudioIdentity, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(studioIdentityPayload(identity)))
	return hex.EncodeToString(mac.Sum(nil))
}

func formatStudioIdentityTimestamp(timestamp int64) string {
	return strconv.FormatInt(timestamp, 10)
}

func studioIdentityPayload(identity StudioIdentity) string {
	return strings.Join([]string{
		strings.TrimSpace(identity.UserID),
		strings.TrimSpace(identity.SpaceID),
		strings.TrimSpace(identity.Email),
		strings.TrimSpace(identity.Name),
		formatStudioIdentityTimestamp(identity.Timestamp),
	}, "\n")
}

func studioInternalAuthSecret() string {
	for _, key := range []string{envStudioInternalAuthSecret, envYnetLoopStudioHMACKey, envStudioHMACKey} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	if len(studioHMACBytes) > 0 {
		return string(studioHMACBytes)
	}
	return ""
}

func hasAnyStudioIdentityHeader(getHeader func(string) string) bool {
	for _, key := range []string{
		StudioHeaderUserID,
		StudioHeaderSpaceID,
		StudioHeaderEmail,
		StudioHeaderName,
		StudioHeaderTimestamp,
		StudioHeaderSignature,
	} {
		if strings.TrimSpace(getHeader(key)) != "" {
			return true
		}
	}
	return false
}

func isNumericStudioIdentityID(value string) bool {
	if value == "" {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
