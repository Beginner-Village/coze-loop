// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/coze-dev/coze-loop/backend/infra/middleware/session"
	"github.com/coze-dev/coze-loop/backend/kitex_gen/coze/loop/foundation/user"
	"github.com/coze-dev/coze-loop/backend/kitex_gen/coze/loop/foundation/user/userservice"
	"github.com/coze-dev/coze-loop/backend/pkg/lang/ptr"
	"github.com/coze-dev/coze-loop/backend/pkg/logs"
)

func SessionMW(ss session.ISessionService, us userservice.Client) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		path := string(c.Path())
		if path == "/api/foundation/v1/users/login_by_password" ||
			path == "/api/foundation/v1/users/register" {
			c.Next(ctx)
			return
		}

		sess, err := ss.ValidateSession(ctx, string(c.Cookie(session.SessionKey)))
		if err != nil {
			_ = c.Error(err)
			c.Abort()
			return
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

		ctx = session.WithCtxUser(ctx, &session.User{
			ID:    sess.UserID,
			Name:  name,
			Email: email,
		})

		c.Next(ctx)
	}
}
