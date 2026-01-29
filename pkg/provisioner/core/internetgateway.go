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

type InternetGatewayProvisioner struct {
	clients *client.Clients
}

var _ provisioner.Provisioner = &InternetGatewayProvisioner{}

func init() {
	provisioner.Register("OCI::Core::InternetGateway", NewInternetGatewayProvisioner)
}

func NewInternetGatewayProvisioner(clients *client.Clients) provisioner.Provisioner {
	return &InternetGatewayProvisioner{clients: clients}
}

func (p *InternetGatewayProvisioner) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	var props map[string]any
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	createDetails := core.CreateInternetGatewayDetails{
		CompartmentId: common.String(props["CompartmentId"].(string)),
		VcnId:         common.String(props["VcnId"].(string)),
		IsEnabled:     common.Bool(props["IsEnabled"].(bool)),
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

	createReq := core.CreateInternetGatewayRequest{
		CreateInternetGatewayDetails: createDetails,
	}

	resp, err := client.CreateInternetGateway(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create InternetGateway: %w", err)
	}

	// Build properties from Create response
	properties := map[string]any{
		"CompartmentId": *resp.CompartmentId,
		"VcnId":         *resp.VcnId,
		"Id":            *resp.Id,
		"IsEnabled":     *resp.IsEnabled,
	}

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

func (p *InternetGatewayProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	props, err := util.ApplyPatchDocument(ctx, request, p.Read)
	if err != nil {
		return nil, err
	}

	updateDetails := core.UpdateInternetGatewayDetails{}

	if isEnabled, ok := util.ExtractBool(props, "IsEnabled"); ok {
		updateDetails.IsEnabled = common.Bool(isEnabled)
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

	updateReq := core.UpdateInternetGatewayRequest{
		IgId:                         common.String(request.NativeID),
		UpdateInternetGatewayDetails: updateDetails,
	}

	resp, err := client.UpdateInternetGateway(ctx, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update InternetGateway: %w", err)
	}

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationUpdate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        *resp.Id,
		},
	}, nil
}

func (p *InternetGatewayProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
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
		return nil, fmt.Errorf("failed to read InternetGateway before delete: %w", err)
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

	deleteReq := core.DeleteInternetGatewayRequest{
		IgId: common.String(request.NativeID),
	}

	_, err = client.DeleteInternetGateway(ctx, deleteReq)
	if err != nil {
		return nil, fmt.Errorf("failed to delete InternetGateway: %w", err)
	}

	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationDelete,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (p *InternetGatewayProvisioner) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			RequestID:       request.RequestID,
		},
	}, nil
}

func (p *InternetGatewayProvisioner) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	getReq := core.GetInternetGatewayRequest{
		IgId: common.String(request.NativeID),
	}

	resp, err := client.GetInternetGateway(ctx, getReq)
	if err != nil {
		if serviceErr, ok := common.IsServiceError(err); ok && serviceErr.GetHTTPStatusCode() == 404 {
			return &resource.ReadResult{
				ResourceType: "OCI::Core::InternetGateway",
				ErrorCode:    resource.OperationErrorCodeNotFound,
			}, nil
		}
		return nil, fmt.Errorf("failed to read InternetGateway: %w", err)
	}

	props := map[string]any{
		"CompartmentId": *resp.CompartmentId,
		"VcnId":         *resp.VcnId,
		"Id":            *resp.Id,
		"IsEnabled":     *resp.IsEnabled,
	}

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
		return nil, fmt.Errorf("failed to marshal InternetGateway properties: %w", err)
	}

	return &resource.ReadResult{
		ResourceType: "OCI::Core::InternetGateway",
		Properties:   string(propBytes),
	}, nil
}

func (p *InternetGatewayProvisioner) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	compartmentId, ok := request.AdditionalProperties["CompartmentId"]
	if !ok {
		return nil, fmt.Errorf("CompartmentId is required for listing InternetGateways")
	}

	listReq := core.ListInternetGatewaysRequest{
		CompartmentId: common.String(compartmentId),
	}

	// Optional: Filter by VcnId
	if vcnId, ok := request.AdditionalProperties["VcnId"]; ok {
		listReq.VcnId = common.String(vcnId)
	}

	resp, err := client.ListInternetGateways(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list InternetGateways: %w", err)
	}

	nativeIDs := make([]string, 0, len(resp.Items))
	for _, ig := range resp.Items {
		nativeIDs = append(nativeIDs, *ig.Id)
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}
