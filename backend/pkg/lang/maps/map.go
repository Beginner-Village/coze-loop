// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package maps

import (
	"github.com/bytedance/gg/gmap"
)

func ToSlice[K comparable, V any, R any](m map[K]V, iteratee func(k K, v V) R) []R {
	return gmap.ToSlice(m, iteratee)
}
