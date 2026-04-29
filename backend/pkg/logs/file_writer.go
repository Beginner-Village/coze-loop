// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package logs

import (
	"io"
	"os"
	"strconv"

	"gopkg.in/natefinch/lumberjack.v2"
)

// NewWriter returns the writer used by the default logger.
// If LOG_FILE is empty, only stdout is used; otherwise logs are dual-written
// to stdout and the rotating file backed by lumberjack.
//
// Knobs (env, all optional):
//
//	LOG_FILE          — path to log file (default: stdout-only)
//	LOG_MAX_SIZE_MB   — rotate at N megabytes (default 100)
//	LOG_MAX_BACKUPS   — keep N rotated backups (default 7)
//	LOG_MAX_AGE_DAYS  — discard backups older than N days (default 30)
//	LOG_COMPRESS      — gzip rotated backups (default true)
func NewWriter() io.Writer {
	logFile := os.Getenv("LOG_FILE")
	if logFile == "" {
		return os.Stdout
	}
	lj := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    envInt("LOG_MAX_SIZE_MB", 100),
		MaxBackups: envInt("LOG_MAX_BACKUPS", 7),
		MaxAge:     envInt("LOG_MAX_AGE_DAYS", 30),
		Compress:   envBool("LOG_COMPRESS", true),
	}
	return io.MultiWriter(os.Stdout, lj)
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
