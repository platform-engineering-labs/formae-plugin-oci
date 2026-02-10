// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package core

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/client"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/util"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

type VolumeProvisioner struct {
	clients *client.Clients
}

var _ provisioner.Provisioner = &VolumeProvisioner{}

func init() {
	provisioner.Register("OCI::Core::Volume", NewVolumeProvisioner)
}

func NewVolumeProvisioner(clients *client.Clients) provisioner.Provisioner {
	return &VolumeProvisioner{clients: clients}
}

func (p *VolumeProvisioner) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	svc, err := p.clients.GetBlockstorageClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Blockstorage client: %w", err)
	}

	var props map[string]any
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	createDetails := core.CreateVolumeDetails{
		CompartmentId:      common.String(props["CompartmentId"].(string)),
		AvailabilityDomain: common.String(props["AvailabilityDomain"].(string)),
	}

	if displayName, ok := util.ExtractString(props, "DisplayName"); ok {
		createDetails.DisplayName = common.String(displayName)
	}
	if sizeInGBs, ok := extractInt64Field(props, "SizeInGBs"); ok {
		createDetails.SizeInGBs = common.Int64(sizeInGBs)
	}
	if vpusPerGB, ok := extractInt64Field(props, "VpusPerGB"); ok {
		createDetails.VpusPerGB = common.Int64(vpusPerGB)
	}
	if isAutoTuneEnabled, ok := util.ExtractBool(props, "IsAutoTuneEnabled"); ok {
		createDetails.IsAutoTuneEnabled = common.Bool(isAutoTuneEnabled)
	}
	if kmsKeyId, ok := util.ExtractString(props, "KmsKeyId"); ok {
		createDetails.KmsKeyId = common.String(kmsKeyId)
	}
	if freeformTags, ok := util.ExtractFreeformTags(props, "FreeformTags"); ok {
		createDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractDefinedTags(props, "DefinedTags"); ok {
		createDetails.DefinedTags = definedTags
	}

	createReq := core.CreateVolumeRequest{
		CreateVolumeDetails: createDetails,
	}

	resp, err := svc.CreateVolume(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create Volume: %w", err)
	}

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCreate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        *resp.Id,
		},
	}, nil
}

func (p *VolumeProvisioner) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	svc, err := p.clients.GetBlockstorageClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Blockstorage client: %w", err)
	}

	getReq := core.GetVolumeRequest{
		VolumeId: common.String(request.NativeID),
	}

	resp, err := svc.GetVolume(ctx, getReq)
	if err != nil {
		if serviceErr, ok := common.IsServiceError(err); ok && serviceErr.GetHTTPStatusCode() == 404 {
			return &resource.ReadResult{
				ResourceType: "OCI::Core::Volume",
				ErrorCode:    resource.OperationErrorCodeNotFound,
			}, nil
		}
		return nil, fmt.Errorf("failed to read Volume: %w", err)
	}

	properties := buildVolumeProperties(resp.Volume)

	propBytes, err := json.Marshal(properties)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Volume properties: %w", err)
	}

	return &resource.ReadResult{
		ResourceType: "OCI::Core::Volume",
		Properties:   string(propBytes),
	}, nil
}

func (p *VolumeProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	svc, err := p.clients.GetBlockstorageClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Blockstorage client: %w", err)
	}

	props, err := util.ApplyPatchDocument(ctx, request, p.Read)
	if err != nil {
		return nil, err
	}

	updateDetails := core.UpdateVolumeDetails{}

	if displayName, ok := util.ExtractString(props, "DisplayName"); ok {
		updateDetails.DisplayName = common.String(displayName)
	}
	if sizeInGBs, ok := extractInt64Field(props, "SizeInGBs"); ok {
		updateDetails.SizeInGBs = common.Int64(sizeInGBs)
	}
	if vpusPerGB, ok := extractInt64Field(props, "VpusPerGB"); ok {
		updateDetails.VpusPerGB = common.Int64(vpusPerGB)
	}
	if isAutoTuneEnabled, ok := util.ExtractBool(props, "IsAutoTuneEnabled"); ok {
		updateDetails.IsAutoTuneEnabled = common.Bool(isAutoTuneEnabled)
	}
	if freeformTags, ok := util.ExtractFreeformTags(props, "FreeformTags"); ok {
		updateDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractDefinedTags(props, "DefinedTags"); ok {
		updateDetails.DefinedTags = definedTags
	}

	updateReq := core.UpdateVolumeRequest{
		VolumeId:            common.String(request.NativeID),
		UpdateVolumeDetails: updateDetails,
	}

	resp, err := svc.UpdateVolume(ctx, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update Volume: %w", err)
	}

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationUpdate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        *resp.Id,
		},
	}, nil
}

func (p *VolumeProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	svc, err := p.clients.GetBlockstorageClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Blockstorage client: %w", err)
	}

	readReq := &resource.ReadRequest{
		NativeID: request.NativeID,
	}
	readRes, err := p.Read(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to read Volume before delete: %w", err)
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

	deleteReq := core.DeleteVolumeRequest{
		VolumeId: common.String(request.NativeID),
	}

	_, err = svc.DeleteVolume(ctx, deleteReq)
	if err != nil {
		return nil, fmt.Errorf("failed to delete Volume: %w", err)
	}

	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationDelete,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (p *VolumeProvisioner) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			RequestID:       request.RequestID,
		},
	}, nil
}

func (p *VolumeProvisioner) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	svc, err := p.clients.GetBlockstorageClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Blockstorage client: %w", err)
	}

	compartmentId, ok := request.AdditionalProperties["CompartmentId"]
	if !ok {
		return nil, fmt.Errorf("CompartmentId is required for listing Volumes")
	}

	listReq := core.ListVolumesRequest{
		CompartmentId: common.String(compartmentId),
	}

	resp, err := svc.ListVolumes(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list Volumes: %w", err)
	}

	nativeIDs := make([]string, 0, len(resp.Items))
	for _, vol := range resp.Items {
		nativeIDs = append(nativeIDs, *vol.Id)
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

func extractInt64Field(props map[string]any, key string) (int64, bool) {
	if v, ok := props[key].(float64); ok {
		return int64(v), true
	}
	return 0, false
}

func buildVolumeProperties(vol core.Volume) map[string]any {
	properties := map[string]any{
		"CompartmentId":      *vol.CompartmentId,
		"AvailabilityDomain": *vol.AvailabilityDomain,
		"Id":                 *vol.Id,
	}

	if vol.DisplayName != nil {
		properties["DisplayName"] = *vol.DisplayName
	}
	if vol.SizeInGBs != nil {
		properties["SizeInGBs"] = *vol.SizeInGBs
	}
	if vol.VpusPerGB != nil {
		properties["VpusPerGB"] = *vol.VpusPerGB
	}
	if vol.IsAutoTuneEnabled != nil {
		properties["IsAutoTuneEnabled"] = *vol.IsAutoTuneEnabled
	}
	if vol.KmsKeyId != nil {
		properties["KmsKeyId"] = *vol.KmsKeyId
	}
	if vol.FreeformTags != nil {
		properties["FreeformTags"] = util.FreeformTagsToList(vol.FreeformTags)
	}
	if vol.DefinedTags != nil {
		properties["DefinedTags"] = util.DefinedTagsToList(vol.DefinedTags)
	}

	return properties
}
