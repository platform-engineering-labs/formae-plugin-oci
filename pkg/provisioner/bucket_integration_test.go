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
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
	ociobjstorage "github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner/objectstorage"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getNamespace fetches the Object Storage namespace for the tenancy.
func getNamespace(t *testing.T, ctx context.Context) string {
	t.Helper()
	clients := newTestClients(t)
	osClient, err := clients.GetObjectStorageClient()
	require.NoError(t, err)
	resp, err := osClient.GetNamespace(ctx, objectstorage.GetNamespaceRequest{})
	require.NoError(t, err)
	return *resp.Value
}

func TestBucket_Create(t *testing.T) {
	ctx := context.Background()
	compartmentID := getTestCompartmentID(t)
	clients := newTestClients(t)
	prov := ociobjstorage.NewBucketProvisioner(clients)
	namespace := getNamespace(t, ctx)

	bucketName := fmt.Sprintf("formae-test-create-%d", time.Now().Unix())

	props := mustMarshalJSON(t, map[string]any{
		"CompartmentId": compartmentID,
		"Name":          bucketName,
		"Namespace":     namespace,
	})

	req := &resource.CreateRequest{
		ResourceType: "OCI::ObjectStorage::Bucket",
		Properties:   props,
		TargetConfig: newTestTargetConfig(),
	}

	result, err := prov.Create(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.ProgressResult)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
	assert.Equal(t, bucketName, result.ProgressResult.NativeID)
	t.Logf("Created bucket: %s", result.ProgressResult.NativeID)

	// Cleanup via SDK
	t.Cleanup(func() {
		osClient, _ := clients.GetObjectStorageClient()
		_, _ = osClient.DeleteBucket(ctx, objectstorage.DeleteBucketRequest{
			NamespaceName: common.String(namespace),
			BucketName:    common.String(bucketName),
		})
	})
}

func TestBucket_Read(t *testing.T) {
	ctx := context.Background()
	compartmentID := getTestCompartmentID(t)
	clients := newTestClients(t)
	prov := ociobjstorage.NewBucketProvisioner(clients)
	namespace := getNamespace(t, ctx)

	// Create via SDK
	osClient, err := clients.GetObjectStorageClient()
	require.NoError(t, err)

	bucketName := fmt.Sprintf("formae-test-read-%d", time.Now().Unix())
	_, err = osClient.CreateBucket(ctx, objectstorage.CreateBucketRequest{
		NamespaceName: common.String(namespace),
		CreateBucketDetails: objectstorage.CreateBucketDetails{
			CompartmentId: common.String(compartmentID),
			Name:          common.String(bucketName),
		},
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = osClient.DeleteBucket(ctx, objectstorage.DeleteBucketRequest{
			NamespaceName: common.String(namespace),
			BucketName:    common.String(bucketName),
		})
	})

	// Read via provisioner
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     bucketName,
		TargetConfig: newTestTargetConfig(),
	})
	require.NoError(t, err)
	require.NotNil(t, readResult)
	assert.Equal(t, "OCI::ObjectStorage::Bucket", readResult.ResourceType)
	assert.Empty(t, readResult.ErrorCode)

	var props map[string]any
	err = json.Unmarshal([]byte(readResult.Properties), &props)
	require.NoError(t, err)
	assert.Equal(t, bucketName, props["Name"])
	assert.Equal(t, compartmentID, props["CompartmentId"])
	assert.Equal(t, namespace, props["Namespace"])
}

func TestBucket_Update(t *testing.T) {
	ctx := context.Background()
	compartmentID := getTestCompartmentID(t)
	clients := newTestClients(t)
	prov := ociobjstorage.NewBucketProvisioner(clients)
	namespace := getNamespace(t, ctx)

	// Create via SDK
	osClient, err := clients.GetObjectStorageClient()
	require.NoError(t, err)

	bucketName := fmt.Sprintf("formae-test-update-%d", time.Now().Unix())
	_, err = osClient.CreateBucket(ctx, objectstorage.CreateBucketRequest{
		NamespaceName: common.String(namespace),
		CreateBucketDetails: objectstorage.CreateBucketDetails{
			CompartmentId: common.String(compartmentID),
			Name:          common.String(bucketName),
		},
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = osClient.DeleteBucket(ctx, objectstorage.DeleteBucketRequest{
			NamespaceName: common.String(namespace),
			BucketName:    common.String(bucketName),
		})
	})

	// Update via provisioner — enable object events
	desiredProps := mustMarshalJSON(t, map[string]any{
		"CompartmentId":      compartmentID,
		"Name":               bucketName,
		"Namespace":          namespace,
		"ObjectEventsEnabled": true,
	})

	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          bucketName,
		ResourceType:      "OCI::ObjectStorage::Bucket",
		DesiredProperties: desiredProps,
		TargetConfig:      newTestTargetConfig(),
	})
	require.NoError(t, err)
	require.NotNil(t, updateResult)
	assert.Equal(t, resource.OperationStatusSuccess, updateResult.ProgressResult.OperationStatus)

	// Verify via SDK
	getResp, err := osClient.GetBucket(ctx, objectstorage.GetBucketRequest{
		NamespaceName: common.String(namespace),
		BucketName:    common.String(bucketName),
	})
	require.NoError(t, err)
	assert.True(t, *getResp.ObjectEventsEnabled)
}

func TestBucket_Delete(t *testing.T) {
	ctx := context.Background()
	compartmentID := getTestCompartmentID(t)
	clients := newTestClients(t)
	prov := ociobjstorage.NewBucketProvisioner(clients)
	namespace := getNamespace(t, ctx)

	// Create via SDK
	osClient, err := clients.GetObjectStorageClient()
	require.NoError(t, err)

	bucketName := fmt.Sprintf("formae-test-delete-%d", time.Now().Unix())
	_, err = osClient.CreateBucket(ctx, objectstorage.CreateBucketRequest{
		NamespaceName: common.String(namespace),
		CreateBucketDetails: objectstorage.CreateBucketDetails{
			CompartmentId: common.String(compartmentID),
			Name:          common.String(bucketName),
		},
	})
	require.NoError(t, err)

	// Safety cleanup
	t.Cleanup(func() {
		_, _ = osClient.DeleteBucket(ctx, objectstorage.DeleteBucketRequest{
			NamespaceName: common.String(namespace),
			BucketName:    common.String(bucketName),
		})
	})

	// Delete via provisioner
	deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{
		NativeID:     bucketName,
		TargetConfig: newTestTargetConfig(),
	})
	require.NoError(t, err)
	require.NotNil(t, deleteResult)
	assert.Equal(t, resource.OperationStatusSuccess, deleteResult.ProgressResult.OperationStatus)

	// Verify deleted
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     bucketName,
		TargetConfig: newTestTargetConfig(),
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationErrorCodeNotFound, readResult.ErrorCode)
}

func TestBucket_List(t *testing.T) {
	ctx := context.Background()
	compartmentID := getTestCompartmentID(t)
	clients := newTestClients(t)
	prov := ociobjstorage.NewBucketProvisioner(clients)
	namespace := getNamespace(t, ctx)

	// Create via SDK
	osClient, err := clients.GetObjectStorageClient()
	require.NoError(t, err)

	bucketName := fmt.Sprintf("formae-test-list-%d", time.Now().Unix())
	_, err = osClient.CreateBucket(ctx, objectstorage.CreateBucketRequest{
		NamespaceName: common.String(namespace),
		CreateBucketDetails: objectstorage.CreateBucketDetails{
			CompartmentId: common.String(compartmentID),
			Name:          common.String(bucketName),
		},
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = osClient.DeleteBucket(ctx, objectstorage.DeleteBucketRequest{
			NamespaceName: common.String(namespace),
			BucketName:    common.String(bucketName),
		})
	})

	// List via provisioner
	listResult, err := prov.List(ctx, &resource.ListRequest{
		ResourceType: "OCI::ObjectStorage::Bucket",
		TargetConfig: newTestTargetConfig(),
		AdditionalProperties: map[string]string{
			"CompartmentId": compartmentID,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, listResult)
	assert.NotEmpty(t, listResult.NativeIDs, "List should return at least one bucket")

	found := false
	for _, id := range listResult.NativeIDs {
		if id == bucketName {
			found = true
			break
		}
	}
	assert.True(t, found, "Created bucket %s should appear in list results", bucketName)
}
