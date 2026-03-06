// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

//go:build integration

package provisioner_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner/core"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getTestAvailabilityDomain reads OCI_AVAILABILITY_DOMAIN from the environment.
// If not set, queries the Identity API for the first AD in the compartment.
func getTestAvailabilityDomain(t *testing.T) string {
	t.Helper()
	if ad := os.Getenv("OCI_AVAILABILITY_DOMAIN"); ad != "" {
		t.Logf("Using availability domain (env): %s", ad)
		return ad
	}

	clients := newTestClients(t)
	idClient, err := clients.GetIdentityClient()
	require.NoError(t, err)

	compartmentID := getTestCompartmentID(t)
	resp, err := idClient.ListAvailabilityDomains(context.Background(), identity.ListAvailabilityDomainsRequest{
		CompartmentId: common.String(compartmentID),
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Items, "no availability domains found")

	ad := *resp.Items[0].Name
	t.Logf("Using availability domain (API): %s", ad)
	return ad
}

// pollVolumeStatus polls the provisioner Status until success, failure, or timeout.
func pollVolumeStatus(t *testing.T, ctx context.Context, prov interface {
	Status(context.Context, *resource.StatusRequest) (*resource.StatusResult, error)
}, requestID string, timeout time.Duration) resource.OperationStatus {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		statusResult, err := prov.Status(ctx, &resource.StatusRequest{
			RequestID:    requestID,
			TargetConfig: newTestTargetConfig(),
		})
		require.NoError(t, err)
		require.NotNil(t, statusResult)

		status := statusResult.ProgressResult.OperationStatus
		if status == resource.OperationStatusSuccess || status == resource.OperationStatusFailure {
			return status
		}
		t.Logf("Volume status: %s (message: %s)", status, statusResult.ProgressResult.StatusMessage)
		time.Sleep(5 * time.Second)
	}
	t.Fatalf("Volume operation timed out after %s", timeout)
	return resource.OperationStatusFailure
}

// deleteVolumeAndWait deletes a volume via SDK and waits for termination.
func deleteVolumeAndWait(ctx context.Context, bsClient *ocicore.BlockstorageClient, volumeID string) {
	_, _ = bsClient.DeleteVolume(ctx, ocicore.DeleteVolumeRequest{
		VolumeId: common.String(volumeID),
	})
	// Best-effort wait for termination
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		resp, err := bsClient.GetVolume(ctx, ocicore.GetVolumeRequest{
			VolumeId: common.String(volumeID),
		})
		if err != nil {
			return // likely 404 — deleted
		}
		if resp.LifecycleState == ocicore.VolumeLifecycleStateTerminated {
			return
		}
		time.Sleep(5 * time.Second)
	}
}

func TestVolume_Create(t *testing.T) {
	ctx := context.Background()
	compartmentID := getTestCompartmentID(t)
	ad := getTestAvailabilityDomain(t)
	clients := newTestClients(t)
	prov := core.NewVolumeProvisioner(clients)

	displayName := fmt.Sprintf("formae-test-create-%d", time.Now().Unix())

	props := mustMarshalJSON(t, map[string]any{
		"CompartmentId":      compartmentID,
		"AvailabilityDomain": ad,
		"DisplayName":        displayName,
		"SizeInGBs":          float64(50),
	})

	req := &resource.CreateRequest{
		ResourceType: "OCI::Core::Volume",
		Properties:   props,
		TargetConfig: newTestTargetConfig(),
	}

	result, err := prov.Create(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.ProgressResult)
	// Volume create is async — expect InProgress
	assert.Equal(t, resource.OperationStatusInProgress, result.ProgressResult.OperationStatus)
	assert.NotEmpty(t, result.ProgressResult.NativeID)
	assert.NotEmpty(t, result.ProgressResult.RequestID)
	nativeID := result.ProgressResult.NativeID
	t.Logf("Created volume (async): %s", nativeID)

	// Cleanup via SDK
	t.Cleanup(func() {
		bsClient, _ := clients.GetBlockstorageClient()
		deleteVolumeAndWait(ctx, bsClient, nativeID)
	})

	// Poll until available
	status := pollVolumeStatus(t, ctx, prov, result.ProgressResult.RequestID, 3*time.Minute)
	assert.Equal(t, resource.OperationStatusSuccess, status)
}

func TestVolume_Read(t *testing.T) {
	ctx := context.Background()
	compartmentID := getTestCompartmentID(t)
	ad := getTestAvailabilityDomain(t)
	clients := newTestClients(t)
	prov := core.NewVolumeProvisioner(clients)

	// Create via SDK
	bsClient, err := clients.GetBlockstorageClient()
	require.NoError(t, err)

	displayName := fmt.Sprintf("formae-test-read-%d", time.Now().Unix())
	createResp, err := bsClient.CreateVolume(ctx, ocicore.CreateVolumeRequest{
		CreateVolumeDetails: ocicore.CreateVolumeDetails{
			CompartmentId:      common.String(compartmentID),
			AvailabilityDomain: common.String(ad),
			DisplayName:        common.String(displayName),
			SizeInGBs:          common.Int64(50),
		},
	})
	require.NoError(t, err)
	nativeID := *createResp.Id

	t.Cleanup(func() {
		deleteVolumeAndWait(ctx, bsClient, nativeID)
	})

	// Wait for volume to become available before reading
	status := pollVolumeStatus(t, ctx, prov, nativeID, 3*time.Minute)
	require.Equal(t, resource.OperationStatusSuccess, status)

	// Read via provisioner
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		TargetConfig: newTestTargetConfig(),
	})
	require.NoError(t, err)
	require.NotNil(t, readResult)
	assert.Equal(t, "OCI::Core::Volume", readResult.ResourceType)
	assert.Empty(t, readResult.ErrorCode)

	var props map[string]any
	err = json.Unmarshal([]byte(readResult.Properties), &props)
	require.NoError(t, err)
	assert.Equal(t, displayName, props["DisplayName"])
	assert.Equal(t, compartmentID, props["CompartmentId"])
}

func TestVolume_Update(t *testing.T) {
	ctx := context.Background()
	compartmentID := getTestCompartmentID(t)
	ad := getTestAvailabilityDomain(t)
	clients := newTestClients(t)
	prov := core.NewVolumeProvisioner(clients)

	// Create via SDK
	bsClient, err := clients.GetBlockstorageClient()
	require.NoError(t, err)

	displayName := fmt.Sprintf("formae-test-update-%d", time.Now().Unix())
	createResp, err := bsClient.CreateVolume(ctx, ocicore.CreateVolumeRequest{
		CreateVolumeDetails: ocicore.CreateVolumeDetails{
			CompartmentId:      common.String(compartmentID),
			AvailabilityDomain: common.String(ad),
			DisplayName:        common.String(displayName),
			SizeInGBs:          common.Int64(50),
		},
	})
	require.NoError(t, err)
	nativeID := *createResp.Id

	t.Cleanup(func() {
		deleteVolumeAndWait(ctx, bsClient, nativeID)
	})

	// Wait for volume to become available
	status := pollVolumeStatus(t, ctx, prov, nativeID, 3*time.Minute)
	require.Equal(t, resource.OperationStatusSuccess, status)

	// Update via provisioner
	newDisplayName := fmt.Sprintf("formae-test-updated-%d", time.Now().Unix())
	desiredProps := mustMarshalJSON(t, map[string]any{
		"CompartmentId":      compartmentID,
		"AvailabilityDomain": ad,
		"DisplayName":        newDisplayName,
		"SizeInGBs":          float64(50),
	})

	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          nativeID,
		ResourceType:      "OCI::Core::Volume",
		DesiredProperties: desiredProps,
		TargetConfig:      newTestTargetConfig(),
	})
	require.NoError(t, err)
	require.NotNil(t, updateResult)
	assert.Equal(t, resource.OperationStatusSuccess, updateResult.ProgressResult.OperationStatus)

	// Verify via SDK
	getResp, err := bsClient.GetVolume(ctx, ocicore.GetVolumeRequest{
		VolumeId: common.String(nativeID),
	})
	require.NoError(t, err)
	assert.Equal(t, newDisplayName, *getResp.DisplayName)
}

func TestVolume_Delete(t *testing.T) {
	ctx := context.Background()
	compartmentID := getTestCompartmentID(t)
	ad := getTestAvailabilityDomain(t)
	clients := newTestClients(t)
	prov := core.NewVolumeProvisioner(clients)

	// Create via SDK
	bsClient, err := clients.GetBlockstorageClient()
	require.NoError(t, err)

	displayName := fmt.Sprintf("formae-test-delete-%d", time.Now().Unix())
	createResp, err := bsClient.CreateVolume(ctx, ocicore.CreateVolumeRequest{
		CreateVolumeDetails: ocicore.CreateVolumeDetails{
			CompartmentId:      common.String(compartmentID),
			AvailabilityDomain: common.String(ad),
			DisplayName:        common.String(displayName),
			SizeInGBs:          common.Int64(50),
		},
	})
	require.NoError(t, err)
	nativeID := *createResp.Id

	// Safety cleanup
	t.Cleanup(func() {
		deleteVolumeAndWait(ctx, bsClient, nativeID)
	})

	// Wait for volume to become available
	status := pollVolumeStatus(t, ctx, prov, nativeID, 3*time.Minute)
	require.Equal(t, resource.OperationStatusSuccess, status)

	// Delete via provisioner (async)
	deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{
		NativeID:     nativeID,
		TargetConfig: newTestTargetConfig(),
	})
	require.NoError(t, err)
	require.NotNil(t, deleteResult)
	assert.Equal(t, resource.OperationStatusInProgress, deleteResult.ProgressResult.OperationStatus)

	// Poll until terminated
	deleteStatus := pollVolumeStatus(t, ctx, prov, nativeID, 3*time.Minute)
	assert.Equal(t, resource.OperationStatusSuccess, deleteStatus)
}

func TestVolume_List(t *testing.T) {
	ctx := context.Background()
	compartmentID := getTestCompartmentID(t)
	ad := getTestAvailabilityDomain(t)
	clients := newTestClients(t)
	prov := core.NewVolumeProvisioner(clients)

	// Create via SDK
	bsClient, err := clients.GetBlockstorageClient()
	require.NoError(t, err)

	displayName := fmt.Sprintf("formae-test-list-%d", time.Now().Unix())
	createResp, err := bsClient.CreateVolume(ctx, ocicore.CreateVolumeRequest{
		CreateVolumeDetails: ocicore.CreateVolumeDetails{
			CompartmentId:      common.String(compartmentID),
			AvailabilityDomain: common.String(ad),
			DisplayName:        common.String(displayName),
			SizeInGBs:          common.Int64(50),
		},
	})
	require.NoError(t, err)
	nativeID := *createResp.Id

	t.Cleanup(func() {
		deleteVolumeAndWait(ctx, bsClient, nativeID)
	})

	// Wait for volume to become available
	status := pollVolumeStatus(t, ctx, prov, nativeID, 3*time.Minute)
	require.Equal(t, resource.OperationStatusSuccess, status)

	// List via provisioner
	listResult, err := prov.List(ctx, &resource.ListRequest{
		ResourceType: "OCI::Core::Volume",
		TargetConfig: newTestTargetConfig(),
		AdditionalProperties: map[string]string{
			"CompartmentId": compartmentID,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, listResult)
	assert.NotEmpty(t, listResult.NativeIDs, "List should return at least one volume")

	found := false
	for _, id := range listResult.NativeIDs {
		if id == nativeID {
			found = true
			break
		}
	}
	assert.True(t, found, "Created volume %s should appear in list results", nativeID)
}
