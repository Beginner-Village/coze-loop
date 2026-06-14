// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	stdjson "encoding/json"
	"os"
	"strconv"
	"time"

	"github.com/coze-dev/coze-loop/backend/pkg/errorx"
	"github.com/coze-dev/coze-loop/backend/pkg/json"
)

const (
	SessionKey     = "session_key"
	SessionExpires = 7 * 24 * time.Hour

	envSessionHMACKey = "COZE_LOOP_SESSION_HMAC_KEY"
	envStudioHMACKey  = "COZE_LOOP_STUDIO_HMAC_KEY"
)

// Signing key: must be provided via `COZE_LOOP_SESSION_HMAC_KEY`.
var hmacSecret []byte

// studioHMACBytes is the HMAC secret shared with ynet-studio (backend/domain/user/service/user_impl.go).
// Loop accepts Studio-signed cookies so the Studio UI can embed Loop API
// (observability tab) without a separate Loop login.
// Must be provided via `COZE_LOOP_STUDIO_HMAC_KEY` and match Studio's SESSION_HMAC_SECRET.
var studioHMACBytes []byte

func init() {
	v := os.Getenv(envSessionHMACKey)
	if v == "" {
		panic("[session] 环境变量 COZE_LOOP_SESSION_HMAC_KEY 未配置或为空: 必须配置该变量作为 Loop 会话 HMAC 签名密钥, 服务拒绝以默认密钥启动")
	}
	hmacSecret = []byte(v)

	sv := os.Getenv(envStudioHMACKey)
	if sv == "" {
		panic("[session] 环境变量 COZE_LOOP_STUDIO_HMAC_KEY 未配置或为空: 必须配置该变量作为 Studio 会话 HMAC 签名密钥, 且取值必须与 ynet-studio 侧的 SESSION_HMAC_SECRET 完全一致, 否则 Studio 登录态无法在 Loop 验签")
	}
	studioHMACBytes = []byte(sv)
}

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
	// Do not log the raw session token (it is a replayable credential).

	// 解码会话数据
	data, err := base64.RawURLEncoding.DecodeString(sessionID)
	if err != nil {
		return nil, errorx.New("invalid session format: %w", err)
	}

	// 必须严格长于签名长度,保证 sessionData 非空(否则空 payload + 32 字节
	// 伪造签名也能进入 hmac 校验路径)。
	if len(data) <= 32 {
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
		// 拒绝非法/缺省用户 ID,避免映射到 user "0" 之类的占位身份。
		if ss.ID <= 0 {
			return nil, errorx.New("invalid studio session user id")
		}
		return &Session{
			UserID:    strconv.FormatInt(ss.ID, 10),
			SessionID: ss.ID,
			CreatedAt: ss.CreatedAt,
			ExpiresAt: ss.ExpiresAt,
		}, nil
	}

	return nil, errorx.New("invalid session signature")
}
