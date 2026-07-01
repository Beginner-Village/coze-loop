// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/coze-dev/coze-loop/backend/infra/middleware/session"
	"github.com/coze-dev/coze-loop/backend/kitex_gen/coze/loop/foundation/user"
	"github.com/coze-dev/coze-loop/backend/kitex_gen/coze/loop/foundation/user/userservice"
	"github.com/coze-dev/coze-loop/backend/pkg/lang/ptr"
	"github.com/coze-dev/coze-loop/backend/pkg/logs"
)

func SessionMW(ss session.ISessionService, us userservice.Client) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if identity, present, err := session.ValidateStudioIdentityHeaders(func(name string) string {
			return string(c.Request.Header.Peek(name))
		}, time.Now()); present {
			if err != nil {
				logs.CtxWarn(ctx, "[session] invalid studio internal identity: %v", err)
				_ = c.Error(err)
				c.Abort()
				return
			}
			ctx = withCtxStudioIdentity(ctx, identity)
			c.Next(ctx)
			return
		}

		if workspaceID := extractWorkspaceIDFromReferer(string(c.Request.Header.Peek("Referer"))); workspaceID != "" {
			ctx = session.WithExternalWorkspaceID(ctx, workspaceID)
		}

		path := string(c.Path())
		if path == "/api/foundation/v1/users/login_by_password" ||
			path == "/api/foundation/v1/users/register" {
			c.Next(ctx)
			return
		}

		if path == "/api/foundation/v1/users/session" {
			if cookie := string(c.Cookie(session.SessionKey)); cookie != "" {
				sess, err := ss.ValidateSession(ctx, cookie)
				if err != nil {
					logs.CtxWarn(ctx, "[session] optional session validation failed on users/session: %v", err)
				} else {
					ctx = withCtxSessionUser(ctx, sess, us)
				}
			}
			c.Next(ctx)
			return
		}

		sess, err := ss.ValidateSession(ctx, string(c.Cookie(session.SessionKey)))
		if err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}

		ctx = withCtxSessionUser(ctx, sess, us)
		c.Next(ctx)
	}
}

func extractWorkspaceIDFromReferer(referer string) string {
	const marker = "/space/"
	idx := strings.Index(referer, marker)
	if idx < 0 {
		return ""
	}
	rest := referer[idx+len(marker):]
	end := strings.IndexByte(rest, '/')
	if end >= 0 {
		rest = rest[:end]
	}
	if query := strings.IndexByte(rest, '?'); query >= 0 {
		rest = rest[:query]
	}
	for _, ch := range rest {
		if ch < '0' || ch > '9' {
			return ""
		}
	}
	return rest
}

func withCtxSessionUser(ctx context.Context, sess *session.Session, us userservice.Client) context.Context {
	if sess == nil {
		return ctx
	}

	// 先查 Loop 自己的 user 表;若 user 不存在(例如来自 Studio session),
	// 信任 cookie 中的 UserID,继续放行,避免阻塞跨应用集成场景。
	var name, email string
	resp, err := us.GetUserInfo(ctx, &user.GetUserInfoRequest{
		UserID: ptr.Of(sess.UserID),
	})
	if err != nil || resp == nil || resp.GetUserInfo() == nil {
		logs.CtxWarn(ctx, "[session] user_id %s not found in loop, trusting external session (err=%v)", sess.UserID, err)
		name = "external-" + sess.UserID
		email = ""
	} else {
		name = resp.GetUserInfo().GetName()
		email = resp.GetUserInfo().GetEmail()
	}

	return session.WithCtxUser(ctx, &session.User{
		ID:         sess.UserID,
		Name:       name,
		Email:      email,
		IsExternal: strings.HasPrefix(name, "external-"),
	})
}

func withCtxStudioIdentity(ctx context.Context, identity *session.StudioIdentity) context.Context {
	if identity == nil {
		return ctx
	}
	if identity.SpaceID != "" {
		ctx = session.WithExternalWorkspaceID(ctx, identity.SpaceID)
	}
	name := strings.TrimSpace(identity.Name)
	if name == "" {
		name = "Studio User " + identity.UserID
	}
	return session.WithCtxUser(ctx, &session.User{
		ID:         identity.UserID,
		Name:       name,
		Email:      identity.Email,
		IsExternal: true,
	})
}
