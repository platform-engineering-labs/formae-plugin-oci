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

func TestSubnetRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/subnets/ocid1.subnet..aaa"}: {200, newTestSubnetBody("AVAILABLE")},
		})
		p := core.NewSubnetProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.subnet..aaa"})
		require.NoError(t, err)
		assert.Empty(t, result.ErrorCode)

		var props map[string]any
		require.NoError(t, json.Unmarshal([]byte(result.Properties), &props))
		assert.Equal(t, "10.0.1.0/24", props["CidrBlock"])
	})

	t.Run("not_found", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/subnets/ocid1.subnet..missing"}: {404, `{"code":"NotAuthorizedOrNotFound","message":"not found"}`},
		})
		p := core.NewSubnetProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.subnet..missing"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})

	t.Run("terminal_state", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/subnets/ocid1.subnet..aaa"}: {200, newTestSubnetBody("TERMINATED")},
		})
		p := core.NewSubnetProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.subnet..aaa"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})
}

func TestSubnetCreate(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"POST", "/20160918/subnets"}: {200, newTestSubnetBody("AVAILABLE")},
	})
	p := core.NewSubnetProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{
		"CompartmentId": "ocid1.compartment..xxx",
		"VcnId":         "ocid1.vcn..aaa",
		"CidrBlock":     "10.0.1.0/24",
		"DisplayName":   "test-subnet",
	})
	require.NoError(t, err)

	result, err := p.Create(context.Background(), &resource.CreateRequest{
		ResourceType: "OCI::Core::Subnet",
		Properties:   props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
	assert.Equal(t, "ocid1.subnet..aaa", result.ProgressResult.NativeID)
}

func TestSubnetUpdate(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/subnets/ocid1.subnet..aaa"}: {200, newTestSubnetBody("AVAILABLE")},
		{"PUT", "/20160918/subnets/ocid1.subnet..aaa"}: {200, newTestSubnetBody("AVAILABLE")},
	})
	p := core.NewSubnetProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{"DisplayName": "updated-subnet"})
	require.NoError(t, err)

	result, err := p.Update(context.Background(), &resource.UpdateRequest{
		NativeID:          "ocid1.subnet..aaa",
		ResourceType:      "OCI::Core::Subnet",
		DesiredProperties: props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
}

func TestSubnetDelete(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/subnets/ocid1.subnet..aaa"}:    {200, newTestSubnetBody("AVAILABLE")},
		{"DELETE", "/20160918/subnets/ocid1.subnet..aaa"}: {204, ""},
	})
	p := core.NewSubnetProvisionerWithSvc(svc)

	result, err := p.Delete(context.Background(), &resource.DeleteRequest{NativeID: "ocid1.subnet..aaa"})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
}

func TestSubnetList(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/subnets"}: {200, fmt.Sprintf(`[%s]`, newTestSubnetBody("AVAILABLE"))},
	})
	p := core.NewSubnetProvisionerWithSvc(svc)

	result, err := p.List(context.Background(), &resource.ListRequest{
		ResourceType:         "OCI::Core::Subnet",
		AdditionalProperties: map[string]string{"CompartmentId": "ocid1.compartment..xxx"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"ocid1.subnet..aaa"}, result.NativeIDs)
}

// Helpers

func newTestSubnetBody(lifecycleState string) string {
	return fmt.Sprintf(`{
		"id": "ocid1.subnet..aaa",
		"compartmentId": "ocid1.compartment..xxx",
		"vcnId": "ocid1.vcn..aaa",
		"cidrBlock": "10.0.1.0/24",
		"displayName": "test-subnet",
		"lifecycleState": %q
	}`, lifecycleState)
}
