// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package observability

import (
	"context"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestHTTPMiddleware_IncrementsCounter(t *testing.T) {
	mw := HTTPRequestsMiddleware()
	c := app.NewContext(0)
	c.Request.Header.SetMethod(consts.MethodGet)
	c.Request.SetRequestURI("/api/test")
	c.Response.SetStatusCode(http.StatusOK)

	before := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/api/test", "200"))
	mw(context.Background(), c)
	after := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/api/test", "200"))

	if after != before+1 {
		t.Fatalf("expected counter +1, got before=%v after=%v", before, after)
	}
}

func TestNormalizePath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/api/users/123", "/api/users/:id"},
		{"/", "/"},
		{"", "/"},
		{"/api/users/00000000-0000-4000-8000-000000000000", "/api/users/:uuid"},
	}
	for _, c := range cases {
		if got := normalizePath(c.in); got != c.want {
			t.Errorf("normalizePath(%q)=%q, want %q", c.in, got, c.want)
		}
	}
}
