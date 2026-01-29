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

type NatGatewayProvisioner struct {
	clients *client.Clients
}

var _ provisioner.Provisioner = &NatGatewayProvisioner{}

func init() {
	provisioner.Register("OCI::Core::NatGateway", NewNatGatewayProvisioner)
}

func NewNatGatewayProvisioner(clients *client.Clients) provisioner.Provisioner {
	return &NatGatewayProvisioner{clients: clients}
}

func (p *NatGatewayProvisioner) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	var props map[string]any
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	createDetails := core.CreateNatGatewayDetails{
		CompartmentId: common.String(props["CompartmentId"].(string)),
		VcnId:         common.String(props["VcnId"].(string)),
	}

	if displayName, ok := util.ExtractString(props, "DisplayName"); ok {
		createDetails.DisplayName = common.String(displayName)
	}
	if blockTraffic, ok := util.ExtractBool(props, "BlockTraffic"); ok {
		createDetails.BlockTraffic = common.Bool(blockTraffic)
	}
	if freeformTags, ok := util.ExtractFreeformTags(props, "FreeformTags"); ok {
		createDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractDefinedTags(props, "DefinedTags"); ok {
		createDetails.DefinedTags = definedTags
	}

	createReq := core.CreateNatGatewayRequest{
		CreateNatGatewayDetails: createDetails,
	}

	resp, err := client.CreateNatGateway(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create NatGateway: %w", err)
	}

	// Build properties from Create response
	properties := map[string]any{
		"CompartmentId": *resp.CompartmentId,
		"VcnId":         *resp.VcnId,
		"Id":            *resp.Id,
	}

	if resp.DisplayName != nil {
		properties["DisplayName"] = *resp.DisplayName
	}
	if resp.BlockTraffic != nil {
		properties["BlockTraffic"] = *resp.BlockTraffic
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

func (p *NatGatewayProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	props, err := util.ApplyPatchDocument(ctx, request, p.Read)
	if err != nil {
		return nil, err
	}

	updateDetails := core.UpdateNatGatewayDetails{}

	if blockTraffic, ok := util.ExtractBool(props, "BlockTraffic"); ok {
		updateDetails.BlockTraffic = common.Bool(blockTraffic)
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

	updateReq := core.UpdateNatGatewayRequest{
		NatGatewayId:            common.String(request.NativeID),
		UpdateNatGatewayDetails: updateDetails,
	}

	resp, err := client.UpdateNatGateway(ctx, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update NatGateway: %w", err)
	}

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationUpdate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        *resp.Id,
		},
	}, nil
}

func (p *NatGatewayProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
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
		return nil, fmt.Errorf("failed to read NatGateway before delete: %w", err)
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

	deleteReq := core.DeleteNatGatewayRequest{
		NatGatewayId: common.String(request.NativeID),
	}

	_, err = client.DeleteNatGateway(ctx, deleteReq)
	if err != nil {
		return nil, fmt.Errorf("failed to delete NatGateway: %w", err)
	}

	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationDelete,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (p *NatGatewayProvisioner) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			RequestID:       request.RequestID,
		},
	}, nil
}

func (p *NatGatewayProvisioner) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	getReq := core.GetNatGatewayRequest{
		NatGatewayId: common.String(request.NativeID),
	}

	resp, err := client.GetNatGateway(ctx, getReq)
	if err != nil {
		if serviceErr, ok := common.IsServiceError(err); ok && serviceErr.GetHTTPStatusCode() == 404 {
			return &resource.ReadResult{
				ResourceType: "OCI::Core::NatGateway",
				ErrorCode:    resource.OperationErrorCodeNotFound,
			}, nil
		}
		return nil, fmt.Errorf("failed to read NatGateway: %w", err)
	}

	props := map[string]any{
		"CompartmentId": *resp.CompartmentId,
		"VcnId":         *resp.VcnId,
		"Id":            *resp.Id,
	}

	if resp.DisplayName != nil {
		props["DisplayName"] = *resp.DisplayName
	}
	if resp.BlockTraffic != nil {
		props["BlockTraffic"] = *resp.BlockTraffic
	}
	if resp.FreeformTags != nil {
		props["FreeformTags"] = util.FreeformTagsToList(resp.FreeformTags)
	}
	if resp.DefinedTags != nil {
		props["DefinedTags"] = util.DefinedTagsToList(resp.DefinedTags)
	}

	propBytes, err := json.Marshal(props)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal NatGateway properties: %w", err)
	}

	return &resource.ReadResult{
		ResourceType: "OCI::Core::NatGateway",
		Properties:   string(propBytes),
	}, nil
}

func (p *NatGatewayProvisioner) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	compartmentId, ok := request.AdditionalProperties["CompartmentId"]
	if !ok {
		return nil, fmt.Errorf("CompartmentId is required for listing NatGateways")
	}

	listReq := core.ListNatGatewaysRequest{
		CompartmentId: common.String(compartmentId),
	}

	// Optional: Filter by VcnId
	if vcnId, ok := request.AdditionalProperties["VcnId"]; ok {
		listReq.VcnId = common.String(vcnId)
	}

	resp, err := client.ListNatGateways(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list NatGateways: %w", err)
	}

	nativeIDs := make([]string, 0, len(resp.Items))
	for _, ng := range resp.Items {
		nativeIDs = append(nativeIDs, *ng.Id)
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}
