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

func TestInternetGatewayRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/internetGateways/ocid1.internetgateway..aaa"}: {200, newTestInternetGatewayBody("AVAILABLE")},
		})
		p := core.NewInternetGatewayProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.internetgateway..aaa"})
		require.NoError(t, err)
		assert.Empty(t, result.ErrorCode)

		var props map[string]any
		require.NoError(t, json.Unmarshal([]byte(result.Properties), &props))
		assert.Equal(t, true, props["IsEnabled"])
	})

	t.Run("not_found", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/internetGateways/ocid1.internetgateway..missing"}: {404, `{"code":"NotAuthorizedOrNotFound","message":"not found"}`},
		})
		p := core.NewInternetGatewayProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.internetgateway..missing"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})

	t.Run("terminal_state", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/internetGateways/ocid1.internetgateway..aaa"}: {200, newTestInternetGatewayBody("TERMINATED")},
		})
		p := core.NewInternetGatewayProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.internetgateway..aaa"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})
}

func TestInternetGatewayCreate(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"POST", "/20160918/internetGateways"}: {200, newTestInternetGatewayBody("AVAILABLE")},
	})
	p := core.NewInternetGatewayProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{
		"CompartmentId": "ocid1.compartment..xxx",
		"VcnId":         "ocid1.vcn..aaa",
		"IsEnabled":     true,
		"DisplayName":   "test-igw",
	})
	require.NoError(t, err)

	result, err := p.Create(context.Background(), &resource.CreateRequest{
		ResourceType: "OCI::Core::InternetGateway",
		Properties:   props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
	assert.Equal(t, "ocid1.internetgateway..aaa", result.ProgressResult.NativeID)
}

func TestInternetGatewayUpdate(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/internetGateways/ocid1.internetgateway..aaa"}: {200, newTestInternetGatewayBody("AVAILABLE")},
		{"PUT", "/20160918/internetGateways/ocid1.internetgateway..aaa"}: {200, newTestInternetGatewayBody("AVAILABLE")},
	})
	p := core.NewInternetGatewayProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{"DisplayName": "updated-igw"})
	require.NoError(t, err)

	result, err := p.Update(context.Background(), &resource.UpdateRequest{
		NativeID:          "ocid1.internetgateway..aaa",
		ResourceType:      "OCI::Core::InternetGateway",
		DesiredProperties: props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
}

func TestInternetGatewayDelete(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/internetGateways/ocid1.internetgateway..aaa"}:    {200, newTestInternetGatewayBody("AVAILABLE")},
		{"DELETE", "/20160918/internetGateways/ocid1.internetgateway..aaa"}: {204, ""},
	})
	p := core.NewInternetGatewayProvisionerWithSvc(svc)

	result, err := p.Delete(context.Background(), &resource.DeleteRequest{NativeID: "ocid1.internetgateway..aaa"})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
}

func TestInternetGatewayList(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/internetGateways"}: {200, fmt.Sprintf(`[%s]`, newTestInternetGatewayBody("AVAILABLE"))},
	})
	p := core.NewInternetGatewayProvisionerWithSvc(svc)

	result, err := p.List(context.Background(), &resource.ListRequest{
		ResourceType:         "OCI::Core::InternetGateway",
		AdditionalProperties: map[string]string{"CompartmentId": "ocid1.compartment..xxx"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"ocid1.internetgateway..aaa"}, result.NativeIDs)
}

// Helpers

func newTestInternetGatewayBody(lifecycleState string) string {
	return fmt.Sprintf(`{
		"id": "ocid1.internetgateway..aaa",
		"compartmentId": "ocid1.compartment..xxx",
		"vcnId": "ocid1.vcn..aaa",
		"isEnabled": true,
		"displayName": "test-igw",
		"lifecycleState": %q
	}`, lifecycleState)
}
