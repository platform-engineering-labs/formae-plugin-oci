// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package provisioner

import (
	"context"
	"encoding/json"

	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

// readAfterWrite is a decorator that wraps a Provisioner and automatically
// calls Read() after a successful synchronous Create or Update. This ensures
// ResourceProperties always contains the complete set of fields from the API,
// preventing validateRequiredFields from dropping resources due to missing
// schema-required fields.
//
// For async operations (OperationStatusInProgress), the decorator is a no-op —
// properties will come from Status() polling instead.
type readAfterWrite struct {
	inner Provisioner
}

func (w *readAfterWrite) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	result, err := w.inner.Create(ctx, request)
	if err != nil {
		return nil, err
	}

	pr := result.ProgressResult
	if pr.OperationStatus == resource.OperationStatusSuccess && pr.NativeID != "" {
		readResp, readErr := w.inner.Read(ctx, &resource.ReadRequest{
			NativeID:     pr.NativeID,
			ResourceType: request.ResourceType,
			TargetConfig: request.TargetConfig,
		})
		if readErr == nil && readResp.ErrorCode == "" {
			pr.ResourceProperties = json.RawMessage(readResp.Properties)
		}
	}

	return result, nil
}

func (w *readAfterWrite) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	result, err := w.inner.Update(ctx, request)
	if err != nil {
		return nil, err
	}

	pr := result.ProgressResult
	if pr.OperationStatus == resource.OperationStatusSuccess && pr.NativeID != "" {
		readResp, readErr := w.inner.Read(ctx, &resource.ReadRequest{
			NativeID:     pr.NativeID,
			ResourceType: request.ResourceType,
			TargetConfig: request.TargetConfig,
		})
		if readErr == nil && readResp.ErrorCode == "" {
			pr.ResourceProperties = json.RawMessage(readResp.Properties)
		}
	}

	return result, nil
}

func (w *readAfterWrite) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	return w.inner.Delete(ctx, request)
}

func (w *readAfterWrite) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	return w.inner.Status(ctx, request)
}

func (w *readAfterWrite) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	return w.inner.Read(ctx, request)
}

func (w *readAfterWrite) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	return w.inner.List(ctx, request)
}
