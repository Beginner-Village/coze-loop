// Copyright (c) 2026 coze-dev Authors
// SPDX-License-Identifier: Apache-2.0

//go:build integration

// This file is gated behind the `integration` build tag so the default
// `go test ./...` never compiles it and never touches a real database. Run it
// explicitly against a real MySQL carrying the production workflow_meta
// snapshot:
//
//	WF_TEST_DSN='root:<pw>@tcp(10.10.10.226:13306)/ynet-loop?charset=utf8mb4&parseTime=true&loc=Local' \
//	  go test -tags integration -run TestWorkflowProviderRealDB \
//	  ./modules/observability/infra/workflow/ -v -count=1
//
// The test is strictly read-only (BatchGet -> SELECT ... WHERE id IN (...)).
package workflow

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/coze-dev/coze-loop/backend/infra/db"
	"github.com/coze-dev/coze-loop/backend/modules/observability/domain/trace/entity/loop_span"
)

// TestWorkflowProviderRealDB proves that the WorkflowProvider, wired to the real
// MySQL-backed WorkflowMetaDao, resolves a span's numeric sub_workflow_id (Int64
// in tags_long) into the human-readable workflow name stored in the production
// observability_workflow_meta snapshot table.
func TestWorkflowProviderRealDB(t *testing.T) {
	dsn := os.Getenv("WF_TEST_DSN")
	if dsn == "" {
		t.Skip("WF_TEST_DSN not set; skipping real-DB integration test")
	}

	// Open a real connection the same way cmd/workflow_meta_import does, then
	// wrap it as a db.Provider via db.NewDB so we exercise the production
	// DAO -> Provider path end to end.
	provider, err := db.NewDB(
		mysql.Open(dsn),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Warn)},
	)
	require.NoError(t, err, "connect real MySQL via db.NewDB")

	wfProvider := NewWorkflowProvider(provider)

	// Two known production rows confirmed present in observability_workflow_meta.
	const (
		idTransfer     int64 = 7621100656002072576
		nameTransfer         = "tranfer_bbw_2603"
		idRecentPayee  int64 = 7642550560138199040
		nameRecentPayee      = "trans_recent_payee_bbw_2603_1"
	)

	// Production span shape: workflow node spans carry sub_workflow_id as an
	// Int64 in tags_long, keyed by "{traceID}-{spanID}" in the result map.
	spans := loop_span.SpanList{
		{
			TraceID:  "it-trace-1",
			SpanID:   "it-span-1",
			TagsLong: map[string]int64{"sub_workflow_id": idTransfer},
		},
		{
			TraceID:  "it-trace-2",
			SpanID:   "it-span-2",
			TagsLong: map[string]int64{"sub_workflow_id": idRecentPayee},
		},
	}

	got, err := wfProvider.BatchGetWorkflows(context.Background(), spans)
	require.NoError(t, err)
	require.Len(t, got, 2)

	type info struct {
		ID      int64  `json:"id"`
		Name    string `json:"name"`
		IconURI string `json:"icon_uri"`
	}

	var i1 info
	require.NoError(t, json.Unmarshal([]byte(got["it-trace-1-it-span-1"]), &i1))
	assert.Equal(t, idTransfer, i1.ID)
	assert.Contains(t, i1.Name, nameTransfer)
	t.Logf("resolved %d -> name=%q icon_uri=%q", i1.ID, i1.Name, i1.IconURI)

	var i2 info
	require.NoError(t, json.Unmarshal([]byte(got["it-trace-2-it-span-2"]), &i2))
	assert.Equal(t, idRecentPayee, i2.ID)
	assert.Contains(t, i2.Name, nameRecentPayee)
	t.Logf("resolved %d -> name=%q icon_uri=%q", i2.ID, i2.Name, i2.IconURI)
}
