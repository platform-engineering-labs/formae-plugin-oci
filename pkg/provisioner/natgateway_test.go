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

func TestNatGatewayRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/natGateways/ocid1.natgateway..aaa"}: {200, newTestNatGatewayBody("AVAILABLE")},
		})
		p := core.NewNatGatewayProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.natgateway..aaa"})
		require.NoError(t, err)
		assert.Empty(t, result.ErrorCode)

		var props map[string]any
		require.NoError(t, json.Unmarshal([]byte(result.Properties), &props))
		assert.Equal(t, "test-natgw", props["DisplayName"])
	})

	t.Run("not_found", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/natGateways/ocid1.natgateway..missing"}: {404, `{"code":"NotAuthorizedOrNotFound","message":"not found"}`},
		})
		p := core.NewNatGatewayProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.natgateway..missing"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})

	t.Run("terminal_state", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/natGateways/ocid1.natgateway..aaa"}: {200, newTestNatGatewayBody("TERMINATED")},
		})
		p := core.NewNatGatewayProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.natgateway..aaa"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})
}

func TestNatGatewayCreate(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"POST", "/20160918/natGateways"}: {200, newTestNatGatewayBody("AVAILABLE")},
	})
	p := core.NewNatGatewayProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{
		"CompartmentId": "ocid1.compartment..xxx",
		"VcnId":         "ocid1.vcn..aaa",
		"DisplayName":   "test-natgw",
	})
	require.NoError(t, err)

	result, err := p.Create(context.Background(), &resource.CreateRequest{
		ResourceType: "OCI::Core::NatGateway",
		Properties:   props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
	assert.Equal(t, "ocid1.natgateway..aaa", result.ProgressResult.NativeID)
}

func TestNatGatewayUpdate(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/natGateways/ocid1.natgateway..aaa"}: {200, newTestNatGatewayBody("AVAILABLE")},
		{"PUT", "/20160918/natGateways/ocid1.natgateway..aaa"}: {200, newTestNatGatewayBody("AVAILABLE")},
	})
	p := core.NewNatGatewayProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{"DisplayName": "updated-natgw"})
	require.NoError(t, err)

	result, err := p.Update(context.Background(), &resource.UpdateRequest{
		NativeID:          "ocid1.natgateway..aaa",
		ResourceType:      "OCI::Core::NatGateway",
		DesiredProperties: props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
}

func TestNatGatewayDelete(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/natGateways/ocid1.natgateway..aaa"}:    {200, newTestNatGatewayBody("AVAILABLE")},
		{"DELETE", "/20160918/natGateways/ocid1.natgateway..aaa"}: {204, ""},
	})
	p := core.NewNatGatewayProvisionerWithSvc(svc)

	result, err := p.Delete(context.Background(), &resource.DeleteRequest{NativeID: "ocid1.natgateway..aaa"})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
}

func TestNatGatewayList(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/natGateways"}: {200, fmt.Sprintf(`[%s]`, newTestNatGatewayBody("AVAILABLE"))},
	})
	p := core.NewNatGatewayProvisionerWithSvc(svc)

	result, err := p.List(context.Background(), &resource.ListRequest{
		ResourceType:         "OCI::Core::NatGateway",
		AdditionalProperties: map[string]string{"CompartmentId": "ocid1.compartment..xxx"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"ocid1.natgateway..aaa"}, result.NativeIDs)
}

// Helpers

func newTestNatGatewayBody(lifecycleState string) string {
	return fmt.Sprintf(`{
		"id": "ocid1.natgateway..aaa",
		"compartmentId": "ocid1.compartment..xxx",
		"vcnId": "ocid1.vcn..aaa",
		"displayName": "test-natgw",
		"blockTraffic": false,
		"lifecycleState": %q
	}`, lifecycleState)
}
