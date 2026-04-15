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

	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner/core"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVCNRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/vcns/ocid1.vcn..aaa"}: {200, newTestVCNBody("AVAILABLE")},
		})
		p := core.NewVCNProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.vcn..aaa"})
		require.NoError(t, err)
		assert.Empty(t, result.ErrorCode)

		var props map[string]any
		require.NoError(t, json.Unmarshal([]byte(result.Properties), &props))
		assert.Equal(t, "10.0.0.0/16", props["CidrBlock"])
	})

	t.Run("not_found", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/vcns/ocid1.vcn..missing"}: {404, `{"code":"NotAuthorizedOrNotFound","message":"not found"}`},
		})
		p := core.NewVCNProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.vcn..missing"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})

	t.Run("terminal_state", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/vcns/ocid1.vcn..aaa"}: {200, newTestVCNBody("TERMINATED")},
		})
		p := core.NewVCNProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.vcn..aaa"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})
}

func TestVCNCreate(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"POST", "/20160918/vcns"}: {200, newTestVCNBody("AVAILABLE")},
	})
	p := core.NewVCNProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{
		"CompartmentId": "ocid1.compartment..xxx",
		"CidrBlock":     "10.0.0.0/16",
		"DisplayName":   "test-vcn",
	})
	require.NoError(t, err)

	result, err := p.Create(context.Background(), &resource.CreateRequest{
		ResourceType: "OCI::Core::VCN",
		Properties:   props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
	assert.Equal(t, "ocid1.vcn..aaa", result.ProgressResult.NativeID)
}

func TestVCNUpdate(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"PUT", "/20160918/vcns/ocid1.vcn..aaa"}: {200, newTestVCNBody("AVAILABLE")},
	})
	p := core.NewVCNProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{"DisplayName": "updated-vcn"})
	require.NoError(t, err)

	result, err := p.Update(context.Background(), &resource.UpdateRequest{
		NativeID:          "ocid1.vcn..aaa",
		ResourceType:      "OCI::Core::VCN",
		DesiredProperties: props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
}

func TestVCNDelete(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/vcns/ocid1.vcn..aaa"}:    {200, newTestVCNBody("AVAILABLE")},
		{"DELETE", "/20160918/vcns/ocid1.vcn..aaa"}: {204, ""},
	})
	p := core.NewVCNProvisionerWithSvc(svc)

	result, err := p.Delete(context.Background(), &resource.DeleteRequest{NativeID: "ocid1.vcn..aaa"})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
}

func TestVCNList(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/vcns"}: {200, fmt.Sprintf(`[%s]`, newTestVCNBody("AVAILABLE"))},
	})
	p := core.NewVCNProvisionerWithSvc(svc)

	result, err := p.List(context.Background(), &resource.ListRequest{
		ResourceType:         "OCI::Core::VCN",
		AdditionalProperties: map[string]string{"CompartmentId": "ocid1.compartment..xxx"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"ocid1.vcn..aaa"}, result.NativeIDs)
}

// Helpers

func newTestVirtualNetworkClient(t *testing.T, responses map[route]canned) *ocicore.VirtualNetworkClient {
	t.Helper()
	host := newTestDispatcher(t, responses)
	c, err := ocicore.NewVirtualNetworkClientWithConfigurationProvider(fakeOCIConfigProvider(t))
	require.NoError(t, err)
	applyTestRetryPolicy(&c)
	c.Host = host
	return &c
}

func newTestVCNBody(lifecycleState string) string {
	return fmt.Sprintf(`{
		"id": "ocid1.vcn..aaa",
		"compartmentId": "ocid1.compartment..xxx",
		"cidrBlock": "10.0.0.0/16",
		"displayName": "test-vcn",
		"lifecycleState": %q
	}`, lifecycleState)
}
