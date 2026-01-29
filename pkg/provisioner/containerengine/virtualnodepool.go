// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package containerengine

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/containerengine"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/client"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/util"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

type VirtualNodePoolProvisioner struct {
	clients *client.Clients
}

var _ provisioner.Provisioner = &VirtualNodePoolProvisioner{}

func init() {
	provisioner.Register("OCI::ContainerEngine::VirtualNodePool", NewVirtualNodePoolProvisioner)
}

func NewVirtualNodePoolProvisioner(clients *client.Clients) provisioner.Provisioner {
	return &VirtualNodePoolProvisioner{clients: clients}
}

func (p *VirtualNodePoolProvisioner) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	client, err := p.clients.GetContainerEngineClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ContainerEngine client: %w", err)
	}

	var props map[string]any
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	createDetails := containerengine.CreateVirtualNodePoolDetails{
		CompartmentId: common.String(props["CompartmentId"].(string)),
		ClusterId:     common.String(props["ClusterId"].(string)),
		DisplayName:   common.String(props["DisplayName"].(string)),
	}

	// Parse PlacementConfigurations (required)
	if placementConfigs, ok := props["PlacementConfigurations"].([]any); ok {
		configs := make([]containerengine.PlacementConfiguration, 0, len(placementConfigs))
		for _, pc := range placementConfigs {
			if pcMap, ok := pc.(map[string]any); ok {
				placementConfig := containerengine.PlacementConfiguration{}
				if ad, ok := util.ExtractString(pcMap, "AvailabilityDomain"); ok {
					placementConfig.AvailabilityDomain = common.String(ad)
				}
				if subnetId, ok := util.ExtractString(pcMap, "SubnetId"); ok {
					placementConfig.SubnetId = common.String(subnetId)
				}
				if faultDomains, ok := util.ExtractStringSlice(pcMap, "FaultDomains"); ok {
					placementConfig.FaultDomain = faultDomains
				}
				configs = append(configs, placementConfig)
			}
		}
		createDetails.PlacementConfigurations = configs
	}

	// Parse PodConfiguration (required)
	if podConfig, ok := props["PodConfiguration"].(map[string]any); ok {
		config := &containerengine.PodConfiguration{}
		if subnetId, ok := util.ExtractString(podConfig, "SubnetId"); ok {
			config.SubnetId = common.String(subnetId)
		}
		if shape, ok := util.ExtractString(podConfig, "Shape"); ok {
			config.Shape = common.String(shape)
		}
		if nsgIds, ok := util.ExtractStringSlice(podConfig, "NsgIds"); ok {
			config.NsgIds = nsgIds
		}
		createDetails.PodConfiguration = config
	}

	if size, ok := props["Size"].(float64); ok {
		createDetails.Size = common.Int(int(size))
	}

	if nsgIds, ok := util.ExtractStringSlice(props, "NsgIds"); ok {
		createDetails.NsgIds = nsgIds
	}

	// Parse InitialVirtualNodeLabels (using formae.Tag structure: key/value)
	if initialLabels, ok := props["InitialVirtualNodeLabels"].([]any); ok {
		labels := make([]containerengine.InitialVirtualNodeLabel, 0, len(initialLabels))
		for _, label := range initialLabels {
			if labelMap, ok := label.(map[string]any); ok {
				kv := containerengine.InitialVirtualNodeLabel{}
				// Support both lowercase (formae.Tag) and capitalized formats
				if key, ok := util.ExtractString(labelMap, "Key"); ok {
					kv.Key = common.String(key)
				} else if key, ok := util.ExtractString(labelMap, "key"); ok {
					kv.Key = common.String(key)
				}
				if value, ok := util.ExtractString(labelMap, "Value"); ok {
					kv.Value = common.String(value)
				} else if value, ok := util.ExtractString(labelMap, "value"); ok {
					kv.Value = common.String(value)
				}
				labels = append(labels, kv)
			}
		}
		createDetails.InitialVirtualNodeLabels = labels
	}

	// Parse Taints
	if taints, ok := props["Taints"].([]any); ok {
		taintList := make([]containerengine.Taint, 0, len(taints))
		for _, taint := range taints {
			if taintMap, ok := taint.(map[string]any); ok {
				t := containerengine.Taint{}
				if key, ok := util.ExtractString(taintMap, "Key"); ok {
					t.Key = common.String(key)
				}
				if value, ok := util.ExtractString(taintMap, "Value"); ok {
					t.Value = common.String(value)
				}
				if effect, ok := util.ExtractString(taintMap, "Effect"); ok {
					t.Effect = common.String(effect)
				}
				taintList = append(taintList, t)
			}
		}
		createDetails.Taints = taintList
	}

	if freeformTags, ok := util.ExtractFreeformTags(props, "FreeformTags"); ok {
		createDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractDefinedTags(props, "DefinedTags"); ok {
		createDetails.DefinedTags = definedTags
	}

	createReq := containerengine.CreateVirtualNodePoolRequest{
		CreateVirtualNodePoolDetails: createDetails,
	}

	resp, err := client.CreateVirtualNodePool(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create VirtualNodePool: %w", err)
	}

	// VirtualNodePool creation is async - return in-progress with WorkRequest ID
	return &resource.CreateResult{
		ProgressResult: CreateInProgressResult(resource.OperationCreate, *resp.OpcWorkRequestId),
	}, nil
}

func (p *VirtualNodePoolProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	client, err := p.clients.GetContainerEngineClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ContainerEngine client: %w", err)
	}

	props, err := util.ApplyPatchDocument(ctx, request, p.Read)
	if err != nil {
		return nil, err
	}

	updateDetails := containerengine.UpdateVirtualNodePoolDetails{}

	if displayName, ok := util.ExtractString(props, "DisplayName"); ok {
		updateDetails.DisplayName = common.String(displayName)
	}

	// Parse PlacementConfigurations for update
	if placementConfigs, ok := props["PlacementConfigurations"].([]any); ok {
		configs := make([]containerengine.PlacementConfiguration, 0, len(placementConfigs))
		for _, pc := range placementConfigs {
			if pcMap, ok := pc.(map[string]any); ok {
				placementConfig := containerengine.PlacementConfiguration{}
				if ad, ok := util.ExtractString(pcMap, "AvailabilityDomain"); ok {
					placementConfig.AvailabilityDomain = common.String(ad)
				}
				if subnetId, ok := util.ExtractString(pcMap, "SubnetId"); ok {
					placementConfig.SubnetId = common.String(subnetId)
				}
				if faultDomains, ok := util.ExtractStringSlice(pcMap, "FaultDomains"); ok {
					placementConfig.FaultDomain = faultDomains
				}
				configs = append(configs, placementConfig)
			}
		}
		updateDetails.PlacementConfigurations = configs
	}

	// Parse PodConfiguration for update
	if podConfig, ok := props["PodConfiguration"].(map[string]any); ok {
		config := &containerengine.PodConfiguration{}
		if subnetId, ok := util.ExtractString(podConfig, "SubnetId"); ok {
			config.SubnetId = common.String(subnetId)
		}
		if shape, ok := util.ExtractString(podConfig, "Shape"); ok {
			config.Shape = common.String(shape)
		}
		if nsgIds, ok := util.ExtractStringSlice(podConfig, "NsgIds"); ok {
			config.NsgIds = nsgIds
		}
		updateDetails.PodConfiguration = config
	}

	if size, ok := props["Size"].(float64); ok {
		updateDetails.Size = common.Int(int(size))
	}

	if nsgIds, ok := util.ExtractStringSlice(props, "NsgIds"); ok {
		updateDetails.NsgIds = nsgIds
	}

	// Parse InitialVirtualNodeLabels for update
	if initialLabels, ok := props["InitialVirtualNodeLabels"].([]any); ok {
		labels := make([]containerengine.InitialVirtualNodeLabel, 0, len(initialLabels))
		for _, label := range initialLabels {
			if labelMap, ok := label.(map[string]any); ok {
				kv := containerengine.InitialVirtualNodeLabel{}
				if key, ok := util.ExtractString(labelMap, "Key"); ok {
					kv.Key = common.String(key)
				} else if key, ok := util.ExtractString(labelMap, "key"); ok {
					kv.Key = common.String(key)
				}
				if value, ok := util.ExtractString(labelMap, "Value"); ok {
					kv.Value = common.String(value)
				} else if value, ok := util.ExtractString(labelMap, "value"); ok {
					kv.Value = common.String(value)
				}
				labels = append(labels, kv)
			}
		}
		updateDetails.InitialVirtualNodeLabels = labels
	}

	// Parse Taints for update
	if taints, ok := props["Taints"].([]any); ok {
		taintList := make([]containerengine.Taint, 0, len(taints))
		for _, taint := range taints {
			if taintMap, ok := taint.(map[string]any); ok {
				t := containerengine.Taint{}
				if key, ok := util.ExtractString(taintMap, "Key"); ok {
					t.Key = common.String(key)
				}
				if value, ok := util.ExtractString(taintMap, "Value"); ok {
					t.Value = common.String(value)
				}
				if effect, ok := util.ExtractString(taintMap, "Effect"); ok {
					t.Effect = common.String(effect)
				}
				taintList = append(taintList, t)
			}
		}
		updateDetails.Taints = taintList
	}

	if freeformTags, ok := util.ExtractFreeformTags(props, "FreeformTags"); ok {
		updateDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractDefinedTags(props, "DefinedTags"); ok {
		updateDetails.DefinedTags = definedTags
	}

	updateReq := containerengine.UpdateVirtualNodePoolRequest{
		VirtualNodePoolId:            common.String(request.NativeID),
		UpdateVirtualNodePoolDetails: updateDetails,
	}

	resp, err := client.UpdateVirtualNodePool(ctx, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update VirtualNodePool: %w", err)
	}

	// Update is async - return in-progress with WorkRequest ID
	return &resource.UpdateResult{
		ProgressResult: CreateInProgressResult(resource.OperationUpdate, *resp.OpcWorkRequestId),
	}, nil
}

func (p *VirtualNodePoolProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	client, err := p.clients.GetContainerEngineClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ContainerEngine client: %w", err)
	}

	// Check if VirtualNodePool exists before attempting delete
	readReq := &resource.ReadRequest{
		NativeID: request.NativeID,
	}
	readRes, err := p.Read(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to read VirtualNodePool before delete: %w", err)
	}
	if readRes.ErrorCode == resource.OperationErrorCodeNotFound {
		// VirtualNodePool already deleted
		return &resource.DeleteResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationDelete,
				OperationStatus: resource.OperationStatusSuccess,
				NativeID:        request.NativeID,
			},
		}, nil
	}

	deleteReq := containerengine.DeleteVirtualNodePoolRequest{
		VirtualNodePoolId: common.String(request.NativeID),
	}

	resp, err := client.DeleteVirtualNodePool(ctx, deleteReq)
	if err != nil {
		return nil, fmt.Errorf("failed to delete VirtualNodePool: %w", err)
	}

	// Delete is async - return in-progress with WorkRequest ID
	return &resource.DeleteResult{
		ProgressResult: CreateInProgressResult(resource.OperationDelete, *resp.OpcWorkRequestId),
	}, nil
}

func (p *VirtualNodePoolProvisioner) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	client, err := p.clients.GetContainerEngineClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ContainerEngine client: %w", err)
	}

	// Poll the WorkRequest for status
	result, err := CheckWorkRequestStatus(ctx, client, request.RequestID, resource.OperationCheckStatus)
	if err != nil {
		return nil, err
	}

	return &resource.StatusResult{
		ProgressResult: result,
	}, nil
}

func (p *VirtualNodePoolProvisioner) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	client, err := p.clients.GetContainerEngineClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ContainerEngine client: %w", err)
	}

	getReq := containerengine.GetVirtualNodePoolRequest{
		VirtualNodePoolId: common.String(request.NativeID),
	}

	resp, err := client.GetVirtualNodePool(ctx, getReq)
	if err != nil {
		// Check if not found
		if serviceErr, ok := common.IsServiceError(err); ok && serviceErr.GetHTTPStatusCode() == 404 {
			return &resource.ReadResult{
				ResourceType: "OCI::ContainerEngine::VirtualNodePool",
				ErrorCode:    resource.OperationErrorCodeNotFound,
			}, nil
		}
		return nil, fmt.Errorf("failed to read VirtualNodePool: %w", err)
	}

	// Build properties map
	props := map[string]any{
		"CompartmentId": *resp.CompartmentId,
		"Id":            *resp.Id,
		"ClusterId":     *resp.ClusterId,
		"DisplayName":   *resp.DisplayName,
	}

	if resp.LifecycleState != "" {
		props["LifecycleState"] = string(resp.LifecycleState)
	}

	if resp.Size != nil {
		props["Size"] = *resp.Size
	}

	if resp.NsgIds != nil {
		props["NsgIds"] = resp.NsgIds
	}

	// PlacementConfigurations
	if len(resp.PlacementConfigurations) > 0 {
		placementConfigs := make([]map[string]any, 0, len(resp.PlacementConfigurations))
		for _, pc := range resp.PlacementConfigurations {
			pcMap := map[string]any{}
			if pc.AvailabilityDomain != nil {
				pcMap["AvailabilityDomain"] = *pc.AvailabilityDomain
			}
			if pc.SubnetId != nil {
				pcMap["SubnetId"] = *pc.SubnetId
			}
			if len(pc.FaultDomain) > 0 {
				pcMap["FaultDomains"] = pc.FaultDomain
			}
			placementConfigs = append(placementConfigs, pcMap)
		}
		props["PlacementConfigurations"] = placementConfigs
	}

	// PodConfiguration
	if resp.PodConfiguration != nil {
		podConfig := map[string]any{}
		if resp.PodConfiguration.SubnetId != nil {
			podConfig["SubnetId"] = *resp.PodConfiguration.SubnetId
		}
		if resp.PodConfiguration.Shape != nil {
			podConfig["Shape"] = *resp.PodConfiguration.Shape
		}
		if resp.PodConfiguration.NsgIds != nil {
			podConfig["NsgIds"] = resp.PodConfiguration.NsgIds
		}
		props["PodConfiguration"] = podConfig
	}

	// InitialVirtualNodeLabels
	if len(resp.InitialVirtualNodeLabels) > 0 {
		labels := make([]map[string]any, 0, len(resp.InitialVirtualNodeLabels))
		for _, label := range resp.InitialVirtualNodeLabels {
			labelMap := map[string]any{}
			if label.Key != nil {
				labelMap["Key"] = *label.Key
			}
			if label.Value != nil {
				labelMap["Value"] = *label.Value
			}
			labels = append(labels, labelMap)
		}
		props["InitialVirtualNodeLabels"] = labels
	}

	// Taints
	if len(resp.Taints) > 0 {
		taints := make([]map[string]any, 0, len(resp.Taints))
		for _, taint := range resp.Taints {
			taintMap := map[string]any{}
			if taint.Key != nil {
				taintMap["Key"] = *taint.Key
			}
			if taint.Value != nil {
				taintMap["Value"] = *taint.Value
			}
			if taint.Effect != nil {
				taintMap["Effect"] = *taint.Effect
			}
			taints = append(taints, taintMap)
		}
		props["Taints"] = taints
	}

	if resp.FreeformTags != nil {
		props["FreeformTags"] = util.FreeformTagsToList(resp.FreeformTags)
	}
	if resp.DefinedTags != nil {
		props["DefinedTags"] = util.DefinedTagsToList(resp.DefinedTags)
	}

	propBytes, err := json.Marshal(props)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal VirtualNodePool properties: %w", err)
	}

	return &resource.ReadResult{
		ResourceType: "OCI::ContainerEngine::VirtualNodePool",
		Properties:   string(propBytes),
	}, nil
}

func (p *VirtualNodePoolProvisioner) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	client, err := p.clients.GetContainerEngineClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ContainerEngine client: %w", err)
	}

	var compartmentId string
	var clusterId string

	// Check if CompartmentId is provided directly
	if cid, ok := request.AdditionalProperties["CompartmentId"]; ok {
		compartmentId = cid
	}

	// Check if ClusterId is provided
	if clid, ok := request.AdditionalProperties["ClusterId"]; ok {
		clusterId = clid
		// If we have ClusterId but no CompartmentId, derive it from the cluster
		if compartmentId == "" {
			getReq := containerengine.GetClusterRequest{
				ClusterId: common.String(clusterId),
			}
			resp, err := client.GetCluster(ctx, getReq)
			if err != nil {
				return nil, fmt.Errorf("failed to get Cluster to derive CompartmentId: %w", err)
			}
			compartmentId = *resp.CompartmentId
		}
	}

	if compartmentId == "" {
		return nil, fmt.Errorf("CompartmentId is required for listing VirtualNodePools (either directly or derived from ClusterId)")
	}

	listReq := containerengine.ListVirtualNodePoolsRequest{
		CompartmentId: common.String(compartmentId),
		// Filter out deleted/deleting/failed virtual node pools - only return active or in-progress states
		LifecycleState: []containerengine.VirtualNodePoolLifecycleStateEnum{
			containerengine.VirtualNodePoolLifecycleStateCreating,
			containerengine.VirtualNodePoolLifecycleStateActive,
			containerengine.VirtualNodePoolLifecycleStateUpdating,
			containerengine.VirtualNodePoolLifecycleStateNeedsAttention,
		},
	}

	// Filter by ClusterId if provided
	if clusterId != "" {
		listReq.ClusterId = common.String(clusterId)
	}

	resp, err := client.ListVirtualNodePools(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list VirtualNodePools: %w", err)
	}

	nativeIDs := make([]string, 0, len(resp.Items))
	for _, pool := range resp.Items {
		nativeIDs = append(nativeIDs, *pool.Id)
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}
