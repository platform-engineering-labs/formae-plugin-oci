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
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner/core"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deleteVCNAndWait deletes a VCN and polls until it's gone.
func deleteVCNAndWait(ctx context.Context, vnClient *ocicore.VirtualNetworkClient, vcnID string) {
	_, _ = vnClient.DeleteVcn(ctx, ocicore.DeleteVcnRequest{
		VcnId: common.String(vcnID),
	})
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		_, err := vnClient.GetVcn(ctx, ocicore.GetVcnRequest{
			VcnId: common.String(vcnID),
		})
		if err != nil {
			return // 404 → gone
		}
		time.Sleep(2 * time.Second)
	}
}

// TestVCN exercises the full VCN provisioner lifecycle using a single VCN.
// OCI us-chicago-1 has a VCN limit of 2, so creating multiple VCNs across
// separate tests causes LimitExceeded errors due to delayed quota release.
func TestVCN(t *testing.T) {
	ctx := context.Background()
	compartmentID := getTestCompartmentID(t)
	clients := newTestClients(t)
	prov := core.NewVCNProvisioner(clients)

	vnClient, err := clients.GetVirtualNetworkClient()
	require.NoError(t, err)

	var nativeID string

	t.Run("Create", func(t *testing.T) {
		displayName := fmt.Sprintf("formae-test-create-%d", time.Now().Unix())

		props := mustMarshalJSON(t, map[string]any{
			"CompartmentId": compartmentID,
			"DisplayName":   displayName,
			"CidrBlocks":    []string{"10.0.0.0/16"},
		})

		result, err := prov.Create(ctx, &resource.CreateRequest{
			ResourceType: "OCI::Core::VCN",
			Properties:   props,
			TargetConfig: newTestTargetConfig(),
		})
		require.NoError(t, err)
		require.NotNil(t, result.ProgressResult)
		assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
		assert.NotEmpty(t, result.ProgressResult.NativeID)
		nativeID = result.ProgressResult.NativeID
		t.Logf("Created VCN: %s", nativeID)
	})

	if nativeID == "" {
		t.Fatal("Create subtest failed, cannot continue")
	}

	// Safety cleanup at the end of the parent test
	t.Cleanup(func() {
		deleteVCNAndWait(ctx, vnClient, nativeID)
	})

	t.Run("Read", func(t *testing.T) {
		readResult, err := prov.Read(ctx, &resource.ReadRequest{
			NativeID:     nativeID,
			TargetConfig: newTestTargetConfig(),
		})
		require.NoError(t, err)
		require.NotNil(t, readResult)
		assert.Equal(t, "OCI::Core::VCN", readResult.ResourceType)
		assert.Empty(t, readResult.ErrorCode)

		var props map[string]any
		err = json.Unmarshal([]byte(readResult.Properties), &props)
		require.NoError(t, err)
		assert.Equal(t, compartmentID, props["CompartmentId"])
	})

	t.Run("Update", func(t *testing.T) {
		newDisplayName := fmt.Sprintf("formae-test-updated-%d", time.Now().Unix())
		desiredProps := mustMarshalJSON(t, map[string]any{
			"CompartmentId": compartmentID,
			"DisplayName":   newDisplayName,
			"CidrBlocks":    []string{"10.0.0.0/16"},
		})

		updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
			NativeID:          nativeID,
			ResourceType:      "OCI::Core::VCN",
			DesiredProperties: desiredProps,
			TargetConfig:      newTestTargetConfig(),
		})
		require.NoError(t, err)
		require.NotNil(t, updateResult)
		assert.Equal(t, resource.OperationStatusSuccess, updateResult.ProgressResult.OperationStatus)

		// Verify via SDK
		getResp, err := vnClient.GetVcn(ctx, ocicore.GetVcnRequest{
			VcnId: common.String(nativeID),
		})
		require.NoError(t, err)
		assert.Equal(t, newDisplayName, *getResp.DisplayName)
	})

	t.Run("List", func(t *testing.T) {
		listResult, err := prov.List(ctx, &resource.ListRequest{
			ResourceType: "OCI::Core::VCN",
			TargetConfig: newTestTargetConfig(),
			AdditionalProperties: map[string]string{
				"CompartmentId": compartmentID,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, listResult)
		assert.NotEmpty(t, listResult.NativeIDs)

		found := false
		for _, id := range listResult.NativeIDs {
			if id == nativeID {
				found = true
				break
			}
		}
		assert.True(t, found, "Created VCN %s should appear in list results", nativeID)
	})

	t.Run("Delete", func(t *testing.T) {
		deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{
			NativeID:     nativeID,
			TargetConfig: newTestTargetConfig(),
		})
		require.NoError(t, err)
		require.NotNil(t, deleteResult)
		// Delete is async — returns InProgress, then poll Status until done
		assert.Equal(t, resource.OperationStatusInProgress, deleteResult.ProgressResult.OperationStatus)

		// Poll Status until delete completes
		require.Eventually(t, func() bool {
			statusResult, err := prov.Status(ctx, &resource.StatusRequest{
				RequestID:    nativeID,
				NativeID:     nativeID,
				TargetConfig: newTestTargetConfig(),
			})
			if err != nil {
				return false
			}
			return statusResult.ProgressResult.OperationStatus == resource.OperationStatusSuccess
		}, 3*time.Minute, 5*time.Second, "VCN should be fully deleted within 3 minutes")

		// Verify deleted
		readResult, err := prov.Read(ctx, &resource.ReadRequest{
			NativeID:     nativeID,
			TargetConfig: newTestTargetConfig(),
		})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, readResult.ErrorCode)
	})
}
