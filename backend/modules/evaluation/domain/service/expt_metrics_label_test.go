// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"testing"

	"github.com/coze-dev/coze-loop/backend/modules/evaluation/domain/entity"
)

func TestExptStatusLabel(t *testing.T) {
	cases := []struct {
		in   entity.ExptStatus
		want string
	}{
		{entity.ExptStatus_Success, "success"},
		{entity.ExptStatus_Failed, "failed"},
		{entity.ExptStatus_Terminated, "terminated"},
		{entity.ExptStatus_SystemTerminated, "system_terminated"},
		{entity.ExptStatus_Pending, "other"},
		{entity.ExptStatus_Processing, "other"},
	}
	for _, c := range cases {
		if got := exptStatusLabel(c.in); got != c.want {
			t.Errorf("exptStatusLabel(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}
