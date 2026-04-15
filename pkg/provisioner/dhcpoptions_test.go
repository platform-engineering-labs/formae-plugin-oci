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

func TestDhcpOptionsRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/dhcps/ocid1.dhcpoptions..aaa"}: {200, newTestDhcpOptionsBody("AVAILABLE")},
		})
		p := core.NewDhcpOptionsProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.dhcpoptions..aaa"})
		require.NoError(t, err)
		assert.Empty(t, result.ErrorCode)

		var props map[string]any
		require.NoError(t, json.Unmarshal([]byte(result.Properties), &props))
		assert.Equal(t, "test-dhcp-options", props["DisplayName"])
	})

	t.Run("not_found", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/dhcps/ocid1.dhcpoptions..missing"}: {404, `{"code":"NotAuthorizedOrNotFound","message":"not found"}`},
		})
		p := core.NewDhcpOptionsProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.dhcpoptions..missing"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})

	t.Run("terminal_state", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/dhcps/ocid1.dhcpoptions..aaa"}: {200, newTestDhcpOptionsBody("TERMINATED")},
		})
		p := core.NewDhcpOptionsProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.dhcpoptions..aaa"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})
}

func TestDhcpOptionsCreate(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"POST", "/20160918/dhcps"}: {200, newTestDhcpOptionsBody("AVAILABLE")},
	})
	p := core.NewDhcpOptionsProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{
		"CompartmentId": "ocid1.compartment..xxx",
		"VcnId":         "ocid1.vcn..aaa",
		"DisplayName":   "test-dhcp-options",
		"Options": []map[string]any{
			{
				"type":       "DomainNameServer",
				"serverType": "VcnLocalPlusInternet",
			},
		},
	})
	require.NoError(t, err)

	result, err := p.Create(context.Background(), &resource.CreateRequest{
		ResourceType: "OCI::Core::DhcpOptions",
		Properties:   props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
	assert.Equal(t, "ocid1.dhcpoptions..aaa", result.ProgressResult.NativeID)
}

func TestDhcpOptionsUpdate(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/dhcps/ocid1.dhcpoptions..aaa"}: {200, newTestDhcpOptionsBody("AVAILABLE")},
		{"PUT", "/20160918/dhcps/ocid1.dhcpoptions..aaa"}: {200, newTestDhcpOptionsBody("AVAILABLE")},
	})
	p := core.NewDhcpOptionsProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{"DisplayName": "updated-dhcp-options"})
	require.NoError(t, err)

	result, err := p.Update(context.Background(), &resource.UpdateRequest{
		NativeID:          "ocid1.dhcpoptions..aaa",
		ResourceType:      "OCI::Core::DhcpOptions",
		DesiredProperties: props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
}

func TestDhcpOptionsDelete(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/dhcps/ocid1.dhcpoptions..aaa"}:    {200, newTestDhcpOptionsBody("AVAILABLE")},
		{"DELETE", "/20160918/dhcps/ocid1.dhcpoptions..aaa"}: {204, ""},
	})
	p := core.NewDhcpOptionsProvisionerWithSvc(svc)

	result, err := p.Delete(context.Background(), &resource.DeleteRequest{NativeID: "ocid1.dhcpoptions..aaa"})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
}

func TestDhcpOptionsList(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/dhcps"}: {200, fmt.Sprintf(`[%s]`, newTestDhcpOptionsBody("AVAILABLE"))},
	})
	p := core.NewDhcpOptionsProvisionerWithSvc(svc)

	result, err := p.List(context.Background(), &resource.ListRequest{
		ResourceType:         "OCI::Core::DhcpOptions",
		AdditionalProperties: map[string]string{"CompartmentId": "ocid1.compartment..xxx"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"ocid1.dhcpoptions..aaa"}, result.NativeIDs)
}

// Helpers

func newTestDhcpOptionsBody(lifecycleState string) string {
	return fmt.Sprintf(`{
		"id": "ocid1.dhcpoptions..aaa",
		"compartmentId": "ocid1.compartment..xxx",
		"vcnId": "ocid1.vcn..aaa",
		"displayName": "test-dhcp-options",
		"options": [{"type": "DomainNameServer", "serverType": "VcnLocalPlusInternet"}],
		"lifecycleState": %q
	}`, lifecycleState)
}
