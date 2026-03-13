// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package util

import (
	"errors"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

// extractServiceError walks the error chain to find an OCI ServiceError.
func extractServiceError(err error) common.ServiceError {
	for currentErr := err; currentErr != nil; currentErr = errors.Unwrap(currentErr) {
		if se, ok := common.IsServiceError(currentErr); ok {
			return se
		}
	}
	return nil
}

// HandleOCIServiceError converts OCI service errors to OperationErrorCode.
// Returns the error code and true if the error was identified, false otherwise.
// OCI error codes are checked first (precise), then HTTP status codes (fallback).
func HandleOCIServiceError(err error) (resource.OperationErrorCode, bool) {
	if err == nil {
		return resource.OperationErrorCodeNotSet, false
	}

	serviceErr := extractServiceError(err)
	if serviceErr == nil {
		return resource.OperationErrorCodeNotSet, false
	}

	// OCI error codes — precise, checked first
	switch serviceErr.GetCode() {
	case "NotAuthorizedOrNotFound", "RelatedResourceNotAuthorizedOrNotFound":
		return resource.OperationErrorCodeNotFound, true
	case "IncorrectState":
		return resource.OperationErrorCodeNotStabilized, true
	case "ResourceAlreadyExists", "NotAuthorizedOrResourceAlreadyExists":
		return resource.OperationErrorCodeAlreadyExists, true
	case "TooManyRequests":
		return resource.OperationErrorCodeThrottling, true
	case "LimitExceeded":
		return resource.OperationErrorCodeServiceLimitExceeded, true
	case "InsufficientServicePermissions":
		return resource.OperationErrorCodeAccessDenied, true
	case "InvalidParameter":
		return resource.OperationErrorCodeInvalidRequest, true
	}

	// HTTP status codes — fallback for codes not explicitly handled above
	switch serviceErr.GetHTTPStatusCode() {
	case 404:
		return resource.OperationErrorCodeNotFound, true
	case 409:
		return resource.OperationErrorCodeResourceConflict, true
	case 429:
		return resource.OperationErrorCodeThrottling, true
	case 500, 502, 503:
		return resource.OperationErrorCodeServiceInternalError, true
	case 504:
		return resource.OperationErrorCodeServiceTimeout, true
	}

	return resource.OperationErrorCodeNotSet, false
}

// serviceErrorMessage extracts the OCI service error message, falling back to err.Error().
func serviceErrorMessage(err error, operationName string, action string) string {
	if se := extractServiceError(err); se != nil {
		return fmt.Sprintf("%s cannot be %s: %s", operationName, action, se.GetMessage())
	}
	return err.Error()
}

// HandleDeleteError converts OCI service errors to a DeleteResult with ErrorCode set.
// Returns (nil, original error) for non-OCI errors.
func HandleDeleteError(err error, resourceType string, nativeID string, operationName string) (*resource.DeleteResult, error) {
	if err == nil {
		return nil, nil
	}

	errorCode, ok := HandleOCIServiceError(err)
	if !ok {
		return nil, err
	}

	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationDelete,
			OperationStatus: resource.OperationStatusFailure,
			ErrorCode:       errorCode,
			StatusMessage:   serviceErrorMessage(err, operationName, "deleted"),
			NativeID:        nativeID,
		},
	}, nil
}

// HandleCreateError converts OCI service errors to a CreateResult with ErrorCode set.
// Returns (nil, original error) for non-OCI errors.
func HandleCreateError(err error, resourceType string, operationName string) (*resource.CreateResult, error) {
	if err == nil {
		return nil, nil
	}

	errorCode, ok := HandleOCIServiceError(err)
	if !ok {
		return nil, err
	}

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCreate,
			OperationStatus: resource.OperationStatusFailure,
			ErrorCode:       errorCode,
			StatusMessage:   serviceErrorMessage(err, operationName, "created"),
		},
	}, nil
}

// HandleUpdateError converts OCI service errors to an UpdateResult with ErrorCode set.
// Returns (nil, original error) for non-OCI errors.
func HandleUpdateError(err error, resourceType string, nativeID string, operationName string) (*resource.UpdateResult, error) {
	if err == nil {
		return nil, nil
	}

	errorCode, ok := HandleOCIServiceError(err)
	if !ok {
		return nil, err
	}

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationUpdate,
			OperationStatus: resource.OperationStatusFailure,
			ErrorCode:       errorCode,
			StatusMessage:   serviceErrorMessage(err, operationName, "updated"),
			NativeID:        nativeID,
		},
	}, nil
}
