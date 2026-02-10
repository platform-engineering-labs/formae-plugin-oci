// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package util

import "sort"

// ExtractFreeformTags converts Listing<oci.FreeformTag> ([{Key, Value}]) to map[string]string for OCI API
func ExtractFreeformTags(props map[string]any, key string) (map[string]string, bool) {
	slice, ok := props[key].([]any)
	if !ok || len(slice) == 0 {
		return nil, false
	}
	result := make(map[string]string, len(slice))
	for _, item := range slice {
		if tag, ok := item.(map[string]any); ok {
			k, _ := tag["Key"].(string)
			v, _ := tag["Value"].(string)
			if k != "" {
				result[k] = v
			}
		}
	}
	if len(result) > 0 {
		return result, true
	}
	return nil, false
}

// ExtractDefinedTags converts Listing<oci.DefinedTag> ([{Namespace, Key, Value}]) to map[string]map[string]any for OCI API
func ExtractDefinedTags(props map[string]any, key string) (map[string]map[string]any, bool) {
	slice, ok := props[key].([]any)
	if !ok || len(slice) == 0 {
		return nil, false
	}
	result := make(map[string]map[string]any)
	for _, item := range slice {
		if tag, ok := item.(map[string]any); ok {
			ns, _ := tag["Namespace"].(string)
			k, _ := tag["Key"].(string)
			v := tag["Value"]
			if ns != "" && k != "" {
				if result[ns] == nil {
					result[ns] = make(map[string]any)
				}
				result[ns][k] = v
			}
		}
	}
	if len(result) > 0 {
		return result, true
	}
	return nil, false
}

// FreeformTagsToList converts OCI's map[string]string to Listing<oci.FreeformTag> format for responses
func FreeformTagsToList(tags map[string]string) []map[string]string {
	if len(tags) == 0 {
		return nil
	}
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	result := make([]map[string]string, 0, len(tags))
	for _, k := range keys {
		result = append(result, map[string]string{"Key": k, "Value": tags[k]})
	}
	return result
}

// DefinedTagsToList converts OCI's map[string]map[string]any to Listing<oci.DefinedTag> format for responses
func DefinedTagsToList(tags map[string]map[string]any) []map[string]any {
	if len(tags) == 0 {
		return nil
	}
	namespaces := make([]string, 0, len(tags))
	for ns := range tags {
		namespaces = append(namespaces, ns)
	}
	sort.Strings(namespaces)
	result := make([]map[string]any, 0)
	for _, ns := range namespaces {
		keys := make([]string, 0, len(tags[ns]))
		for k := range tags[ns] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			result = append(result, map[string]any{"Namespace": ns, "Key": k, "Value": tags[ns][k]})
		}
	}
	return result
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
