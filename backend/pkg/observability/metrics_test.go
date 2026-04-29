// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package observability

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestHTTPRequestsTotalIncrement(t *testing.T) {
	HTTPRequestsTotal.WithLabelValues("GET", "/foo", "200").Inc()
	got := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/foo", "200"))
	if got != 1 {
		t.Fatalf("expected 1, got %v", got)
	}
}

func TestLoopTaskTotalIncrement(t *testing.T) {
	LoopTaskTotal.WithLabelValues("success").Inc()
	got := testutil.ToFloat64(LoopTaskTotal.WithLabelValues("success"))
	if got != 1 {
		t.Fatalf("expected 1, got %v", got)
	}
}
