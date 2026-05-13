// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package observability

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/adaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// IsMetricsEnabled reports whether Prometheus exposure is enabled.
// Default: enabled. Set METRICS_ENABLED=false to disable.
func IsMetricsEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("METRICS_ENABLED")))
	return v != "false" && v != "0" && v != "off"
}

// PromHandler returns the standard net/http promhttp handler, suitable for use
// outside of Hertz (e.g., a sidecar HTTP server on a different port).
func PromHandler() http.Handler {
	return promhttp.Handler()
}

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
