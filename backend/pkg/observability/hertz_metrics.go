// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package observability

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/adaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsHandler returns a Hertz handler that serves Prometheus metrics
// using promhttp.Handler() bridged through Hertz's net/http adaptor.
func MetricsHandler() app.HandlerFunc {
	httpHandler := promhttp.Handler()
	return func(ctx context.Context, c *app.RequestContext) {
		req, err := adaptor.GetCompatRequest(&c.Request)
		if err != nil {
			c.AbortWithStatus(500)
			return
		}
		w := adaptor.GetCompatResponseWriter(&c.Response)
		httpHandler.ServeHTTP(w, req)
	}
}
