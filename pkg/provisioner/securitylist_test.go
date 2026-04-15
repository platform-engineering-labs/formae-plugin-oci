// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

//go:build integration

package provisioner_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner/core"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityListRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/securityLists/ocid1.securitylist..aaa"}: {200, newTestSecurityListBody("AVAILABLE")},
		})
		p := core.NewSecurityListProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.securitylist..aaa"})
		require.NoError(t, err)
		assert.Empty(t, result.ErrorCode)

		var props map[string]any
		require.NoError(t, json.Unmarshal([]byte(result.Properties), &props))
		assert.Equal(t, "test-sl", props["DisplayName"])
	})

	t.Run("not_found", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/securityLists/ocid1.securitylist..missing"}: {404, `{"code":"NotAuthorizedOrNotFound","message":"not found"}`},
		})
		p := core.NewSecurityListProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.securitylist..missing"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})

	t.Run("terminal_state", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/securityLists/ocid1.securitylist..aaa"}: {200, newTestSecurityListBody("TERMINATED")},
		})
		p := core.NewSecurityListProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.securitylist..aaa"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})
}

func TestSecurityListCreate(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"POST", "/20160918/securityLists"}: {200, newTestSecurityListBody("AVAILABLE")},
	})
	p := core.NewSecurityListProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{
		"CompartmentId": "ocid1.compartment..xxx",
		"VcnId":         "ocid1.vcn..aaa",
		"DisplayName":   "test-sl",
		"IngressSecurityRules": []map[string]any{
			{
				"Protocol": "6",
				"Source":   "0.0.0.0/0",
			},
		},
		"EgressSecurityRules": []map[string]any{
			{
				"Protocol":    "all",
				"Destination": "0.0.0.0/0",
			},
		},
	})
	require.NoError(t, err)

	result, err := p.Create(context.Background(), &resource.CreateRequest{
		ResourceType: "OCI::Core::SecurityList",
		Properties:   props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
	assert.Equal(t, "ocid1.securitylist..aaa", result.ProgressResult.NativeID)
}

func TestSecurityListUpdate(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/securityLists/ocid1.securitylist..aaa"}: {200, newTestSecurityListBody("AVAILABLE")},
		{"PUT", "/20160918/securityLists/ocid1.securitylist..aaa"}: {200, newTestSecurityListBody("AVAILABLE")},
	})
	p := core.NewSecurityListProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{"DisplayName": "updated-sl"})
	require.NoError(t, err)

	result, err := p.Update(context.Background(), &resource.UpdateRequest{
		NativeID:          "ocid1.securitylist..aaa",
		ResourceType:      "OCI::Core::SecurityList",
		DesiredProperties: props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
}

func TestSecurityListDelete(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/securityLists/ocid1.securitylist..aaa"}:    {200, newTestSecurityListBody("AVAILABLE")},
		{"DELETE", "/20160918/securityLists/ocid1.securitylist..aaa"}: {204, ""},
	})
	p := core.NewSecurityListProvisionerWithSvc(svc)

	result, err := p.Delete(context.Background(), &resource.DeleteRequest{NativeID: "ocid1.securitylist..aaa"})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
}

func TestSecurityListList(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/securityLists"}: {200, fmt.Sprintf(`[%s]`, newTestSecurityListBody("AVAILABLE"))},
	})
	p := core.NewSecurityListProvisionerWithSvc(svc)

	result, err := p.List(context.Background(), &resource.ListRequest{
		ResourceType:         "OCI::Core::SecurityList",
		AdditionalProperties: map[string]string{"CompartmentId": "ocid1.compartment..xxx"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"ocid1.securitylist..aaa"}, result.NativeIDs)
}

// Helpers

func newTestSecurityListBody(lifecycleState string) string {
	return fmt.Sprintf(`{
		"id": "ocid1.securitylist..aaa",
		"compartmentId": "ocid1.compartment..xxx",
		"vcnId": "ocid1.vcn..aaa",
		"displayName": "test-sl",
		"ingressSecurityRules": [
			{
				"protocol": "6",
				"source": "0.0.0.0/0"
			}
		],
		"egressSecurityRules": [
			{
				"protocol": "all",
				"destination": "0.0.0.0/0"
			}
		],
		"lifecycleState": %q
	}`, lifecycleState)
}
