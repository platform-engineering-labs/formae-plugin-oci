// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package containerengine

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/containerengine"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

// CheckWorkRequestStatus polls a WorkRequest and converts to a formae ProgressResult.
// This is shared across all ContainerEngine resources (Cluster, NodePool, VirtualNodePool)
// since they all use the same async pattern.
func CheckWorkRequestStatus(
	ctx context.Context,
	client *containerengine.ContainerEngineClient,
	workRequestId string,
	operation resource.Operation,
) (*resource.ProgressResult, error) {
	resp, err := client.GetWorkRequest(ctx, containerengine.GetWorkRequestRequest{
		WorkRequestId: common.String(workRequestId),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get work request %s: %w", workRequestId, err)
	}

	switch resp.Status {
	case containerengine.WorkRequestStatusSucceeded:
		nativeID := extractResourceId(resp.Resources, containerengine.WorkRequestResourceActionTypeCreated)
		if nativeID == "" {
			// For updates, try to get the updated resource
			nativeID = extractResourceId(resp.Resources, containerengine.WorkRequestResourceActionTypeUpdated)
		}
		if nativeID == "" {
			// For deletes or other operations, try to get any related resource
			nativeID = extractResourceId(resp.Resources, containerengine.WorkRequestResourceActionTypeRelated)
		}
		return &resource.ProgressResult{
			Operation:       operation,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        nativeID,
		}, nil

	case containerengine.WorkRequestStatusFailed:
		errorMsg := getWorkRequestErrors(ctx, client, workRequestId, resp.CompartmentId)
		return &resource.ProgressResult{
			Operation:       operation,
			OperationStatus: resource.OperationStatusFailure,
			StatusMessage:   errorMsg,
		}, nil

	case containerengine.WorkRequestStatusCanceled:
		return &resource.ProgressResult{
			Operation:       operation,
			OperationStatus: resource.OperationStatusFailure,
			StatusMessage:   "Operation was canceled",
		}, nil

	default: // ACCEPTED, IN_PROGRESS, CANCELING
		return &resource.ProgressResult{
			Operation:       operation,
			OperationStatus: resource.OperationStatusInProgress,
			RequestID:       workRequestId,
		}, nil
	}
}

// extractResourceId finds the resource identifier from WorkRequest resources by action type
func extractResourceId(resources []containerengine.WorkRequestResource, actionType containerengine.WorkRequestResourceActionTypeEnum) string {
	for _, r := range resources {
		if r.ActionType == actionType && r.Identifier != nil {
			return *r.Identifier
		}
	}
	return ""
}

// getWorkRequestErrors retrieves error messages from a failed WorkRequest
func getWorkRequestErrors(ctx context.Context, client *containerengine.ContainerEngineClient, workRequestId string, compartmentId *string) string {
	if compartmentId == nil {
		return "Work request failed (no compartment ID to retrieve errors)"
	}

	resp, err := client.ListWorkRequestErrors(ctx, containerengine.ListWorkRequestErrorsRequest{
		WorkRequestId: common.String(workRequestId),
		CompartmentId: compartmentId,
	})
	if err != nil {
		return fmt.Sprintf("Work request failed (could not retrieve error details: %v)", err)
	}

	if len(resp.Items) == 0 {
		return "Work request failed (no error details available)"
	}

	var messages []string
	for _, item := range resp.Items {
		if item.Message != nil {
			messages = append(messages, *item.Message)
		}
	}

	if len(messages) == 0 {
		return "Work request failed (no error messages)"
	}

	return strings.Join(messages, "; ")
}

// CreateInProgressResult creates a standard in-progress result with a WorkRequest ID
func CreateInProgressResult(operation resource.Operation, workRequestId string) *resource.ProgressResult {
	return &resource.ProgressResult{
		Operation:       operation,
		OperationStatus: resource.OperationStatusInProgress,
		RequestID:       workRequestId,
	}
}
