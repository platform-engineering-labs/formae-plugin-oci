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

func TestVolumeRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newTestBlockstorageClient(t, map[route]canned{
			{"GET", "/20160918/volumes/ocid1.volume..aaa"}: {200, newTestVolumeBody("AVAILABLE")},
		})
		p := core.NewVolumeProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.volume..aaa"})
		require.NoError(t, err)
		assert.Empty(t, result.ErrorCode)

		var props map[string]any
		require.NoError(t, json.Unmarshal([]byte(result.Properties), &props))
		assert.Equal(t, "test-volume", props["DisplayName"])
	})

	t.Run("not_found", func(t *testing.T) {
		svc := newTestBlockstorageClient(t, map[route]canned{
			{"GET", "/20160918/volumes/ocid1.volume..missing"}: {404, `{"code":"NotAuthorizedOrNotFound","message":"not found"}`},
		})
		p := core.NewVolumeProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.volume..missing"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})

	t.Run("terminal_state", func(t *testing.T) {
		svc := newTestBlockstorageClient(t, map[route]canned{
			{"GET", "/20160918/volumes/ocid1.volume..aaa"}: {200, newTestVolumeBody("TERMINATED")},
		})
		p := core.NewVolumeProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.volume..aaa"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})
}

func TestVolumeCreate(t *testing.T) {
	svc := newTestBlockstorageClient(t, map[route]canned{
		{"POST", "/20160918/volumes"}: {200, newTestVolumeBody("PROVISIONING")},
	})
	p := core.NewVolumeProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{
		"CompartmentId":      "ocid1.compartment..xxx",
		"AvailabilityDomain": "US-CHICAGO-1-AD-1",
		"DisplayName":        "test-volume",
		"SizeInGBs":          50,
	})
	require.NoError(t, err)

	result, err := p.Create(context.Background(), &resource.CreateRequest{
		ResourceType: "OCI::Core::Volume",
		Properties:   props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusInProgress, result.ProgressResult.OperationStatus)
	assert.Equal(t, "ocid1.volume..aaa", result.ProgressResult.NativeID)
	assert.Equal(t, "ocid1.volume..aaa", result.ProgressResult.RequestID)
}

func TestVolumeUpdate(t *testing.T) {
	svc := newTestBlockstorageClient(t, map[route]canned{
		{"GET", "/20160918/volumes/ocid1.volume..aaa"}: {200, newTestVolumeBody("AVAILABLE")},
		{"PUT", "/20160918/volumes/ocid1.volume..aaa"}: {200, newTestVolumeBody("AVAILABLE")},
	})
	p := core.NewVolumeProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{"DisplayName": "updated-volume"})
	require.NoError(t, err)

	result, err := p.Update(context.Background(), &resource.UpdateRequest{
		NativeID:          "ocid1.volume..aaa",
		ResourceType:      "OCI::Core::Volume",
		DesiredProperties: props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
}

func TestVolumeDelete(t *testing.T) {
	svc := newTestBlockstorageClient(t, map[route]canned{
		{"GET", "/20160918/volumes/ocid1.volume..aaa"}:    {200, newTestVolumeBody("AVAILABLE")},
		{"DELETE", "/20160918/volumes/ocid1.volume..aaa"}: {204, ""},
	})
	p := core.NewVolumeProvisionerWithSvc(svc)

	result, err := p.Delete(context.Background(), &resource.DeleteRequest{NativeID: "ocid1.volume..aaa"})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusInProgress, result.ProgressResult.OperationStatus)
	assert.Equal(t, "ocid1.volume..aaa", result.ProgressResult.NativeID)
	assert.Equal(t, "ocid1.volume..aaa", result.ProgressResult.RequestID)
}

func TestVolumeList(t *testing.T) {
	svc := newTestBlockstorageClient(t, map[route]canned{
		{"GET", "/20160918/volumes"}: {200, fmt.Sprintf(`[%s]`, newTestVolumeBody("AVAILABLE"))},
	})
	p := core.NewVolumeProvisionerWithSvc(svc)

	result, err := p.List(context.Background(), &resource.ListRequest{
		ResourceType:         "OCI::Core::Volume",
		AdditionalProperties: map[string]string{"CompartmentId": "ocid1.compartment..xxx"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"ocid1.volume..aaa"}, result.NativeIDs)
}

// Helpers

func newTestBlockstorageClient(t *testing.T, responses map[route]canned) *ocicore.BlockstorageClient {
	t.Helper()
	host := newTestDispatcher(t, responses)
	c, err := ocicore.NewBlockstorageClientWithConfigurationProvider(fakeOCIConfigProvider(t))
	require.NoError(t, err)
	applyTestRetryPolicy(&c)
	c.Host = host
	return &c
}

func newTestVolumeBody(lifecycleState string) string {
	return fmt.Sprintf(`{
		"id": "ocid1.volume..aaa",
		"compartmentId": "ocid1.compartment..xxx",
		"availabilityDomain": "US-CHICAGO-1-AD-1",
		"displayName": "test-volume",
		"sizeInGBs": 50,
		"lifecycleState": %q
	}`, lifecycleState)
}
