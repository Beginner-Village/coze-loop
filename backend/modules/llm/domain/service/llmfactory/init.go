// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package llmfactory

func NewFactory() IFactory {
	return &FactoryImpl{}
}
