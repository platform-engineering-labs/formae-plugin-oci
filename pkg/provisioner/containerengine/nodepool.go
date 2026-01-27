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
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/client"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/util"
)

type NodePoolProvisioner struct {
	clients *client.Clients
}

var _ provisioner.Provisioner = &NodePoolProvisioner{}

func init() {
	provisioner.Register("OCI::ContainerEngine::NodePool", NewNodePoolProvisioner)
}

func NewNodePoolProvisioner(clients *client.Clients) provisioner.Provisioner {
	return &NodePoolProvisioner{clients: clients}
}

func (p *NodePoolProvisioner) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	client, err := p.clients.GetContainerEngineClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ContainerEngine client: %w", err)
	}

	var props map[string]any
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	createDetails := containerengine.CreateNodePoolDetails{
		CompartmentId: common.String(props["CompartmentId"].(string)),
		ClusterId:     common.String(props["ClusterId"].(string)),
		Name:          common.String(props["Name"].(string)),
		NodeShape:     common.String(props["NodeShape"].(string)),
	}

	if kubernetesVersion, ok := util.ExtractString(props, "KubernetesVersion"); ok {
		createDetails.KubernetesVersion = common.String(kubernetesVersion)
	}

	if nodeShapeConfig, ok := props["NodeShapeConfig"].(map[string]any); ok {
		config := &containerengine.CreateNodeShapeConfigDetails{}
		if ocpus, ok := nodeShapeConfig["ocpus"].(float64); ok {
			config.Ocpus = common.Float32(float32(ocpus))
		}
		if memoryInGBs, ok := nodeShapeConfig["memoryInGBs"].(float64); ok {
			config.MemoryInGBs = common.Float32(float32(memoryInGBs))
		}
		createDetails.NodeShapeConfig = config
	}

	if nodeConfigDetails, ok := props["NodeConfigDetails"].(map[string]any); ok {
		config := containerengine.CreateNodePoolNodeConfigDetails{}

		if size, ok := nodeConfigDetails["size"].(float64); ok {
			config.Size = common.Int(int(size))
		}

		if placementConfigs, ok := nodeConfigDetails["placementConfigs"].([]any); ok {
			configs := make([]containerengine.NodePoolPlacementConfigDetails, 0, len(placementConfigs))
			for _, pc := range placementConfigs {
				if pcMap, ok := pc.(map[string]any); ok {
					placementConfig := containerengine.NodePoolPlacementConfigDetails{}
					if ad, ok := util.ExtractString(pcMap, "availabilityDomain"); ok {
						placementConfig.AvailabilityDomain = common.String(ad)
					}
					if subnetId, ok := util.ExtractString(pcMap, "subnetId"); ok {
						placementConfig.SubnetId = common.String(subnetId)
					}
					if capacityReservationId, ok := util.ExtractString(pcMap, "capacityReservationId"); ok {
						placementConfig.CapacityReservationId = common.String(capacityReservationId)
					}
					if faultDomains, ok := util.ExtractStringSlice(pcMap, "faultDomains"); ok {
						placementConfig.FaultDomains = faultDomains
					}
					configs = append(configs, placementConfig)
				}
			}
			config.PlacementConfigs = configs
		}

		if nsgIds, ok := util.ExtractStringSlice(nodeConfigDetails, "nsgIds"); ok {
			config.NsgIds = nsgIds
		}
		if isPvEncryptionInTransitEnabled, ok := util.ExtractBool(nodeConfigDetails, "isPvEncryptionInTransitEnabled"); ok {
			config.IsPvEncryptionInTransitEnabled = common.Bool(isPvEncryptionInTransitEnabled)
		}
		if freeformTags, ok := util.ExtractTag(nodeConfigDetails, "freeformTags"); ok {
			config.FreeformTags = freeformTags
		}
		if definedTags, ok := util.ExtractNestedTag(nodeConfigDetails, "definedTags"); ok {
			config.DefinedTags = definedTags
		}

		createDetails.NodeConfigDetails = &config
	}

	// Parse NodeSourceDetails (required - nested class fields stay camelCase)
	// User must provide an OKE-optimized image OCID for their region and K8s version
	// See: https://docs.oracle.com/en-us/iaas/Content/ContEng/Reference/contengimagesshapes.htm
	if nodeSourceDetails, ok := props["NodeSourceDetails"].(map[string]any); ok {
		if imageId, ok := util.ExtractString(nodeSourceDetails, "imageId"); ok {
			sourceDetails := containerengine.NodeSourceViaImageDetails{
				ImageId: common.String(imageId),
			}
			if bootVolumeSizeInGBs, ok := nodeSourceDetails["bootVolumeSizeInGBs"].(float64); ok {
				sourceDetails.BootVolumeSizeInGBs = common.Int64(int64(bootVolumeSizeInGBs))
			}
			createDetails.NodeSourceDetails = sourceDetails
		} else {
			return nil, fmt.Errorf("nodeSourceDetails.imageId is required but not provided")
		}
	} else {
		return nil, fmt.Errorf("nodeSourceDetails is required for NodePool creation - specify the OKE-optimized image OCID for your region")
	}

	if sshPublicKey, ok := util.ExtractString(props, "SshPublicKey"); ok {
		createDetails.SshPublicKey = common.String(sshPublicKey)
	}

	// Parse InitialNodeLabels (nested class fields stay camelCase)
	if initialNodeLabels, ok := props["InitialNodeLabels"].([]any); ok {
		labels := make([]containerengine.KeyValue, 0, len(initialNodeLabels))
		for _, label := range initialNodeLabels {
			if labelMap, ok := label.(map[string]any); ok {
				kv := containerengine.KeyValue{}
				if key, ok := util.ExtractString(labelMap, "key"); ok {
					kv.Key = common.String(key)
				}
				if value, ok := util.ExtractString(labelMap, "value"); ok {
					kv.Value = common.String(value)
				}
				labels = append(labels, kv)
			}
		}
		createDetails.InitialNodeLabels = labels
	}

	if freeformTags, ok := util.ExtractTag(props, "FreeformTags"); ok {
		createDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractNestedTag(props, "DefinedTags"); ok {
		createDetails.DefinedTags = definedTags
	}

	createReq := containerengine.CreateNodePoolRequest{
		CreateNodePoolDetails: createDetails,
	}

	resp, err := client.CreateNodePool(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create NodePool: %w", err)
	}

	// NodePool creation is async - return in-progress with WorkRequest ID
	return &resource.CreateResult{
		ProgressResult: CreateInProgressResult(resource.OperationCreate, *resp.OpcWorkRequestId),
	}, nil
}

func (p *NodePoolProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	client, err := p.clients.GetContainerEngineClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ContainerEngine client: %w", err)
	}

	props, err := util.ApplyPatchDocument(ctx, request, p.Read)
	if err != nil {
		return nil, err
	}

	updateDetails := containerengine.UpdateNodePoolDetails{}

	if name, ok := util.ExtractString(props, "Name"); ok {
		updateDetails.Name = common.String(name)
	}
	if kubernetesVersion, ok := util.ExtractString(props, "KubernetesVersion"); ok {
		updateDetails.KubernetesVersion = common.String(kubernetesVersion)
	}

	// Parse NodeShapeConfig for flexible shapes
	if nodeShapeConfig, ok := props["NodeShapeConfig"].(map[string]any); ok {
		config := &containerengine.UpdateNodeShapeConfigDetails{}
		if ocpus, ok := nodeShapeConfig["Ocpus"].(float64); ok {
			config.Ocpus = common.Float32(float32(ocpus))
		}
		if memoryInGBs, ok := nodeShapeConfig["MemoryInGBs"].(float64); ok {
			config.MemoryInGBs = common.Float32(float32(memoryInGBs))
		}
		updateDetails.NodeShapeConfig = config
	}

	// Parse NodeConfigDetails for scaling
	if nodeConfigDetails, ok := props["NodeConfigDetails"].(map[string]any); ok {
		config := &containerengine.UpdateNodePoolNodeConfigDetails{}

		if size, ok := nodeConfigDetails["Size"].(float64); ok {
			config.Size = common.Int(int(size))
		}
		if nsgIds, ok := util.ExtractStringSlice(nodeConfigDetails, "NsgIds"); ok {
			config.NsgIds = nsgIds
		}
		if isPvEncryptionInTransitEnabled, ok := util.ExtractBool(nodeConfigDetails, "IsPvEncryptionInTransitEnabled"); ok {
			config.IsPvEncryptionInTransitEnabled = common.Bool(isPvEncryptionInTransitEnabled)
		}

		// Parse PlacementConfigs for update
		if placementConfigs, ok := nodeConfigDetails["PlacementConfigs"].([]any); ok {
			configs := make([]containerengine.NodePoolPlacementConfigDetails, 0, len(placementConfigs))
			for _, pc := range placementConfigs {
				if pcMap, ok := pc.(map[string]any); ok {
					placementConfig := containerengine.NodePoolPlacementConfigDetails{}
					if ad, ok := util.ExtractString(pcMap, "AvailabilityDomain"); ok {
						placementConfig.AvailabilityDomain = common.String(ad)
					}
					if subnetId, ok := util.ExtractString(pcMap, "SubnetId"); ok {
						placementConfig.SubnetId = common.String(subnetId)
					}
					configs = append(configs, placementConfig)
				}
			}
			config.PlacementConfigs = configs
		}

		updateDetails.NodeConfigDetails = config
	}

	// Parse InitialNodeLabels
	if initialNodeLabels, ok := props["InitialNodeLabels"].([]any); ok {
		labels := make([]containerengine.KeyValue, 0, len(initialNodeLabels))
		for _, label := range initialNodeLabels {
			if labelMap, ok := label.(map[string]any); ok {
				kv := containerengine.KeyValue{}
				if key, ok := util.ExtractString(labelMap, "Key"); ok {
					kv.Key = common.String(key)
				}
				if value, ok := util.ExtractString(labelMap, "Value"); ok {
					kv.Value = common.String(value)
				}
				labels = append(labels, kv)
			}
		}
		updateDetails.InitialNodeLabels = labels
	}

	if freeformTags, ok := util.ExtractTag(props, "FreeformTags"); ok {
		updateDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractNestedTag(props, "DefinedTags"); ok {
		updateDetails.DefinedTags = definedTags
	}

	updateReq := containerengine.UpdateNodePoolRequest{
		NodePoolId:            common.String(request.NativeID),
		UpdateNodePoolDetails: updateDetails,
	}

	resp, err := client.UpdateNodePool(ctx, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update NodePool: %w", err)
	}

	// Update is async - return in-progress with WorkRequest ID
	return &resource.UpdateResult{
		ProgressResult: CreateInProgressResult(resource.OperationUpdate, *resp.OpcWorkRequestId),
	}, nil
}

func (p *NodePoolProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	client, err := p.clients.GetContainerEngineClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ContainerEngine client: %w", err)
	}

	// Check if NodePool exists before attempting delete
	readReq := &resource.ReadRequest{
		NativeID: request.NativeID,
	}
	readRes, err := p.Read(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to read NodePool before delete: %w", err)
	}
	if readRes.ErrorCode == resource.OperationErrorCodeNotFound {
		// NodePool already deleted
		return &resource.DeleteResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationDelete,
				OperationStatus: resource.OperationStatusSuccess,
				NativeID:        request.NativeID,
			},
		}, nil
	}

	deleteReq := containerengine.DeleteNodePoolRequest{
		NodePoolId: common.String(request.NativeID),
	}

	resp, err := client.DeleteNodePool(ctx, deleteReq)
	if err != nil {
		return nil, fmt.Errorf("failed to delete NodePool: %w", err)
	}

	// Delete is async - return in-progress with WorkRequest ID
	return &resource.DeleteResult{
		ProgressResult: CreateInProgressResult(resource.OperationDelete, *resp.OpcWorkRequestId),
	}, nil
}

func (p *NodePoolProvisioner) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	client, err := p.clients.GetContainerEngineClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ContainerEngine client: %w", err)
	}

	// Poll the WorkRequest for status
	// The operation type will be determined from the WorkRequest itself
	result, err := CheckWorkRequestStatus(ctx, client, request.RequestID, resource.OperationCheckStatus)
	if err != nil {
		return nil, err
	}

	return &resource.StatusResult{
		ProgressResult: result,
	}, nil
}

func (p *NodePoolProvisioner) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	client, err := p.clients.GetContainerEngineClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ContainerEngine client: %w", err)
	}

	getReq := containerengine.GetNodePoolRequest{
		NodePoolId: common.String(request.NativeID),
	}

	resp, err := client.GetNodePool(ctx, getReq)
	if err != nil {
		// Check if not found
		if serviceErr, ok := common.IsServiceError(err); ok && serviceErr.GetHTTPStatusCode() == 404 {
			return &resource.ReadResult{
				ResourceType: "OCI::ContainerEngine::NodePool",
				ErrorCode:    resource.OperationErrorCodeNotFound,
			}, nil
		}
		return nil, fmt.Errorf("failed to read NodePool: %w", err)
	}

	// Build properties map
	props := map[string]any{
		"CompartmentId": *resp.CompartmentId,
		"Id":            *resp.Id,
		"ClusterId":     *resp.ClusterId,
		"Name":          *resp.Name,
		"NodeShape":     *resp.NodeShape,
	}

	if resp.KubernetesVersion != nil {
		props["KubernetesVersion"] = *resp.KubernetesVersion
	}
	if resp.LifecycleState != "" {
		props["LifecycleState"] = string(resp.LifecycleState)
	}

	// NodeShapeConfig
	if resp.NodeShapeConfig != nil {
		shapeConfig := map[string]any{}
		if resp.NodeShapeConfig.Ocpus != nil {
			shapeConfig["Ocpus"] = *resp.NodeShapeConfig.Ocpus
		}
		if resp.NodeShapeConfig.MemoryInGBs != nil {
			shapeConfig["MemoryInGBs"] = *resp.NodeShapeConfig.MemoryInGBs
		}
		if len(shapeConfig) > 0 {
			props["NodeShapeConfig"] = shapeConfig
		}
	}

	// NodeConfigDetails
	if resp.NodeConfigDetails != nil {
		nodeConfig := map[string]any{}
		if resp.NodeConfigDetails.Size != nil {
			nodeConfig["Size"] = *resp.NodeConfigDetails.Size
		}
		if resp.NodeConfigDetails.NsgIds != nil {
			nodeConfig["NsgIds"] = resp.NodeConfigDetails.NsgIds
		}
		if resp.NodeConfigDetails.IsPvEncryptionInTransitEnabled != nil {
			nodeConfig["IsPvEncryptionInTransitEnabled"] = *resp.NodeConfigDetails.IsPvEncryptionInTransitEnabled
		}

		// PlacementConfigs
		if len(resp.NodeConfigDetails.PlacementConfigs) > 0 {
			placementConfigs := make([]map[string]any, 0, len(resp.NodeConfigDetails.PlacementConfigs))
			for _, pc := range resp.NodeConfigDetails.PlacementConfigs {
				pcMap := map[string]any{}
				if pc.AvailabilityDomain != nil {
					pcMap["AvailabilityDomain"] = *pc.AvailabilityDomain
				}
				if pc.SubnetId != nil {
					pcMap["SubnetId"] = *pc.SubnetId
				}
				if pc.CapacityReservationId != nil {
					pcMap["CapacityReservationId"] = *pc.CapacityReservationId
				}
				if len(pc.FaultDomains) > 0 {
					pcMap["FaultDomains"] = pc.FaultDomains
				}
				placementConfigs = append(placementConfigs, pcMap)
			}
			nodeConfig["PlacementConfigs"] = placementConfigs
		}

		props["NodeConfigDetails"] = nodeConfig
	}

	// InitialNodeLabels
	if len(resp.InitialNodeLabels) > 0 {
		labels := make([]map[string]any, 0, len(resp.InitialNodeLabels))
		for _, label := range resp.InitialNodeLabels {
			labelMap := map[string]any{}
			if label.Key != nil {
				labelMap["Key"] = *label.Key
			}
			if label.Value != nil {
				labelMap["Value"] = *label.Value
			}
			labels = append(labels, labelMap)
		}
		props["InitialNodeLabels"] = labels
	}

	if resp.SshPublicKey != nil {
		props["SshPublicKey"] = *resp.SshPublicKey
	}
	if resp.FreeformTags != nil {
		props["FreeformTags"] = resp.FreeformTags
	}
	if resp.DefinedTags != nil {
		props["DefinedTags"] = resp.DefinedTags
	}

	propBytes, err := json.Marshal(props)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal NodePool properties: %w", err)
	}

	return &resource.ReadResult{
		ResourceType: "OCI::ContainerEngine::NodePool",
		Properties:   string(propBytes),
	}, nil
}

func (p *NodePoolProvisioner) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
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
		return nil, fmt.Errorf("CompartmentId is required for listing NodePools (either directly or derived from ClusterId)")
	}

	listReq := containerengine.ListNodePoolsRequest{
		CompartmentId: common.String(compartmentId),
		// Filter out deleted/deleting/failed node pools - only return active or in-progress states
		LifecycleState: []containerengine.NodePoolLifecycleStateEnum{
			containerengine.NodePoolLifecycleStateCreating,
			containerengine.NodePoolLifecycleStateActive,
			containerengine.NodePoolLifecycleStateUpdating,
			containerengine.NodePoolLifecycleStateInactive,
			containerengine.NodePoolLifecycleStateNeedsAttention,
		},
	}

	// Filter by ClusterId if provided
	if clusterId != "" {
		listReq.ClusterId = common.String(clusterId)
	}

	resp, err := client.ListNodePools(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list NodePools: %w", err)
	}

	nativeIDs := make([]string, 0, len(resp.Items))
	for _, nodePool := range resp.Items {
		nativeIDs = append(nativeIDs, *nodePool.Id)
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

