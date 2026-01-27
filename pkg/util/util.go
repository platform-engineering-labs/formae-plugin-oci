// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package util

import "fmt"

// ExtractTag converts a map[string]any to map[string]string for OCI FreeformTags
func ExtractTag(props map[string]any, key string) (map[string]string, bool) {
	if tags, ok := props[key].(map[string]any); ok && len(tags) > 0 {
		result := make(map[string]string, len(tags))
		for k, v := range tags {
			result[k] = fmt.Sprintf("%v", v)
		}
		return result, true
	}
	return nil, false
}

// ExtractNestedTag converts a map[string]any to map[string]map[string]any for OCI DefinedTags
func ExtractNestedTag(props map[string]any, key string) (map[string]map[string]any, bool) {
	if tags, ok := props[key].(map[string]any); ok && len(tags) > 0 {
		result := make(map[string]map[string]any, len(tags))
		for k, v := range tags {
			if m, ok := v.(map[string]any); ok {
				result[k] = m
			}
		}
		if len(result) > 0 {
			return result, true
		}
	}
	return nil, false
}

// validateString checks if a value is a non-empty string or a resolved reference
func validateString(val any) (string, bool) {
	// Case 1: Direct string
	if str, ok := val.(string); ok && str != "" {
		return str, true
	}
	// Case 2: Reference object - check $value for resolved value
	if ref, ok := val.(map[string]any); ok {
		// Try $value first (standard resolved reference)
		if value, ok := ref["$value"].(string); ok && value != "" {
			return value, true
		}
		// If we have $ref but no $value, the reference is unresolved
		// This can happen for nested refs during Create operations
		if _, hasRef := ref["$ref"]; hasRef {
			// Reference exists but not resolved yet - return empty
			return "", false
		}
	}
	return "", false
}

// ExtractString extracts an optional string property, returning it only if present and non-empty
func ExtractString(props map[string]any, key string) (string, bool) {
	return validateString(props[key])
}

// ExtractBool extracts an optional bool property
func ExtractBool(props map[string]any, key string) (bool, bool) {
	if val, ok := props[key].(bool); ok {
		return val, true
	}
	return false, false
}

// ExtractStringSlice converts []any to []string, reusing ExtractString's validation
func ExtractStringSlice(props map[string]any, key string) ([]string, bool) {
	if slice, ok := props[key].([]any); ok && len(slice) > 0 {
		result := make([]string, 0, len(slice))
		for _, v := range slice {
			if str, ok := validateString(v); ok {
				result = append(result, str)
			} else {
				// Invalid element - not a non-empty string
				return nil, false
			}
		}
		if len(result) > 0 {
			return result, true
		}
	}
	return nil, false
}

// ExtractResolvedReference extracts a string value from either:
// - A plain string
// - A reference object with "$value" key (resolved reference from formae)
// This handles the case where references may or may not be resolved yet.
func ExtractResolvedReference(props map[string]any, key string) (string, bool) {
	val := props[key]

	// Case 1: Already a plain string
	if str, ok := val.(string); ok && str != "" {
		return str, true
	}

	// Case 2: Reference object with $value field (resolved reference)
	if ref, ok := val.(map[string]any); ok {
		if value, ok := ref["$value"].(string); ok && value != "" {
			return value, true
		}
	}

	return "", false
}
