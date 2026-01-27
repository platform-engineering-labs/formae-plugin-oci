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

type ClusterProvisioner struct {
	clients *client.Clients
}

var _ provisioner.Provisioner = &ClusterProvisioner{}

func init() {
	provisioner.Register("OCI::ContainerEngine::Cluster", NewClusterProvisioner)
}

func NewClusterProvisioner(clients *client.Clients) provisioner.Provisioner {
	return &ClusterProvisioner{clients: clients}
}

func (p *ClusterProvisioner) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	client, err := p.clients.GetContainerEngineClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ContainerEngine client: %w", err)
	}

	var props map[string]any
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	// Extract required properties - handle both direct strings and resolved references
	compartmentId, _ := util.ExtractString(props, "CompartmentId")
	vcnId, _ := util.ExtractString(props, "VcnId")
	k8sVersion, _ := util.ExtractString(props, "KubernetesVersion")

	createDetails := containerengine.CreateClusterDetails{
		CompartmentId:     common.String(compartmentId),
		VcnId:             common.String(vcnId),
		KubernetesVersion: common.String(k8sVersion),
	}

	if name, ok := util.ExtractString(props, "Name"); ok {
		createDetails.Name = common.String(name)
	}

	if clusterType, ok := util.ExtractString(props, "ClusterType"); ok {
		createDetails.Type = containerengine.ClusterTypeEnum(clusterType)
	}

	// Parse EndpointConfig (nested class fields stay camelCase)
	if endpointConfig, ok := props["EndpointConfig"].(map[string]any); ok {
		config := &containerengine.CreateClusterEndpointConfigDetails{}
		if subnetId, ok := util.ExtractString(endpointConfig, "subnetId"); ok {
			config.SubnetId = common.String(subnetId)
		} else if subnetId, ok := util.ExtractString(endpointConfig, "SubnetId"); ok {
			config.SubnetId = common.String(subnetId)
		}
		if isPublicIpEnabled, ok := util.ExtractBool(endpointConfig, "isPublicIpEnabled"); ok {
			config.IsPublicIpEnabled = common.Bool(isPublicIpEnabled)
		} else if isPublicIpEnabled, ok := util.ExtractBool(endpointConfig, "IsPublicIpEnabled"); ok {
			config.IsPublicIpEnabled = common.Bool(isPublicIpEnabled)
		}
		if nsgIds, ok := util.ExtractStringSlice(endpointConfig, "nsgIds"); ok {
			config.NsgIds = nsgIds
		} else if nsgIds, ok := util.ExtractStringSlice(endpointConfig, "NsgIds"); ok {
			config.NsgIds = nsgIds
		}
		createDetails.EndpointConfig = config
	}

	// Parse Options (nested class fields stay camelCase - no SubResourceHint)
	if options, ok := props["Options"].(map[string]any); ok {
		clusterOptions := &containerengine.ClusterCreateOptions{}

		if serviceLbSubnetIds, ok := util.ExtractStringSlice(options, "serviceLbSubnetIds"); ok {
			clusterOptions.ServiceLbSubnetIds = serviceLbSubnetIds
		}

		if kubernetesNetworkConfig, ok := options["kubernetesNetworkConfig"].(map[string]any); ok {
			networkConfig := &containerengine.KubernetesNetworkConfig{}
			if podsCidr, ok := util.ExtractString(kubernetesNetworkConfig, "podsCidr"); ok {
				networkConfig.PodsCidr = common.String(podsCidr)
			}
			if servicesCidr, ok := util.ExtractString(kubernetesNetworkConfig, "servicesCidr"); ok {
				networkConfig.ServicesCidr = common.String(servicesCidr)
			}
			clusterOptions.KubernetesNetworkConfig = networkConfig
		}

		if addOns, ok := options["addOns"].(map[string]any); ok {
			addOnOptions := &containerengine.AddOnOptions{}
			if isKubernetesDashboardEnabled, ok := util.ExtractBool(addOns, "isKubernetesDashboardEnabled"); ok {
				addOnOptions.IsKubernetesDashboardEnabled = common.Bool(isKubernetesDashboardEnabled)
			}
			if isTillerEnabled, ok := util.ExtractBool(addOns, "isTillerEnabled"); ok {
				addOnOptions.IsTillerEnabled = common.Bool(isTillerEnabled)
			}
			clusterOptions.AddOns = addOnOptions
		}

		if admissionControllerOptions, ok := options["admissionControllerOptions"].(map[string]any); ok {
			admissionOptions := &containerengine.AdmissionControllerOptions{}
			if isPodSecurityPolicyEnabled, ok := util.ExtractBool(admissionControllerOptions, "isPodSecurityPolicyEnabled"); ok {
				admissionOptions.IsPodSecurityPolicyEnabled = common.Bool(isPodSecurityPolicyEnabled)
			}
			clusterOptions.AdmissionControllerOptions = admissionOptions
		}

		createDetails.Options = clusterOptions
	}

	if freeformTags, ok := util.ExtractTag(props, "FreeformTags"); ok {
		createDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractNestedTag(props, "DefinedTags"); ok {
		createDetails.DefinedTags = definedTags
	}

	createReq := containerengine.CreateClusterRequest{
		CreateClusterDetails: createDetails,
	}

	resp, err := client.CreateCluster(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cluster: %w", err)
	}

	// Cluster creation is async - return in-progress with WorkRequest ID
	return &resource.CreateResult{
		ProgressResult: CreateInProgressResult(resource.OperationCreate, *resp.OpcWorkRequestId),
	}, nil
}

func (p *ClusterProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	client, err := p.clients.GetContainerEngineClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ContainerEngine client: %w", err)
	}

	props, err := util.ApplyPatchDocument(ctx, request, p.Read)
	if err != nil {
		return nil, err
	}

	updateDetails := containerengine.UpdateClusterDetails{}

	if name, ok := util.ExtractString(props, "Name"); ok {
		updateDetails.Name = common.String(name)
	}
	if kubernetesVersion, ok := util.ExtractString(props, "KubernetesVersion"); ok {
		updateDetails.KubernetesVersion = common.String(kubernetesVersion)
	}
	if freeformTags, ok := util.ExtractTag(props, "FreeformTags"); ok {
		updateDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractNestedTag(props, "DefinedTags"); ok {
		updateDetails.DefinedTags = definedTags
	}

	// Parse Options update
	if options, ok := props["Options"].(map[string]any); ok {
		clusterOptions := &containerengine.UpdateClusterOptionsDetails{}

		if admissionControllerOptions, ok := options["AdmissionControllerOptions"].(map[string]any); ok {
			admissionOptions := &containerengine.AdmissionControllerOptions{}
			if isPodSecurityPolicyEnabled, ok := util.ExtractBool(admissionControllerOptions, "IsPodSecurityPolicyEnabled"); ok {
				admissionOptions.IsPodSecurityPolicyEnabled = common.Bool(isPodSecurityPolicyEnabled)
			}
			clusterOptions.AdmissionControllerOptions = admissionOptions
		}

		updateDetails.Options = clusterOptions
	}

	updateReq := containerengine.UpdateClusterRequest{
		ClusterId:            common.String(request.NativeID),
		UpdateClusterDetails: updateDetails,
	}

	resp, err := client.UpdateCluster(ctx, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update Cluster: %w", err)
	}

	// Update is async - return in-progress with WorkRequest ID
	return &resource.UpdateResult{
		ProgressResult: CreateInProgressResult(resource.OperationUpdate, *resp.OpcWorkRequestId),
	}, nil
}

func (p *ClusterProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	client, err := p.clients.GetContainerEngineClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ContainerEngine client: %w", err)
	}

	// Check if Cluster exists before attempting delete
	readReq := &resource.ReadRequest{
		NativeID: request.NativeID,
	}
	readRes, err := p.Read(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to read Cluster before delete: %w", err)
	}
	if readRes.ErrorCode == resource.OperationErrorCodeNotFound {
		// Cluster already deleted
		return &resource.DeleteResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationDelete,
				OperationStatus: resource.OperationStatusSuccess,
				NativeID:        request.NativeID,
			},
		}, nil
	}

	deleteReq := containerengine.DeleteClusterRequest{
		ClusterId: common.String(request.NativeID),
	}

	resp, err := client.DeleteCluster(ctx, deleteReq)
	if err != nil {
		return nil, fmt.Errorf("failed to delete Cluster: %w", err)
	}

	// Delete is async - return in-progress with WorkRequest ID
	return &resource.DeleteResult{
		ProgressResult: CreateInProgressResult(resource.OperationDelete, *resp.OpcWorkRequestId),
	}, nil
}

func (p *ClusterProvisioner) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
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

func (p *ClusterProvisioner) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	client, err := p.clients.GetContainerEngineClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ContainerEngine client: %w", err)
	}

	getReq := containerengine.GetClusterRequest{
		ClusterId: common.String(request.NativeID),
	}

	resp, err := client.GetCluster(ctx, getReq)
	if err != nil {
		// Check if not found
		if serviceErr, ok := common.IsServiceError(err); ok && serviceErr.GetHTTPStatusCode() == 404 {
			return &resource.ReadResult{
				ResourceType: "OCI::ContainerEngine::Cluster",
				ErrorCode:    resource.OperationErrorCodeNotFound,
			}, nil
		}
		return nil, fmt.Errorf("failed to read Cluster: %w", err)
	}

	// Build properties map
	props := map[string]any{
		"CompartmentId":     *resp.CompartmentId,
		"Id":                *resp.Id,
		"VcnId":             *resp.VcnId,
		"KubernetesVersion": *resp.KubernetesVersion,
	}

	if resp.Name != nil {
		props["Name"] = *resp.Name
	}
	if resp.Type != "" {
		props["Type"] = string(resp.Type)
	}
	if resp.LifecycleState != "" {
		props["LifecycleState"] = string(resp.LifecycleState)
	}
	if resp.Endpoints != nil {
		endpoints := map[string]any{}
		if resp.Endpoints.Kubernetes != nil {
			endpoints["Kubernetes"] = *resp.Endpoints.Kubernetes
		}
		if resp.Endpoints.PublicEndpoint != nil {
			endpoints["PublicEndpoint"] = *resp.Endpoints.PublicEndpoint
		}
		if resp.Endpoints.PrivateEndpoint != nil {
			endpoints["PrivateEndpoint"] = *resp.Endpoints.PrivateEndpoint
		}
		if len(endpoints) > 0 {
			props["Endpoints"] = endpoints
		}
	}

	// EndpointConfig - required for patches to work
	if resp.EndpointConfig != nil {
		endpointConfig := map[string]any{}
		if resp.EndpointConfig.SubnetId != nil {
			endpointConfig["subnetId"] = *resp.EndpointConfig.SubnetId
		}
		if resp.EndpointConfig.IsPublicIpEnabled != nil {
			endpointConfig["isPublicIpEnabled"] = *resp.EndpointConfig.IsPublicIpEnabled
		}
		if len(resp.EndpointConfig.NsgIds) > 0 {
			endpointConfig["nsgIds"] = resp.EndpointConfig.NsgIds
		}
		if len(endpointConfig) > 0 {
			props["EndpointConfig"] = endpointConfig
		}
	}

	// Options - required for patches to work
	if resp.Options != nil {
		options := map[string]any{}
		if len(resp.Options.ServiceLbSubnetIds) > 0 {
			options["serviceLbSubnetIds"] = resp.Options.ServiceLbSubnetIds
		}
		if resp.Options.KubernetesNetworkConfig != nil {
			networkConfig := map[string]any{}
			if resp.Options.KubernetesNetworkConfig.PodsCidr != nil {
				networkConfig["podsCidr"] = *resp.Options.KubernetesNetworkConfig.PodsCidr
			}
			if resp.Options.KubernetesNetworkConfig.ServicesCidr != nil {
				networkConfig["servicesCidr"] = *resp.Options.KubernetesNetworkConfig.ServicesCidr
			}
			if len(networkConfig) > 0 {
				options["kubernetesNetworkConfig"] = networkConfig
			}
		}
		if resp.Options.AddOns != nil {
			addOns := map[string]any{}
			if resp.Options.AddOns.IsKubernetesDashboardEnabled != nil {
				addOns["isKubernetesDashboardEnabled"] = *resp.Options.AddOns.IsKubernetesDashboardEnabled
			}
			if resp.Options.AddOns.IsTillerEnabled != nil {
				addOns["isTillerEnabled"] = *resp.Options.AddOns.IsTillerEnabled
			}
			if len(addOns) > 0 {
				options["addOns"] = addOns
			}
		}
		if len(options) > 0 {
			props["Options"] = options
		}
	}

	if resp.FreeformTags != nil {
		props["FreeformTags"] = resp.FreeformTags
	}
	if resp.DefinedTags != nil {
		props["DefinedTags"] = resp.DefinedTags
	}

	propBytes, err := json.Marshal(props)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Cluster properties: %w", err)
	}

	return &resource.ReadResult{
		ResourceType: "OCI::ContainerEngine::Cluster",
		Properties:   string(propBytes),
	}, nil
}

func (p *ClusterProvisioner) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	client, err := p.clients.GetContainerEngineClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get ContainerEngine client: %w", err)
	}

	// CompartmentId is required for listing
	compartmentId, ok := request.AdditionalProperties["CompartmentId"]
	if !ok {
		return nil, fmt.Errorf("CompartmentId is required for listing Clusters")
	}

	listReq := containerengine.ListClustersRequest{
		CompartmentId: common.String(compartmentId),
		// Filter out deleted/deleting/failed clusters - only return active or in-progress states
		LifecycleState: []containerengine.ClusterLifecycleStateEnum{
			containerengine.ClusterLifecycleStateCreating,
			containerengine.ClusterLifecycleStateActive,
			containerengine.ClusterLifecycleStateUpdating,
		},
	}

	resp, err := client.ListClusters(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list Clusters: %w", err)
	}

	nativeIDs := make([]string, 0, len(resp.Items))
	for _, cluster := range resp.Items {
		nativeIDs = append(nativeIDs, *cluster.Id)
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

