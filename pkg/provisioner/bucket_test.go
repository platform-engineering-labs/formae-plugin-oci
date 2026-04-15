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

	ociobjectstorage "github.com/oracle/oci-go-sdk/v65/objectstorage"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner/objectstorage"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBucketRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newTestObjectStorageClient(t, map[route]canned{
			{"GET", "/n/testnamespace/b/test-bucket"}: {200, newTestBucketBody()},
		})
		p := objectstorage.NewBucketProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "test-bucket"})
		require.NoError(t, err)
		assert.Empty(t, result.ErrorCode)

		var props map[string]any
		require.NoError(t, json.Unmarshal([]byte(result.Properties), &props))
		assert.Equal(t, "test-bucket", props["Name"])
		assert.Equal(t, "testnamespace", props["Namespace"])
	})

	t.Run("not_found", func(t *testing.T) {
		svc := newTestObjectStorageClient(t, map[route]canned{
			{"GET", "/n/testnamespace/b/missing-bucket"}: {404, `{"code":"BucketNotFound","message":"not found"}`},
		})
		p := objectstorage.NewBucketProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "missing-bucket"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})
}

func TestBucketCreate(t *testing.T) {
	svc := newTestObjectStorageClient(t, map[route]canned{
		{"POST", "/n/testnamespace/b"}: {200, newTestBucketBody()},
	})
	p := objectstorage.NewBucketProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{
		"CompartmentId": "ocid1.compartment..xxx",
		"Name":          "test-bucket",
		"Namespace":     "testnamespace",
	})
	require.NoError(t, err)

	result, err := p.Create(context.Background(), &resource.CreateRequest{
		ResourceType: "OCI::ObjectStorage::Bucket",
		Properties:   props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
	assert.Equal(t, "test-bucket", result.ProgressResult.NativeID)
}

func TestBucketUpdate(t *testing.T) {
	svc := newTestObjectStorageClient(t, map[route]canned{
		{"GET", "/n/testnamespace/b/test-bucket"}:  {200, newTestBucketBody()},
		{"POST", "/n/testnamespace/b/test-bucket"}: {200, newTestBucketBody()},
	})
	p := objectstorage.NewBucketProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{"PublicAccessType": "ObjectRead"})
	require.NoError(t, err)

	result, err := p.Update(context.Background(), &resource.UpdateRequest{
		NativeID:          "test-bucket",
		ResourceType:      "OCI::ObjectStorage::Bucket",
		DesiredProperties: props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
	assert.Equal(t, "test-bucket", result.ProgressResult.NativeID)
}

func TestBucketDelete(t *testing.T) {
	svc := newTestObjectStorageClient(t, map[route]canned{
		{"GET", "/n/testnamespace/b/test-bucket"}:    {200, newTestBucketBody()},
		{"DELETE", "/n/testnamespace/b/test-bucket"}: {204, ""},
	})
	p := objectstorage.NewBucketProvisionerWithSvc(svc)

	result, err := p.Delete(context.Background(), &resource.DeleteRequest{NativeID: "test-bucket"})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
	assert.Equal(t, "test-bucket", result.ProgressResult.NativeID)
}

func TestBucketList(t *testing.T) {
	svc := newTestObjectStorageClient(t, map[route]canned{
		{"GET", "/n/testnamespace/b"}: {200, fmt.Sprintf(`[%s]`, newTestBucketBody())},
	})
	p := objectstorage.NewBucketProvisionerWithSvc(svc)

	result, err := p.List(context.Background(), &resource.ListRequest{
		ResourceType: "OCI::ObjectStorage::Bucket",
		AdditionalProperties: map[string]string{
			"CompartmentId": "ocid1.compartment..xxx",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"test-bucket"}, result.NativeIDs)
}

// Helpers

func newTestObjectStorageClient(t *testing.T, responses map[route]canned) *ociobjectstorage.ObjectStorageClient {
	t.Helper()
	// The ObjectStorage client needs a namespace to build URLs.
	// We add a GetNamespace route that returns "testnamespace".
	responses[route{"GET", "/n"}] = canned{200, `"testnamespace"`}
	host := newTestDispatcher(t, responses)
	c, err := ociobjectstorage.NewObjectStorageClientWithConfigurationProvider(fakeOCIConfigProvider(t))
	require.NoError(t, err)
	applyTestRetryPolicy(&c)
	c.Host = host
	return &c
}

func newTestBucketBody() string {
	return `{
		"name": "test-bucket",
		"compartmentId": "ocid1.compartment..xxx",
		"namespace": "testnamespace",
		"publicAccessType": "NoPublicAccess",
		"storageTier": "Standard"
	}`
}
