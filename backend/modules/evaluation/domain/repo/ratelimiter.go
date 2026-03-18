// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package repo

import "context"

//go:generate mockgen -destination mocks/ratelimiter_mock.go -package mocks . RateLimiter
type RateLimiter interface {
	AllowInvoke(ctx context.Context, spaceID int64) bool
}
