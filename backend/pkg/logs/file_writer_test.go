// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package logs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewWriter_NoFileWhenLogFileEmpty(t *testing.T) {
	t.Setenv("LOG_FILE", "")
	w := NewWriter()
	if w == nil {
		t.Fatal("expected writer, got nil")
	}
}

func TestNewWriter_CreatesFileWhenLogFileSet(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "app.log")
	t.Setenv("LOG_FILE", logPath)
	w := NewWriter()
	if _, err := w.Write([]byte("hello\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("expected log file, got error: %v", err)
	}
}

func TestNewWriter_RotatesAtMaxSize(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "app.log")
	t.Setenv("LOG_FILE", logPath)
	t.Setenv("LOG_MAX_SIZE_MB", "1")
	t.Setenv("LOG_MAX_BACKUPS", "3")
	w := NewWriter()
	chunk := make([]byte, 4096)
	for i := range chunk {
		chunk[i] = 'A'
	}
	for i := 0; i < 400; i++ {
		_, _ = w.Write(chunk)
	}
	entries, _ := os.ReadDir(tmp)
	count := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "app") {
			count++
		}
	}
	if count < 2 {
		t.Fatalf("expected ≥2 log files after rotation, got %d", count)
	}
}
