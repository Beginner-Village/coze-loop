// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package vfs

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCSVReader(t *testing.T) {
	for _, tc := range []struct {
		name     string
		filename string
		wantErr  assert.ErrorAssertionFunc
		wantData [][]string
	}{
		{
			name:     "bom",
			filename: "testdata/bom.csv",
			wantErr:  assert.NoError,
			wantData: [][]string{{"Input", "Output"}, {"#1-in😼", "#1-out"}},
		},
		{
			name:     "GB-18030",
			filename: "testdata/gb18030.csv",
			wantErr:  assert.NoError,
			wantData: [][]string{{"姓名😼", "性别", "年龄", "生日", "职业"}, {"小王", "男", "25", "2-Jan", "工程师"}},
		},
		{
			name:     "GBK",
			filename: "testdata/gbk.csv",
			wantErr:  assert.NoError,
			wantData: [][]string{{"姓名?", "性别", "年龄", "生日", "职业"}, {"小王", "男", "25", "2-Jan", "工程师"}}, // GBK does not support emoji.
		},
		{
			name:     "detect no confidence, try gb18030",
			filename: "testdata/exl_gb_default_csv.csv",
			wantErr:  assert.NoError,
			wantData: [][]string{{"in", "out", "desc", "繁体", "符号"}, {"中文", "全角【】ｎａｍｅ", "你好？", "導入文件", "1354+—)(*&^%$#!@"}}, // GBK does not support emoji.
		},
		{
			name:     "lazy quote",
			filename: "testdata/lazy_quote.csv",
			wantErr:  assert.NoError,
			wantData: [][]string{{"id", "name", "description"}, {"1", "Apple", "A \"red\" fruit\n2,Banana,\"A yellow fruit"}},
		},
		{
			name:     "cut utf-8 in the middle",
			filename: "testdata/utf8_cut.csv",
			wantErr:  assert.NoError,
			wantData: [][]string{{"question", "选项A", "选项B", "选项C", "选项D", "选项E", "选项F", "实际答案"}, {"小明得了-1分，他的分级是什么？", "A. 异常", "B. 不及格", "C. 及格", "D. 良好", "E. 优秀", "F. 完美", "A"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := tc.filename
			r, err := os.Open(name)
			require.NoError(t, err)

			got, err := NewCSVReader(r)
			if !tc.wantErr(t, err) {
				return
			}

			l1, err := got.Read()
			require.NoError(t, err)
			l2, err := got.Read()
			require.NoError(t, err)
			assert.Equal(t, [][]string{l1, l2}, tc.wantData)
		})
	}
}
