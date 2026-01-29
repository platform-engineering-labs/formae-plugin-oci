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

type ServiceGatewayProvisioner struct {
	clients *client.Clients
}

var _ provisioner.Provisioner = &ServiceGatewayProvisioner{}

func init() {
	provisioner.Register("OCI::Core::ServiceGateway", NewServiceGatewayProvisioner)
}

func NewServiceGatewayProvisioner(clients *client.Clients) provisioner.Provisioner {
	return &ServiceGatewayProvisioner{clients: clients}
}

func (p *ServiceGatewayProvisioner) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	var props map[string]any
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	// Extract services array
	services, ok := props["Services"].([]any)
	if !ok {
		return nil, fmt.Errorf("services is required and must be an array")
	}

	serviceList := make([]core.ServiceIdRequestDetails, 0, len(services))
	for _, svc := range services {
		svcMap, ok := svc.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("each service must be an object with serviceId")
		}
		// ServiceGatewayService is a plain class, so fields stay camelCase (not transformed)
		// Try both camelCase and PascalCase for compatibility
		serviceId, ok := svcMap["serviceId"].(string)
		if !ok {
			if serviceIdUpper, okUpper := svcMap["ServiceId"].(string); okUpper {
				serviceId = serviceIdUpper
				ok = true
			}
		}
		if !ok {
			return nil, fmt.Errorf("serviceId is required for each service")
		}
		serviceList = append(serviceList, core.ServiceIdRequestDetails{
			ServiceId: common.String(serviceId),
		})
	}

	createDetails := core.CreateServiceGatewayDetails{
		CompartmentId: common.String(props["CompartmentId"].(string)),
		VcnId:         common.String(props["VcnId"].(string)),
		Services:      serviceList,
	}

	if displayName, ok := util.ExtractString(props, "DisplayName"); ok {
		createDetails.DisplayName = common.String(displayName)
	}
	if freeformTags, ok := util.ExtractFreeformTags(props, "FreeformTags"); ok {
		createDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractDefinedTags(props, "DefinedTags"); ok {
		createDetails.DefinedTags = definedTags
	}

	createReq := core.CreateServiceGatewayRequest{
		CreateServiceGatewayDetails: createDetails,
	}

	resp, err := client.CreateServiceGateway(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create ServiceGateway: %w", err)
	}

	// Build properties from Create response
	properties := map[string]any{
		"CompartmentId": *resp.CompartmentId,
		"VcnId":         *resp.VcnId,
		"Id":            *resp.Id,
	}

	// Convert services to array of maps
	servicesArray := make([]map[string]string, 0, len(resp.Services))
	for _, svc := range resp.Services {
		servicesArray = append(servicesArray, map[string]string{
			"serviceId": *svc.ServiceId,
		})
	}
	properties["Services"] = servicesArray

	if resp.DisplayName != nil {
		properties["DisplayName"] = *resp.DisplayName
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
			NativeID:           *resp.Id,
			ResourceProperties: json.RawMessage(propertiesBytes),
		},
	}, nil
}

func (p *ServiceGatewayProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	props, err := util.ApplyPatchDocument(ctx, request, p.Read)
	if err != nil {
		return nil, err
	}

	updateDetails := core.UpdateServiceGatewayDetails{}

	// Services can be updated
	if services, ok := props["Services"].([]any); ok {
		serviceList := make([]core.ServiceIdRequestDetails, 0, len(services))
		for _, svc := range services {
			svcMap, ok := svc.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("each service must be an object with serviceId")
			}
			// ServiceGatewayService is a plain class, so fields stay camelCase
			// Try both camelCase and PascalCase for compatibility
			serviceId, ok := svcMap["serviceId"].(string)
			if !ok {
				if serviceIdUpper, okUpper := svcMap["ServiceId"].(string); okUpper {
					serviceId = serviceIdUpper
					ok = true
				}
			}
			if !ok {
				return nil, fmt.Errorf("serviceId is required for each service")
			}
			serviceList = append(serviceList, core.ServiceIdRequestDetails{
				ServiceId: common.String(serviceId),
			})
		}
		updateDetails.Services = serviceList
	}

	if displayName, ok := util.ExtractString(props, "DisplayName"); ok {
		updateDetails.DisplayName = common.String(displayName)
	}
	if freeformTags, ok := util.ExtractFreeformTags(props, "FreeformTags"); ok {
		updateDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractDefinedTags(props, "DefinedTags"); ok {
		updateDetails.DefinedTags = definedTags
	}

	updateReq := core.UpdateServiceGatewayRequest{
		ServiceGatewayId:            common.String(request.NativeID),
		UpdateServiceGatewayDetails: updateDetails,
	}

	resp, err := client.UpdateServiceGateway(ctx, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update ServiceGateway: %w", err)
	}

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationUpdate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        *resp.Id,
		},
	}, nil
}

func (p *ServiceGatewayProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	// Check if exists
	readReq := &resource.ReadRequest{
		NativeID: request.NativeID,
	}
	readRes, err := p.Read(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to read ServiceGateway before delete: %w", err)
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

	deleteReq := core.DeleteServiceGatewayRequest{
		ServiceGatewayId: common.String(request.NativeID),
	}

	_, err = client.DeleteServiceGateway(ctx, deleteReq)
	if err != nil {
		return nil, fmt.Errorf("failed to delete ServiceGateway: %w", err)
	}

	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationDelete,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (p *ServiceGatewayProvisioner) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			RequestID:       request.RequestID,
		},
	}, nil
}

func (p *ServiceGatewayProvisioner) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	getReq := core.GetServiceGatewayRequest{
		ServiceGatewayId: common.String(request.NativeID),
	}

	resp, err := client.GetServiceGateway(ctx, getReq)
	if err != nil {
		if serviceErr, ok := common.IsServiceError(err); ok && serviceErr.GetHTTPStatusCode() == 404 {
			return &resource.ReadResult{
				ResourceType: "OCI::Core::ServiceGateway",
				ErrorCode:    resource.OperationErrorCodeNotFound,
			}, nil
		}
		return nil, fmt.Errorf("failed to read ServiceGateway: %w", err)
	}

	props := map[string]any{
		"CompartmentId": *resp.CompartmentId,
		"VcnId":         *resp.VcnId,
		"Id":            *resp.Id,
	}

	// Convert services to array of maps
	servicesArray := make([]map[string]string, 0, len(resp.Services))
	for _, svc := range resp.Services {
		servicesArray = append(servicesArray, map[string]string{
			"serviceId": *svc.ServiceId,
		})
	}
	props["Services"] = servicesArray

	if resp.DisplayName != nil {
		props["DisplayName"] = *resp.DisplayName
	}
	if resp.FreeformTags != nil {
		props["FreeformTags"] = util.FreeformTagsToList(resp.FreeformTags)
	}
	if resp.DefinedTags != nil {
		props["DefinedTags"] = util.DefinedTagsToList(resp.DefinedTags)
	}

	propBytes, err := json.Marshal(props)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ServiceGateway properties: %w", err)
	}

	return &resource.ReadResult{
		ResourceType: "OCI::Core::ServiceGateway",
		Properties:   string(propBytes),
	}, nil
}

func (p *ServiceGatewayProvisioner) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	compartmentId, ok := request.AdditionalProperties["CompartmentId"]
	if !ok {
		return nil, fmt.Errorf("CompartmentId is required for listing ServiceGateways")
	}

	listReq := core.ListServiceGatewaysRequest{
		CompartmentId: common.String(compartmentId),
	}

	// Optional: Filter by VcnId
	if vcnId, ok := request.AdditionalProperties["VcnId"]; ok {
		listReq.VcnId = common.String(vcnId)
	}

	resp, err := client.ListServiceGateways(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list ServiceGateways: %w", err)
	}

	nativeIDs := make([]string, 0, len(resp.Items))
	for _, sg := range resp.Items {
		nativeIDs = append(nativeIDs, *sg.Id)
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}
