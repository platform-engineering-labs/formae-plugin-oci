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

// HandleOCIServiceError converts OCI service errors to OperationErrorCode.
// Returns the error code and true if the error was identified, false otherwise.
// This function unwraps errors to find OCI service errors even when they're wrapped.
func HandleOCIServiceError(err error) (resource.OperationErrorCode, bool) {
	if err == nil {
		return resource.OperationErrorCodeNotSet, false
	}

	// Try to find a service error in the error chain
	var serviceErr common.ServiceError
	currentErr := err
	for currentErr != nil {
		// Check if current error is a service error
		if se, ok := common.IsServiceError(currentErr); ok {
			serviceErr = se
			break
		}
		// Unwrap and try again
		currentErr = errors.Unwrap(currentErr)
	}

	if serviceErr == nil {
		return resource.OperationErrorCodeNotSet, false
	}

	statusCode := serviceErr.GetHTTPStatusCode()
	errorCode := serviceErr.GetCode()

	// Map HTTP status codes and OCI error codes to OperationErrorCode
	switch statusCode {
	case 404:
		return resource.OperationErrorCodeNotFound, true
	case 409:
		// 409 Conflict - resource has dependencies or is in use
		return resource.OperationErrorCodeResourceConflict, true
	case 429:
		// 429 Too Many Requests - throttling
		return resource.OperationErrorCodeThrottling, true
	case 500, 502, 503:
		// Server errors
		return resource.OperationErrorCodeServiceInternalError, true
	case 504:
		// Gateway timeout
		return resource.OperationErrorCodeServiceTimeout, true
	}

	// Also check OCI-specific error codes
	switch errorCode {
	case "NotAuthorizedOrNotFound":
		return resource.OperationErrorCodeNotFound, true
	case "TooManyRequests":
		return resource.OperationErrorCodeThrottling, true
	}

	return resource.OperationErrorCodeNotSet, false
}

// HandleDeleteError wraps delete operations to convert OCI service errors to DeleteResult.
// If the error is a recoverable OCI service error, it returns a DeleteResult with ErrorCode set.
// Otherwise, it returns the original error.
func HandleDeleteError(err error, resourceType string, nativeID string, operationName string) (*resource.DeleteResult, error) {
	if err == nil {
		return nil, nil
	}

	if errorCode, ok := HandleOCIServiceError(err); ok {
		// Extract service error for message (HandleOCIServiceError already unwrapped it)
		var serviceErr common.ServiceError
		currentErr := err
		for currentErr != nil {
			if se, ok := common.IsServiceError(currentErr); ok {
				serviceErr = se
				break
			}
			currentErr = errors.Unwrap(currentErr)
		}
		
		statusMessage := err.Error()
		if serviceErr != nil {
			statusMessage = fmt.Sprintf("%s cannot be deleted: %s", operationName, serviceErr.GetMessage())
		}
		return &resource.DeleteResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationDelete,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       errorCode,
				StatusMessage:   statusMessage,
				NativeID:        nativeID,
			},
		}, nil
	}

	return nil, err
}

// HandleCreateError wraps create operations to convert OCI service errors to CreateResult.
// If the error is a recoverable OCI service error, it returns a CreateResult with ErrorCode set.
// Otherwise, it returns the original error.
func HandleCreateError(err error, resourceType string, operationName string) (*resource.CreateResult, error) {
	if err == nil {
		return nil, nil
	}

	if errorCode, ok := HandleOCIServiceError(err); ok {
		// Extract service error for message (HandleOCIServiceError already unwrapped it)
		var serviceErr common.ServiceError
		currentErr := err
		for currentErr != nil {
			if se, ok := common.IsServiceError(currentErr); ok {
				serviceErr = se
				break
			}
			currentErr = errors.Unwrap(currentErr)
		}
		
		statusMessage := err.Error()
		if serviceErr != nil {
			statusMessage = fmt.Sprintf("%s cannot be created: %s", operationName, serviceErr.GetMessage())
		}
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       errorCode,
				StatusMessage:   statusMessage,
			},
		}, nil
	}

	return nil, err
}

// HandleUpdateError wraps update operations to convert OCI service errors to UpdateResult.
// If the error is a recoverable OCI service error, it returns an UpdateResult with ErrorCode set.
// Otherwise, it returns the original error.
func HandleUpdateError(err error, resourceType string, nativeID string, operationName string) (*resource.UpdateResult, error) {
	if err == nil {
		return nil, nil
	}

	if errorCode, ok := HandleOCIServiceError(err); ok {
		// Extract service error for message (HandleOCIServiceError already unwrapped it)
		var serviceErr common.ServiceError
		currentErr := err
		for currentErr != nil {
			if se, ok := common.IsServiceError(currentErr); ok {
				serviceErr = se
				break
			}
			currentErr = errors.Unwrap(currentErr)
		}
		
		statusMessage := err.Error()
		if serviceErr != nil {
			statusMessage = fmt.Sprintf("%s cannot be updated: %s", operationName, serviceErr.GetMessage())
		}
		return &resource.UpdateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationUpdate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       errorCode,
				StatusMessage:   statusMessage,
				NativeID:        nativeID,
			},
		}, nil
	}

	return nil, err
}

