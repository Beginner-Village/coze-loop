// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	stdjson "encoding/json"
	"strconv"
	"time"

	"github.com/coze-dev/coze-loop/backend/pkg/errorx"
	"github.com/coze-dev/coze-loop/backend/pkg/json"
	"github.com/coze-dev/coze-loop/backend/pkg/logs"
)

const (
	SessionKey     = "session_key"
	SessionExpires = 7 * 24 * time.Hour
)

// 用于签名的密钥（在实际应用中应从配置中读取或使用环境变量）
var hmacSecret = []byte("openloop-session-hmac-key")

// studioHMACSecret is the HMAC secret hardcoded in ynet-studio (backend/domain/user/service/user_impl.go).
// Loop accepts Studio-signed cookies so the Studio UI can embed Loop API
// (observability tab) without a separate Loop login.
var studioHMACBytes = []byte("openynet-session-hmac-key")

// studioSession mirrors ynet-studio's Session struct with snake_case JSON tags.
type studioSession struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Session struct {
	UserID string

	SessionID int64
	CreatedAt time.Time
	ExpiresAt time.Time
}

//go:generate mockgen -destination=mocks/session_service.go -package=mock_session . ISessionService
type ISessionService interface {
	ValidateSession(ctx context.Context, sessionID string) (*Session, error)
	GenerateSessionKey(ctx context.Context, session *Session) (string, error)
}

type sessionServiceImpl struct{}

func NewSessionService() ISessionService {
	return &sessionServiceImpl{}
}

func (s sessionServiceImpl) GenerateSessionKey(ctx context.Context, session *Session) (string, error) {
	// 设置会话的创建时间和过期时间
	session.CreatedAt = time.Now()
	session.ExpiresAt = time.Now().Add(SessionExpires)

	// 序列化会话数据
	sessionData, err := json.Marshal(session)
	if err != nil {
		return "", err
	}

	// 计算HMAC签名以确保完整性
	h := hmac.New(sha256.New, hmacSecret)
	h.Write(sessionData)
	signature := h.Sum(nil)

	// 组合会话数据和签名
	finalData := append(sessionData, signature...)

	// Base64编码最终结果
	return base64.RawURLEncoding.EncodeToString(finalData), nil
}

func (s sessionServiceImpl) ValidateSession(ctx context.Context, sessionID string) (*Session, error) {
	logs.CtxDebug(ctx, "sessionID: %s", sessionID)

	// 解码会话数据
	data, err := base64.RawURLEncoding.DecodeString(sessionID)
	if err != nil {
		return nil, errorx.New("invalid session format: %w, data:%s", err, sessionID)
	}

	// 确保数据长够长，至少包含会话数据和签名
	if len(data) < 32 { // 简单检查，实际应该更严格
		return nil, errorx.New("session data too short")
	}

	// 分离会话数据和签名
	sessionData := data[:len(data)-32] // 假设签名是32字节
	signature := data[len(data)-32:]

	// 验证签名 — 先用 Loop 自己的 HMAC,失败再 fallback 到 Studio HMAC
	h := hmac.New(sha256.New, hmacSecret)
	h.Write(sessionData)
	if hmac.Equal(signature, h.Sum(nil)) {
		var session Session
		if err := json.Unmarshal(sessionData, &session); err != nil {
			return nil, errorx.New("invalid session data: %w", err)
		}
		if time.Now().After(session.ExpiresAt) {
			return nil, errorx.New("session expired")
		}
		return &session, nil
	}

	// Fallback: 验 Studio 签名的 session
	hs := hmac.New(sha256.New, studioHMACBytes)
	hs.Write(sessionData)
	if hmac.Equal(signature, hs.Sum(nil)) {
		var ss studioSession
		if err := stdjson.Unmarshal(sessionData, &ss); err != nil {
			return nil, errorx.New("invalid studio session data: %w", err)
		}
		if time.Now().After(ss.ExpiresAt) {
			return nil, errorx.New("studio session expired")
		}
		logs.CtxDebug(ctx, "[session] accepted external Studio session, user=%d", ss.ID)
		return &Session{
			UserID:    strconv.FormatInt(ss.ID, 10),
			SessionID: ss.ID,
			CreatedAt: ss.CreatedAt,
			ExpiresAt: ss.ExpiresAt,
		}, nil
	}

	return nil, errorx.New("invalid session signature")
}
