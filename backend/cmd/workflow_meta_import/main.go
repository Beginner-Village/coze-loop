// Copyright (c) 2026 coze-dev Authors
// SPDX-License-Identifier: Apache-2.0

// Command workflow_meta_import bulk-loads a production workflow_meta TSV snapshot
// into the observability_workflow_meta table so that the WorkflowProvider can
// resolve span workflow_id -> name/icon.
//
// The snapshot is produced read-only from production via
// _workflow_export/export-workflow-meta.sh. This tool never connects to
// production; it only writes the already-exported local file into the target
// (dev/onsite) database addressed by -dsn.
//
// Usage:
//
//	go run ./cmd/workflow_meta_import \
//	  -dsn 'user:pass@tcp(127.0.0.1:3306)/cozeloop?charset=utf8mb4&parseTime=true&loc=Local' \
//	  -file /path/to/workflow_meta_all.tsv
//
// The TSV must carry the header row:
//
//	id  name  icon_uri  space_id  app_id  status  content_type  mode  created_at  updated_at  latest_version
//
// created_at/updated_at are unix milliseconds. Rows are upserted by primary key
// (id), so re-running with a fresh snapshot is safe and idempotent.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	"github.com/coze-dev/coze-loop/backend/modules/observability/infra/repo/mysql/gorm_gen/model"
)

func main() {
	var (
		dsn       string
		file      string
		batchSize int
	)
	flag.StringVar(&dsn, "dsn", "", "MySQL DSN, e.g. user:pass@tcp(host:3306)/db?charset=utf8mb4&parseTime=true&loc=Local")
	flag.StringVar(&file, "file", "", "path to workflow_meta TSV snapshot")
	flag.IntVar(&batchSize, "batch", 500, "rows per upsert batch")
	flag.Parse()

	if dsn == "" || file == "" {
		flag.Usage()
		log.Fatal("both -dsn and -file are required")
	}

	rows, err := parseTSV(file)
	if err != nil {
		log.Fatalf("parse tsv: %v", err)
	}
	log.Printf("parsed %d workflow_meta rows from %s", len(rows), file)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		log.Fatalf("open db: %v", err)
	}

	if err := db.AutoMigrate(&model.ObservabilityWorkflowMeta{}); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}

	// Upsert by primary key so re-imports overwrite stale metadata.
	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name", "icon_uri", "space_id", "app_id", "status",
			"content_type", "mode", "latest_version", "created_at",
			"updated_at", "snapshot_at",
		}),
	}).CreateInBatches(rows, batchSize)
	if tx.Error != nil {
		log.Fatalf("upsert: %v", tx.Error)
	}
	log.Printf("imported %d rows into observability_workflow_meta", len(rows))
}

func parseTSV(path string) ([]*model.ObservabilityWorkflowMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	now := time.Now()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 8*1024*1024)

	var out []*model.ObservabilityWorkflowMeta
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if lineNo == 1 {
			// header
			if !strings.HasPrefix(line, "id\t") {
				return nil, fmt.Errorf("unexpected header: %q", line)
			}
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		cols := strings.Split(line, "\t")
		if len(cols) < 11 {
			return nil, fmt.Errorf("line %d: expected 11 columns, got %d", lineNo, len(cols))
		}
		id, err := strconv.ParseInt(cols[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: bad id %q: %w", lineNo, cols[0], err)
		}
		out = append(out, &model.ObservabilityWorkflowMeta{
			ID:            id,
			Name:          cols[1],
			IconURI:       cols[2],
			SpaceID:       atoiDefault(cols[3]),
			AppID:         atoiDefault(cols[4]),
			Status:        int32(atoiDefault(cols[5])),
			ContentType:   int32(atoiDefault(cols[6])),
			Mode:          int32(atoiDefault(cols[7])),
			CreatedAt:     millisToTime(cols[8]),
			UpdatedAt:     millisToTime(cols[9]),
			LatestVersion: cols[10],
			SnapshotAt:    now,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func atoiDefault(s string) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func millisToTime(s string) time.Time {
	ms := atoiDefault(s)
	if ms <= 0 {
		return time.Unix(0, 0)
	}
	return time.UnixMilli(ms)
}
