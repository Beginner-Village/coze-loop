// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package observability

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

// HTTPRequestsMiddleware returns a Hertz middleware that records the
// canonical RED triplet (rate, errors, duration) for every request.
// The path label is normalized via normalizePath to keep cardinality
// bounded — numeric and UUID segments collapse to :id and :uuid.
func HTTPRequestsMiddleware() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		HTTPRequestsInFlight.Inc()
		defer HTTPRequestsInFlight.Dec()

		start := time.Now()
		c.Next(ctx)
		dur := time.Since(start).Seconds()

		method := string(c.Request.Method())
		path := normalizePath(string(c.Request.URI().Path()))
		status := strconv.Itoa(c.Response.StatusCode())

		HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		HTTPRequestDuration.WithLabelValues(method, path, status).Observe(dur)
	}
}

func normalizePath(p string) string {
	if p == "" {
		return "/"
	}
	parts := strings.Split(p, "/")
	for i, seg := range parts {
		if seg == "" {
			continue
		}
		if _, err := strconv.ParseInt(seg, 10, 64); err == nil {
			parts[i] = ":id"
			continue
		}
		if len(seg) == 36 && strings.Count(seg, "-") == 4 {
			parts[i] = ":uuid"
		}
	}
	return strings.Join(parts, "/")
}
