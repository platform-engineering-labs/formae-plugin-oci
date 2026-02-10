// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFreeformTagsToList_Nil(t *testing.T) {
	assert.Nil(t, FreeformTagsToList(nil))
}

func TestFreeformTagsToList_SortedByKey(t *testing.T) {
	tags := map[string]string{
		"Env":     "prod",
		"App":     "web",
		"Creator": "alice",
	}

	got := FreeformTagsToList(tags)

	assert.Equal(t, []map[string]string{
		{"Key": "App", "Value": "web"},
		{"Key": "Creator", "Value": "alice"},
		{"Key": "Env", "Value": "prod"},
	}, got)

	// Run again to confirm determinism
	assert.Equal(t, got, FreeformTagsToList(tags))
}

func TestDefinedTagsToList_Nil(t *testing.T) {
	assert.Nil(t, DefinedTagsToList(nil))
}

func TestDefinedTagsToList_SortedByNamespaceThenKey(t *testing.T) {
	tags := map[string]map[string]any{
		"Operations": {
			"CostCenter": "42",
			"Team":       "platform",
		},
		"AppConfig": {
			"Env": "prod",
		},
	}

	got := DefinedTagsToList(tags)

	assert.Equal(t, []map[string]any{
		{"Namespace": "AppConfig", "Key": "Env", "Value": "prod"},
		{"Namespace": "Operations", "Key": "CostCenter", "Value": "42"},
		{"Namespace": "Operations", "Key": "Team", "Value": "platform"},
	}, got)
}
