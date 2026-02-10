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

type InstanceProvisioner struct {
	clients *client.Clients
}

var _ provisioner.Provisioner = &InstanceProvisioner{}

func init() {
	provisioner.Register("OCI::Core::Instance", NewInstanceProvisioner)
}

func NewInstanceProvisioner(clients *client.Clients) provisioner.Provisioner {
	return &InstanceProvisioner{clients: clients}
}

func (p *InstanceProvisioner) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	svc, err := p.clients.GetComputeClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Compute client: %w", err)
	}

	var props map[string]any
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	launchDetails := core.LaunchInstanceDetails{
		CompartmentId:      common.String(props["CompartmentId"].(string)),
		AvailabilityDomain: common.String(props["AvailabilityDomain"].(string)),
		Shape:              common.String(props["Shape"].(string)),
	}

	if displayName, ok := util.ExtractString(props, "DisplayName"); ok {
		launchDetails.DisplayName = common.String(displayName)
	}

	if sourceDetails, ok := props["SourceDetails"].(map[string]any); ok {
		launchDetails.SourceDetails = parseSourceDetails(sourceDetails)
	}

	if vnicDetails, ok := props["CreateVnicDetails"].(map[string]any); ok {
		launchDetails.CreateVnicDetails = parseCreateVnicDetails(vnicDetails)
	}

	if shapeConfig, ok := props["ShapeConfig"].(map[string]any); ok {
		launchDetails.ShapeConfig = parseShapeConfig(shapeConfig)
	}

	if metadata, ok := props["Metadata"].(map[string]any); ok {
		m := make(map[string]string, len(metadata))
		for k, v := range metadata {
			if s, ok := v.(string); ok {
				m[k] = s
			}
		}
		launchDetails.Metadata = m
	}

	if freeformTags, ok := util.ExtractFreeformTags(props, "FreeformTags"); ok {
		launchDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractDefinedTags(props, "DefinedTags"); ok {
		launchDetails.DefinedTags = definedTags
	}

	createReq := core.LaunchInstanceRequest{
		LaunchInstanceDetails: launchDetails,
	}

	resp, err := svc.LaunchInstance(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create Instance: %w", err)
	}

	// Instance launch is async — return in-progress, poll lifecycle in Status()
	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCreate,
			OperationStatus: resource.OperationStatusInProgress,
			NativeID:        *resp.Id,
			RequestID:       *resp.Id,
		},
	}, nil
}

func (p *InstanceProvisioner) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	svc, err := p.clients.GetComputeClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Compute client: %w", err)
	}

	getReq := core.GetInstanceRequest{
		InstanceId: common.String(request.NativeID),
	}

	resp, err := svc.GetInstance(ctx, getReq)
	if err != nil {
		if serviceErr, ok := common.IsServiceError(err); ok && serviceErr.GetHTTPStatusCode() == 404 {
			return &resource.ReadResult{
				ResourceType: "OCI::Core::Instance",
				ErrorCode:    resource.OperationErrorCodeNotFound,
			}, nil
		}
		return nil, fmt.Errorf("failed to read Instance: %w", err)
	}

	// Treat terminated instances as not found
	if resp.LifecycleState == core.InstanceLifecycleStateTerminated {
		return &resource.ReadResult{
			ResourceType: "OCI::Core::Instance",
			ErrorCode:    resource.OperationErrorCodeNotFound,
		}, nil
	}

	properties := buildInstanceProperties(resp.Instance)

	propBytes, err := json.Marshal(properties)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Instance properties: %w", err)
	}

	return &resource.ReadResult{
		ResourceType: "OCI::Core::Instance",
		Properties:   string(propBytes),
	}, nil
}

func (p *InstanceProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	svc, err := p.clients.GetComputeClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Compute client: %w", err)
	}

	props, err := util.ApplyPatchDocument(ctx, request, p.Read)
	if err != nil {
		return nil, err
	}

	updateDetails := core.UpdateInstanceDetails{}

	if displayName, ok := util.ExtractString(props, "DisplayName"); ok {
		updateDetails.DisplayName = common.String(displayName)
	}
	if shape, ok := util.ExtractString(props, "Shape"); ok {
		updateDetails.Shape = common.String(shape)
	}
	if shapeConfig, ok := props["ShapeConfig"].(map[string]any); ok {
		updateDetails.ShapeConfig = parseUpdateShapeConfig(shapeConfig)
	}
	if metadata, ok := props["Metadata"].(map[string]any); ok {
		m := make(map[string]string, len(metadata))
		for k, v := range metadata {
			if s, ok := v.(string); ok {
				m[k] = s
			}
		}
		updateDetails.Metadata = m
	}
	if freeformTags, ok := util.ExtractFreeformTags(props, "FreeformTags"); ok {
		updateDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractDefinedTags(props, "DefinedTags"); ok {
		updateDetails.DefinedTags = definedTags
	}

	updateReq := core.UpdateInstanceRequest{
		InstanceId:            common.String(request.NativeID),
		UpdateInstanceDetails: updateDetails,
	}

	resp, err := svc.UpdateInstance(ctx, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update Instance: %w", err)
	}

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationUpdate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        *resp.Id,
		},
	}, nil
}

func (p *InstanceProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	svc, err := p.clients.GetComputeClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Compute client: %w", err)
	}

	readReq := &resource.ReadRequest{
		NativeID: request.NativeID,
	}
	readRes, err := p.Read(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to read Instance before delete: %w", err)
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

	deleteReq := core.TerminateInstanceRequest{
		InstanceId:         common.String(request.NativeID),
		PreserveBootVolume: common.Bool(false),
	}

	_, err = svc.TerminateInstance(ctx, deleteReq)
	if err != nil {
		return nil, fmt.Errorf("failed to delete Instance: %w", err)
	}

	// Terminate is async — poll lifecycle in Status()
	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationDelete,
			OperationStatus: resource.OperationStatusInProgress,
			NativeID:        request.NativeID,
			RequestID:       request.NativeID,
		},
	}, nil
}

func (p *InstanceProvisioner) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	svc, err := p.clients.GetComputeClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Compute client: %w", err)
	}

	getReq := core.GetInstanceRequest{
		InstanceId: common.String(request.RequestID),
	}

	resp, err := svc.GetInstance(ctx, getReq)
	if err != nil {
		if serviceErr, ok := common.IsServiceError(err); ok && serviceErr.GetHTTPStatusCode() == 404 {
			// Instance gone — if we were deleting, that's success
			return &resource.StatusResult{
				ProgressResult: &resource.ProgressResult{
					Operation:       resource.OperationCheckStatus,
					OperationStatus: resource.OperationStatusSuccess,
					NativeID:        request.RequestID,
				},
			}, nil
		}
		return nil, fmt.Errorf("failed to check Instance status: %w", err)
	}

	switch resp.LifecycleState {
	case core.InstanceLifecycleStateRunning:
		properties := buildInstanceProperties(resp.Instance)
		propertiesBytes, err := json.Marshal(properties)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal properties: %w", err)
		}
		return &resource.StatusResult{
			ProgressResult: &resource.ProgressResult{
				Operation:          resource.OperationCheckStatus,
				OperationStatus:    resource.OperationStatusSuccess,
				NativeID:           *resp.Id,
				ResourceProperties: json.RawMessage(propertiesBytes),
			},
		}, nil

	case core.InstanceLifecycleStateTerminated:
		return &resource.StatusResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCheckStatus,
				OperationStatus: resource.OperationStatusSuccess,
				NativeID:        *resp.Id,
			},
		}, nil

	case core.InstanceLifecycleStateStopped:
		properties := buildInstanceProperties(resp.Instance)
		propertiesBytes, err := json.Marshal(properties)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal properties: %w", err)
		}
		return &resource.StatusResult{
			ProgressResult: &resource.ProgressResult{
				Operation:          resource.OperationCheckStatus,
				OperationStatus:    resource.OperationStatusSuccess,
				NativeID:           *resp.Id,
				ResourceProperties: json.RawMessage(propertiesBytes),
			},
		}, nil

	default: // PROVISIONING, STARTING, STOPPING, TERMINATING, etc.
		return &resource.StatusResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCheckStatus,
				OperationStatus: resource.OperationStatusInProgress,
				RequestID:       request.RequestID,
				StatusMessage:   fmt.Sprintf("Instance lifecycle state: %s", resp.LifecycleState),
			},
		}, nil
	}
}

func (p *InstanceProvisioner) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	svc, err := p.clients.GetComputeClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Compute client: %w", err)
	}

	compartmentId, ok := request.AdditionalProperties["CompartmentId"]
	if !ok {
		return nil, fmt.Errorf("CompartmentId is required for listing Instances")
	}

	listReq := core.ListInstancesRequest{
		CompartmentId:  common.String(compartmentId),
		LifecycleState: core.InstanceLifecycleStateRunning,
	}

	resp, err := svc.ListInstances(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list Instances: %w", err)
	}

	nativeIDs := make([]string, 0, len(resp.Items))
	for _, inst := range resp.Items {
		nativeIDs = append(nativeIDs, *inst.Id)
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

func parseSourceDetails(data map[string]any) core.InstanceSourceDetails {
	sourceType, _ := extractStringField(data, "sourceType", "SourceType")

	switch sourceType {
	case "image":
		details := core.InstanceSourceViaImageDetails{}
		if imageId, ok := extractStringField(data, "imageId", "ImageId"); ok {
			details.ImageId = common.String(imageId)
		}
		if bootVolumeSizeInGBs, ok := extractInt64Field(data, "BootVolumeSizeInGBs"); ok {
			details.BootVolumeSizeInGBs = common.Int64(bootVolumeSizeInGBs)
		} else if bootVolumeSizeInGBs, ok := extractInt64Field(data, "bootVolumeSizeInGBs"); ok {
			details.BootVolumeSizeInGBs = common.Int64(bootVolumeSizeInGBs)
		}
		return details
	case "bootVolume":
		details := core.InstanceSourceViaBootVolumeDetails{}
		if bootVolumeId, ok := extractStringField(data, "bootVolumeId", "BootVolumeId"); ok {
			details.BootVolumeId = common.String(bootVolumeId)
		}
		return details
	default:
		return nil
	}
}

func parseCreateVnicDetails(data map[string]any) *core.CreateVnicDetails {
	details := &core.CreateVnicDetails{}

	if subnetId, ok := extractStringField(data, "subnetId", "SubnetId"); ok {
		details.SubnetId = common.String(subnetId)
	}
	if displayName, ok := extractStringField(data, "displayName", "DisplayName"); ok {
		details.DisplayName = common.String(displayName)
	}
	if assignPublicIp, ok := extractBoolField(data, "assignPublicIp", "AssignPublicIp"); ok {
		details.AssignPublicIp = common.Bool(assignPublicIp)
	}
	if assignPrivateDnsRecord, ok := extractBoolField(data, "assignPrivateDnsRecord", "AssignPrivateDnsRecord"); ok {
		details.AssignPrivateDnsRecord = common.Bool(assignPrivateDnsRecord)
	}
	if hostnameLabel, ok := extractStringField(data, "hostnameLabel", "HostnameLabel"); ok {
		details.HostnameLabel = common.String(hostnameLabel)
	}
	if nsgIds := extractStringSliceField(data, "nsgIds", "NsgIds"); len(nsgIds) > 0 {
		details.NsgIds = nsgIds
	}
	if privateIp, ok := extractStringField(data, "privateIp", "PrivateIp"); ok {
		details.PrivateIp = common.String(privateIp)
	}
	if skipSourceDestCheck, ok := extractBoolField(data, "skipSourceDestCheck", "SkipSourceDestCheck"); ok {
		details.SkipSourceDestCheck = common.Bool(skipSourceDestCheck)
	}
	if freeformTags, ok := util.ExtractFreeformTags(data, "freeformTags"); ok {
		details.FreeformTags = freeformTags
	} else if freeformTags, ok := util.ExtractFreeformTags(data, "FreeformTags"); ok {
		details.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractDefinedTags(data, "definedTags"); ok {
		details.DefinedTags = definedTags
	} else if definedTags, ok := util.ExtractDefinedTags(data, "DefinedTags"); ok {
		details.DefinedTags = definedTags
	}

	return details
}

func parseShapeConfig(data map[string]any) *core.LaunchInstanceShapeConfigDetails {
	config := &core.LaunchInstanceShapeConfigDetails{}

	if ocpus, ok := extractFloatField(data, "ocpus", "Ocpus"); ok {
		config.Ocpus = common.Float32(float32(ocpus))
	}
	if memoryInGBs, ok := extractFloatField(data, "memoryInGBs", "MemoryInGBs"); ok {
		config.MemoryInGBs = common.Float32(float32(memoryInGBs))
	}
	if baselineOcpuUtilization, ok := extractStringField(data, "baselineOcpuUtilization", "BaselineOcpuUtilization"); ok {
		config.BaselineOcpuUtilization = core.LaunchInstanceShapeConfigDetailsBaselineOcpuUtilizationEnum(baselineOcpuUtilization)
	}

	return config
}

func parseUpdateShapeConfig(data map[string]any) *core.UpdateInstanceShapeConfigDetails {
	config := &core.UpdateInstanceShapeConfigDetails{}

	if ocpus, ok := extractFloatField(data, "ocpus", "Ocpus"); ok {
		config.Ocpus = common.Float32(float32(ocpus))
	}
	if memoryInGBs, ok := extractFloatField(data, "memoryInGBs", "MemoryInGBs"); ok {
		config.MemoryInGBs = common.Float32(float32(memoryInGBs))
	}
	if baselineOcpuUtilization, ok := extractStringField(data, "baselineOcpuUtilization", "BaselineOcpuUtilization"); ok {
		config.BaselineOcpuUtilization = core.UpdateInstanceShapeConfigDetailsBaselineOcpuUtilizationEnum(baselineOcpuUtilization)
	}

	return config
}

func extractFloatField(m map[string]any, lowerKey, upperKey string) (float64, bool) {
	if v, ok := m[lowerKey].(float64); ok {
		return v, true
	}
	if v, ok := m[upperKey].(float64); ok {
		return v, true
	}
	return 0, false
}

func buildInstanceProperties(inst core.Instance) map[string]any {
	properties := map[string]any{
		"CompartmentId":      *inst.CompartmentId,
		"AvailabilityDomain": *inst.AvailabilityDomain,
		"Id":                 *inst.Id,
		"Shape":              *inst.Shape,
	}

	if inst.DisplayName != nil {
		properties["DisplayName"] = *inst.DisplayName
	}
	if inst.LifecycleState != "" {
		properties["LifecycleState"] = string(inst.LifecycleState)
	}

	if inst.SourceDetails != nil {
		switch v := inst.SourceDetails.(type) {
		case core.InstanceSourceViaImageDetails:
			sd := map[string]any{"sourceType": "image"}
			if v.ImageId != nil {
				sd["imageId"] = *v.ImageId
			}
			if v.BootVolumeSizeInGBs != nil {
				sd["bootVolumeSizeInGBs"] = *v.BootVolumeSizeInGBs
			}
			properties["SourceDetails"] = sd
		case core.InstanceSourceViaBootVolumeDetails:
			sd := map[string]any{"sourceType": "bootVolume"}
			if v.BootVolumeId != nil {
				sd["bootVolumeId"] = *v.BootVolumeId
			}
			properties["SourceDetails"] = sd
		}
	}

	if inst.ShapeConfig != nil {
		sc := map[string]any{}
		if inst.ShapeConfig.Ocpus != nil {
			sc["ocpus"] = *inst.ShapeConfig.Ocpus
		}
		if inst.ShapeConfig.MemoryInGBs != nil {
			sc["memoryInGBs"] = *inst.ShapeConfig.MemoryInGBs
		}
		if inst.ShapeConfig.BaselineOcpuUtilization != "" {
			sc["baselineOcpuUtilization"] = string(inst.ShapeConfig.BaselineOcpuUtilization)
		}
		if len(sc) > 0 {
			properties["ShapeConfig"] = sc
		}
	}

	if len(inst.Metadata) > 0 {
		properties["Metadata"] = inst.Metadata
	}

	if inst.FreeformTags != nil {
		properties["FreeformTags"] = util.FreeformTagsToList(inst.FreeformTags)
	}
	if inst.DefinedTags != nil {
		properties["DefinedTags"] = util.DefinedTagsToList(inst.DefinedTags)
	}

	return properties
}
