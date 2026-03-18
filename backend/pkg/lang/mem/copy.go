// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package mem

import (
	"encoding/json"
)

func DeepCopy(src, dst interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}
