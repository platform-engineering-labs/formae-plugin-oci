// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package objectstorage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/client"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/util"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

type BucketProvisioner struct {
	clients *client.Clients
}

var _ provisioner.Provisioner = &BucketProvisioner{}

func init() {
	provisioner.Register("OCI::ObjectStorage::Bucket", NewBucketProvisioner)
}

func NewBucketProvisioner(clients *client.Clients) provisioner.Provisioner {
	return &BucketProvisioner{clients: clients}
}

// getNamespace fetches the Object Storage namespace for the tenancy.
// If namespace is provided in props, it returns that; otherwise fetches dynamically.
func (p *BucketProvisioner) getNamespace(ctx context.Context, client *objectstorage.ObjectStorageClient, props map[string]any) (string, error) {
	if ns, ok := util.ExtractString(props, "Namespace"); ok && ns != "" {
		return ns, nil
	}
	resp, err := client.GetNamespace(ctx, objectstorage.GetNamespaceRequest{})
	if err != nil {
		return "", fmt.Errorf("failed to get Object Storage namespace: %w", err)
	}
	return *resp.Value, nil
}

func (p *BucketProvisioner) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	client, err := p.clients.GetObjectStorageClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ObjectStorage client: %w", err)
	}

	var props map[string]any
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	namespace, err := p.getNamespace(ctx, client, props)
	if err != nil {
		return nil, err
	}

	createDetails := objectstorage.CreateBucketDetails{
		CompartmentId: common.String(props["CompartmentId"].(string)),
		Name:          common.String(props["Name"].(string)),
	}

	if publicAccessType, ok := util.ExtractString(props, "PublicAccessType"); ok {
		createDetails.PublicAccessType = objectstorage.CreateBucketDetailsPublicAccessTypeEnum(publicAccessType)
	}
	if storageTier, ok := util.ExtractString(props, "StorageTier"); ok {
		createDetails.StorageTier = objectstorage.CreateBucketDetailsStorageTierEnum(storageTier)
	}
	if objectEventsEnabled, ok := util.ExtractBool(props, "ObjectEventsEnabled"); ok {
		createDetails.ObjectEventsEnabled = common.Bool(objectEventsEnabled)
	}
	if versioning, ok := util.ExtractString(props, "Versioning"); ok {
		createDetails.Versioning = objectstorage.CreateBucketDetailsVersioningEnum(versioning)
	}
	if freeformTags, ok := util.ExtractFreeformTags(props, "FreeformTags"); ok {
		createDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractDefinedTags(props, "DefinedTags"); ok {
		createDetails.DefinedTags = definedTags
	}

	createReq := objectstorage.CreateBucketRequest{
		NamespaceName:       common.String(namespace),
		CreateBucketDetails: createDetails,
	}

	resp, err := client.CreateBucket(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create Bucket: %w", err)
	}

	// Build properties from Create response
	properties := map[string]any{
		"CompartmentId": *resp.CompartmentId,
		"Name":          *resp.Name,
		"Namespace":     *resp.Namespace,
	}

	if resp.PublicAccessType != "" {
		properties["PublicAccessType"] = string(resp.PublicAccessType)
	}
	if resp.StorageTier != "" {
		properties["StorageTier"] = string(resp.StorageTier)
	}
	if resp.ObjectEventsEnabled != nil {
		properties["ObjectEventsEnabled"] = *resp.ObjectEventsEnabled
	}
	if resp.Versioning != "" {
		properties["Versioning"] = string(resp.Versioning)
	}
	if resp.FreeformTags != nil {
		properties["FreeformTags"] = util.FreeformTagsToList(resp.FreeformTags)
	}
	if resp.DefinedTags != nil {
		properties["DefinedTags"] = util.DefinedTagsToList(resp.DefinedTags)
	}

	propertiesBytes, err := json.Marshal(properties)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal properties: %w", err)
	}

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCreate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           *resp.Name,
			ResourceProperties: json.RawMessage(propertiesBytes),
		},
	}, nil
}

func (p *BucketProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	client, err := p.clients.GetObjectStorageClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ObjectStorage client: %w", err)
	}

	props, err := util.ApplyPatchDocument(ctx, request, p.Read)
	if err != nil {
		return nil, err
	}

	namespace, err := p.getNamespace(ctx, client, props)
	if err != nil {
		return nil, err
	}

	updateDetails := objectstorage.UpdateBucketDetails{}

	if publicAccessType, ok := util.ExtractString(props, "PublicAccessType"); ok {
		updateDetails.PublicAccessType = objectstorage.UpdateBucketDetailsPublicAccessTypeEnum(publicAccessType)
	}

	if versioning, ok := util.ExtractString(props, "Versioning"); ok {
		// OCI returns "Disabled" when versioning is disabled, but the API only accepts "Enabled" or "Suspended"
		// A merge can result in "Disabled" being returned, so we need to handle it.
		if versioning != "Disabled" {
			updateDetails.Versioning = objectstorage.UpdateBucketDetailsVersioningEnum(versioning)
		}
	}

	if objectEventsEnabled, ok := util.ExtractBool(props, "ObjectEventsEnabled"); ok {
		updateDetails.ObjectEventsEnabled = common.Bool(objectEventsEnabled)
	}

	if freeformTags, ok := util.ExtractFreeformTags(props, "FreeformTags"); ok {
		updateDetails.FreeformTags = freeformTags
	}

	if definedTags, ok := util.ExtractDefinedTags(props, "DefinedTags"); ok {
		updateDetails.DefinedTags = definedTags
	}

	updateReq := objectstorage.UpdateBucketRequest{
		NamespaceName:       common.String(namespace),
		BucketName:          common.String(request.NativeID),
		UpdateBucketDetails: updateDetails,
	}

	resp, err := client.UpdateBucket(ctx, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update Bucket: %w", err)
	}

	// Build properties from Update response (same as Create and Read)
	// This ensures formae stores the complete state, preventing drift detection issues
	properties := map[string]any{
		"CompartmentId": *resp.CompartmentId,
		"Name":          *resp.Name,
		"Namespace":     *resp.Namespace,
	}

	if resp.PublicAccessType != "" {
		properties["PublicAccessType"] = string(resp.PublicAccessType)
	}
	if resp.StorageTier != "" {
		properties["StorageTier"] = string(resp.StorageTier)
	}
	if resp.ObjectEventsEnabled != nil {
		properties["ObjectEventsEnabled"] = *resp.ObjectEventsEnabled
	}
	if resp.Versioning != "" {
		properties["Versioning"] = string(resp.Versioning)
	}
	if resp.FreeformTags != nil {
		properties["FreeformTags"] = util.FreeformTagsToList(resp.FreeformTags)
	}
	if resp.DefinedTags != nil {
		properties["DefinedTags"] = util.DefinedTagsToList(resp.DefinedTags)
	}

	propertiesBytes, err := json.Marshal(properties)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Bucket properties: %w", err)
	}

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationUpdate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           *resp.Name,
			ResourceProperties: json.RawMessage(propertiesBytes),
		},
	}, nil
}

func (p *BucketProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	client, err := p.clients.GetObjectStorageClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ObjectStorage client: %w", err)
	}

	namespace, err := p.getNamespace(ctx, client, nil)
	if err != nil {
		return nil, err
	}

	// Check if exists
	readReq := &resource.ReadRequest{
		NativeID: request.NativeID,
	}
	readRes, err := p.Read(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to read Bucket before delete: %w", err)
	}
	if readRes.ErrorCode == resource.OperationErrorCodeNotFound {
		return &resource.DeleteResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationDelete,
				OperationStatus: resource.OperationStatusSuccess,
				NativeID:        request.NativeID,
			},
		}, nil
	}

	deleteReq := objectstorage.DeleteBucketRequest{
		NamespaceName: common.String(namespace),
		BucketName:    common.String(request.NativeID),
	}

	_, err = client.DeleteBucket(ctx, deleteReq)
	if err != nil {
		return nil, fmt.Errorf("failed to delete Bucket: %w", err)
	}

	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationDelete,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (p *BucketProvisioner) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			RequestID:       request.RequestID,
		},
	}, nil
}

func (p *BucketProvisioner) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	client, err := p.clients.GetObjectStorageClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ObjectStorage client: %w", err)
	}

	namespace, err := p.getNamespace(ctx, client, nil)
	if err != nil {
		return nil, err
	}

	getReq := objectstorage.GetBucketRequest{
		NamespaceName: common.String(namespace),
		BucketName:    common.String(request.NativeID),
	}

	resp, err := client.GetBucket(ctx, getReq)
	if err != nil {
		if serviceErr, ok := common.IsServiceError(err); ok && serviceErr.GetHTTPStatusCode() == 404 {
			return &resource.ReadResult{
				ResourceType: "OCI::ObjectStorage::Bucket",
				ErrorCode:    resource.OperationErrorCodeNotFound,
			}, nil
		}
		return nil, fmt.Errorf("failed to read Bucket: %w", err)
	}

	props := map[string]any{
		"CompartmentId": *resp.CompartmentId,
		"Name":          *resp.Name,
		"Namespace":     *resp.Namespace,
	}

	if resp.PublicAccessType != "" {
		props["PublicAccessType"] = string(resp.PublicAccessType)
	}
	if resp.StorageTier != "" {
		props["StorageTier"] = string(resp.StorageTier)
	}
	if resp.ObjectEventsEnabled != nil {
		props["ObjectEventsEnabled"] = *resp.ObjectEventsEnabled
	}
	if resp.Versioning != "" {
		props["Versioning"] = string(resp.Versioning)
	}
	if resp.FreeformTags != nil {
		props["FreeformTags"] = util.FreeformTagsToList(resp.FreeformTags)
	}
	if resp.DefinedTags != nil {
		props["DefinedTags"] = util.DefinedTagsToList(resp.DefinedTags)
	}

	propBytes, err := json.Marshal(props)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Bucket properties: %w", err)
	}

	return &resource.ReadResult{
		ResourceType: "OCI::ObjectStorage::Bucket",
		Properties:   string(propBytes),
	}, nil
}

func (p *BucketProvisioner) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	client, err := p.clients.GetObjectStorageClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ObjectStorage client: %w", err)
	}

	compartmentId, ok := request.AdditionalProperties["CompartmentId"]
	if !ok {
		return nil, fmt.Errorf("CompartmentId is required for listing Buckets")
	}

	namespace, err := p.getNamespace(ctx, client, nil)
	if err != nil {
		return nil, err
	}

	listReq := objectstorage.ListBucketsRequest{
		NamespaceName: common.String(namespace),
		CompartmentId: common.String(compartmentId),
	}

	resp, err := client.ListBuckets(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list Buckets: %w", err)
	}

	nativeIDs := make([]string, 0, len(resp.Items))
	for _, bucket := range resp.Items {
		nativeIDs = append(nativeIDs, *bucket.Name)
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}
