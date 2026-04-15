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

func TestNetworkSecurityGroupRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/networkSecurityGroups/ocid1.nsg..aaa"}: {200, newTestNetworkSecurityGroupBody("AVAILABLE")},
		})
		p := core.NewNetworkSecurityGroupProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.nsg..aaa"})
		require.NoError(t, err)
		assert.Empty(t, result.ErrorCode)

		var props map[string]any
		require.NoError(t, json.Unmarshal([]byte(result.Properties), &props))
		assert.Equal(t, "test-nsg", props["DisplayName"])
	})

	t.Run("not_found", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/networkSecurityGroups/ocid1.nsg..missing"}: {404, `{"code":"NotAuthorizedOrNotFound","message":"not found"}`},
		})
		p := core.NewNetworkSecurityGroupProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.nsg..missing"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})

	t.Run("terminal_state", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/networkSecurityGroups/ocid1.nsg..aaa"}: {200, newTestNetworkSecurityGroupBody("TERMINATED")},
		})
		p := core.NewNetworkSecurityGroupProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.nsg..aaa"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})
}

func TestNetworkSecurityGroupCreate(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"POST", "/20160918/networkSecurityGroups"}: {200, newTestNetworkSecurityGroupBody("AVAILABLE")},
	})
	p := core.NewNetworkSecurityGroupProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{
		"CompartmentId": "ocid1.compartment..xxx",
		"VcnId":         "ocid1.vcn..aaa",
		"DisplayName":   "test-nsg",
	})
	require.NoError(t, err)

	result, err := p.Create(context.Background(), &resource.CreateRequest{
		ResourceType: "OCI::Core::NetworkSecurityGroup",
		Properties:   props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
	assert.Equal(t, "ocid1.nsg..aaa", result.ProgressResult.NativeID)
}

func TestNetworkSecurityGroupUpdate(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/networkSecurityGroups/ocid1.nsg..aaa"}: {200, newTestNetworkSecurityGroupBody("AVAILABLE")},
		{"PUT", "/20160918/networkSecurityGroups/ocid1.nsg..aaa"}: {200, newTestNetworkSecurityGroupBody("AVAILABLE")},
	})
	p := core.NewNetworkSecurityGroupProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{"DisplayName": "updated-nsg"})
	require.NoError(t, err)

	result, err := p.Update(context.Background(), &resource.UpdateRequest{
		NativeID:          "ocid1.nsg..aaa",
		ResourceType:      "OCI::Core::NetworkSecurityGroup",
		DesiredProperties: props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
}

func TestNetworkSecurityGroupDelete(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/networkSecurityGroups/ocid1.nsg..aaa"}:    {200, newTestNetworkSecurityGroupBody("AVAILABLE")},
		{"DELETE", "/20160918/networkSecurityGroups/ocid1.nsg..aaa"}: {204, ""},
	})
	p := core.NewNetworkSecurityGroupProvisionerWithSvc(svc)

	result, err := p.Delete(context.Background(), &resource.DeleteRequest{NativeID: "ocid1.nsg..aaa"})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
}

func TestNetworkSecurityGroupList(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/networkSecurityGroups"}: {200, fmt.Sprintf(`[%s]`, newTestNetworkSecurityGroupBody("AVAILABLE"))},
	})
	p := core.NewNetworkSecurityGroupProvisionerWithSvc(svc)

	result, err := p.List(context.Background(), &resource.ListRequest{
		ResourceType:         "OCI::Core::NetworkSecurityGroup",
		AdditionalProperties: map[string]string{"CompartmentId": "ocid1.compartment..xxx"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"ocid1.nsg..aaa"}, result.NativeIDs)
}

// Helpers

func newTestNetworkSecurityGroupBody(lifecycleState string) string {
	return fmt.Sprintf(`{
		"id": "ocid1.nsg..aaa",
		"compartmentId": "ocid1.compartment..xxx",
		"vcnId": "ocid1.vcn..aaa",
		"displayName": "test-nsg",
		"lifecycleState": %q
	}`, lifecycleState)
}
