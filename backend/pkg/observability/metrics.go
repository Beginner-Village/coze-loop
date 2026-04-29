// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

// Package observability registers Prometheus metrics for Loop:
//   - HTTP RED (requests, latency, in-flight)
//   - Loop task pipeline (counts and per-stage durations)
//   - Evaluator invocations
package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// 通用 RED
var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{Name: "http_requests_total", Help: "HTTP requests"},
		[]string{"method", "path", "status"},
	)
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP latency",
			Buckets: []float64{0.005, 0.01, 0.05, 0.1, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path", "status"},
	)
	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{Name: "http_requests_in_flight", Help: "In-flight"},
	)
)

// Loop 业务指标
var (
	LoopTaskTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "loop_task_total",
			Help: "Tasks counted by status: queued|running|success|failed",
		},
		[]string{"status"},
	)
	LoopTaskDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "loop_task_duration_seconds",
			Help:    "Task wall-clock by stage",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 12), // 0.1s → 400s
		},
		[]string{"stage"},
	)
	LoopEvaluatorInvocationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "loop_evaluator_invocation_total",
			Help: "Evaluator calls by name",
		},
		[]string{"evaluator_name"},
	)
)
