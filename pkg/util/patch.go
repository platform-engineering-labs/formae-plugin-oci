// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package util

import (
	"context"
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

func ApplyPatchDocument(
	ctx context.Context,
	request *resource.UpdateRequest,
	readFunc func(ctx context.Context, readReq *resource.ReadRequest) (*resource.ReadResult, error),
) (map[string]any, error) {
	if request.PatchDocument == nil || *request.PatchDocument == "" {
		var props map[string]any
		if err := json.Unmarshal(request.DesiredProperties, &props); err != nil {
			return nil, fmt.Errorf("failed to parse properties: %w", err)
		}
		return props, nil
	}

	readReq := &resource.ReadRequest{
		NativeID:     request.NativeID,
		ResourceType: request.ResourceType,
		TargetConfig: request.TargetConfig,
	}
	readResult, err := readFunc(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to read existing resource: %w", err)
	}

	existingJSON := []byte(readResult.Properties)
	patchJSON := []byte(*request.PatchDocument)

	patch, err := jsonpatch.DecodePatch(patchJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to decode patch document: %w", err)
	}

	patchedJSON, err := patch.Apply(existingJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to apply patch: %w", err)
	}

	var mergedProps map[string]any
	if err := json.Unmarshal(patchedJSON, &mergedProps); err != nil {
		return nil, fmt.Errorf("failed to parse merged properties: %w", err)
	}

	return mergedProps, nil
}

