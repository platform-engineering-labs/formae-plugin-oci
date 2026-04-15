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

func TestRouteTableRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/routeTables/ocid1.routetable..aaa"}: {200, newTestRouteTableBody("AVAILABLE")},
		})
		p := core.NewRouteTableProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.routetable..aaa"})
		require.NoError(t, err)
		assert.Empty(t, result.ErrorCode)

		var props map[string]any
		require.NoError(t, json.Unmarshal([]byte(result.Properties), &props))
		assert.Equal(t, "test-rt", props["DisplayName"])
	})

	t.Run("not_found", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/routeTables/ocid1.routetable..missing"}: {404, `{"code":"NotAuthorizedOrNotFound","message":"not found"}`},
		})
		p := core.NewRouteTableProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.routetable..missing"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})

	t.Run("terminal_state", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/routeTables/ocid1.routetable..aaa"}: {200, newTestRouteTableBody("TERMINATED")},
		})
		p := core.NewRouteTableProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.routetable..aaa"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})
}

func TestRouteTableCreate(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"POST", "/20160918/routeTables"}: {200, newTestRouteTableBody("AVAILABLE")},
	})
	p := core.NewRouteTableProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{
		"CompartmentId": "ocid1.compartment..xxx",
		"VcnId":         "ocid1.vcn..aaa",
		"DisplayName":   "test-rt",
		"RouteRules": []map[string]any{
			{
				"NetworkEntityId": "ocid1.internetgateway..aaa",
				"Destination":     "0.0.0.0/0",
				"DestinationType": "CIDR_BLOCK",
			},
		},
	})
	require.NoError(t, err)

	result, err := p.Create(context.Background(), &resource.CreateRequest{
		ResourceType: "OCI::Core::RouteTable",
		Properties:   props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
	assert.Equal(t, "ocid1.routetable..aaa", result.ProgressResult.NativeID)
}

func TestRouteTableUpdate(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/routeTables/ocid1.routetable..aaa"}: {200, newTestRouteTableBody("AVAILABLE")},
		{"PUT", "/20160918/routeTables/ocid1.routetable..aaa"}: {200, newTestRouteTableBody("AVAILABLE")},
	})
	p := core.NewRouteTableProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{"DisplayName": "updated-rt"})
	require.NoError(t, err)

	result, err := p.Update(context.Background(), &resource.UpdateRequest{
		NativeID:          "ocid1.routetable..aaa",
		ResourceType:      "OCI::Core::RouteTable",
		DesiredProperties: props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
}

func TestRouteTableDelete(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/routeTables/ocid1.routetable..aaa"}:    {200, newTestRouteTableBody("AVAILABLE")},
		{"DELETE", "/20160918/routeTables/ocid1.routetable..aaa"}: {204, ""},
	})
	p := core.NewRouteTableProvisionerWithSvc(svc)

	result, err := p.Delete(context.Background(), &resource.DeleteRequest{NativeID: "ocid1.routetable..aaa"})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
}

func TestRouteTableList(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/routeTables"}: {200, fmt.Sprintf(`[%s]`, newTestRouteTableBody("AVAILABLE"))},
	})
	p := core.NewRouteTableProvisionerWithSvc(svc)

	result, err := p.List(context.Background(), &resource.ListRequest{
		ResourceType:         "OCI::Core::RouteTable",
		AdditionalProperties: map[string]string{"CompartmentId": "ocid1.compartment..xxx"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"ocid1.routetable..aaa"}, result.NativeIDs)
}

// Helpers

func newTestRouteTableBody(lifecycleState string) string {
	return fmt.Sprintf(`{
		"id": "ocid1.routetable..aaa",
		"compartmentId": "ocid1.compartment..xxx",
		"vcnId": "ocid1.vcn..aaa",
		"displayName": "test-rt",
		"routeRules": [
			{
				"networkEntityId": "ocid1.internetgateway..aaa",
				"destination": "0.0.0.0/0",
				"destinationType": "CIDR_BLOCK"
			}
		],
		"lifecycleState": %q
	}`, lifecycleState)
}
