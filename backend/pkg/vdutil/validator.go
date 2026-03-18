// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package vdutil

import (
	validator "github.com/go-playground/validator/v10"
)

var validate = validator.New()

func Validate(s any) error {
	return validate.Struct(s)
}
