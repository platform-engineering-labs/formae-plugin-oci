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
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/client"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/util"
)

type VCNProvisioner struct {
	clients *client.Clients
}

var _ provisioner.Provisioner = &VCNProvisioner{}

func init() {
	provisioner.Register("OCI::Core::VCN", NewVCNProvisioner)
}

func NewVCNProvisioner(clients *client.Clients) provisioner.Provisioner {
	return &VCNProvisioner{clients: clients}
}

func (p *VCNProvisioner) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	var props map[string]any
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	createDetails := core.CreateVcnDetails{
		CompartmentId: common.String(props["CompartmentId"].(string)),
	}

	if cidrBlock, ok := util.ExtractString(props, "CidrBlock"); ok {
		createDetails.CidrBlock = common.String(cidrBlock)
	}
	if cidrBlocks, ok := util.ExtractStringSlice(props, "CidrBlocks"); ok {
		createDetails.CidrBlocks = cidrBlocks
	}
	if displayName, ok := util.ExtractString(props, "DisplayName"); ok {
		createDetails.DisplayName = common.String(displayName)
	}
	if dnsLabel, ok := util.ExtractString(props, "DnsLabel"); ok {
		createDetails.DnsLabel = common.String(dnsLabel)
	}
	if isIpv6Enabled, ok := util.ExtractBool(props, "IsIpv6Enabled"); ok {
		createDetails.IsIpv6Enabled = common.Bool(isIpv6Enabled)
	}
	if freeformTags, ok := util.ExtractTag(props, "FreeformTags"); ok {
		createDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractNestedTag(props, "DefinedTags"); ok {
		createDetails.DefinedTags = definedTags
	}

	createReq := core.CreateVcnRequest{
		CreateVcnDetails: createDetails,
	}

	resp, err := client.CreateVcn(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create VCN: %w", err)
	}

	// Build properties from Create response
	properties := map[string]any{
		"CompartmentId": *resp.CompartmentId,
		"Id":            *resp.Id,
	}

	if resp.CidrBlock != nil {
		properties["CidrBlock"] = *resp.CidrBlock
	}
	if resp.CidrBlocks != nil {
		properties["CidrBlocks"] = resp.CidrBlocks
	}
	if resp.DisplayName != nil {
		properties["DisplayName"] = *resp.DisplayName
	}
	if resp.DnsLabel != nil {
		properties["DnsLabel"] = *resp.DnsLabel
	}
	if resp.DefaultDhcpOptionsId != nil {
		properties["DefaultDhcpOptionsId"] = *resp.DefaultDhcpOptionsId
	}
	if resp.DefaultRouteTableId != nil {
		properties["DefaultRouteTableId"] = *resp.DefaultRouteTableId
	}
	if resp.DefaultSecurityListId != nil {
		properties["DefaultSecurityListId"] = *resp.DefaultSecurityListId
	}
	if resp.FreeformTags != nil {
		properties["FreeformTags"] = resp.FreeformTags
	}
	if resp.DefinedTags != nil {
		properties["DefinedTags"] = resp.DefinedTags
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

func (p *VCNProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	props, err := util.ApplyPatchDocument(ctx, request, p.Read)
	if err != nil {
		return nil, err
	}

	updateDetails := core.UpdateVcnDetails{}

	if displayName, ok := util.ExtractString(props, "DisplayName"); ok {
		updateDetails.DisplayName = common.String(displayName)
	}
	if freeformTags, ok := util.ExtractTag(props, "FreeformTags"); ok {
		updateDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractNestedTag(props, "DefinedTags"); ok {
		updateDetails.DefinedTags = definedTags
	}

	updateReq := core.UpdateVcnRequest{
		VcnId:            common.String(request.NativeID),
		UpdateVcnDetails: updateDetails,
	}

	resp, err := client.UpdateVcn(ctx, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update VCN: %w", err)
	}

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationUpdate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        *resp.Id,
		},
	}, nil
}

func (p *VCNProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	// Check if VCN exists before attempting delete
	readReq := &resource.ReadRequest{
		NativeID: request.NativeID,
	}
	readRes, err := p.Read(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to read VCN before delete: %w", err)
	}
	if readRes.ErrorCode == resource.OperationErrorCodeNotFound {
		// VCN already deleted
		return &resource.DeleteResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationDelete,
				OperationStatus: resource.OperationStatusSuccess,
				NativeID:        request.NativeID,
			},
		}, nil
	}

	deleteReq := core.DeleteVcnRequest{
		VcnId: common.String(request.NativeID),
	}

	_, err = client.DeleteVcn(ctx, deleteReq)
	if err != nil {
		return nil, fmt.Errorf("failed to delete VCN: %w", err)
	}

	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationDelete,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (p *VCNProvisioner) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	// VCN operations are synchronous, no status check needed
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			RequestID:       request.RequestID,
		},
	}, nil
}

func (p *VCNProvisioner) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	getReq := core.GetVcnRequest{
		VcnId: common.String(request.NativeID),
	}

	resp, err := client.GetVcn(ctx, getReq)
	if err != nil {
		// Check if not found
		if serviceErr, ok := common.IsServiceError(err); ok && serviceErr.GetHTTPStatusCode() == 404 {
			return &resource.ReadResult{
				ResourceType: "OCI::Core::VCN",
				ErrorCode:    resource.OperationErrorCodeNotFound,
			}, nil
		}
		return nil, fmt.Errorf("failed to read VCN: %w", err)
	}

	// Build properties map
	props := map[string]any{
		"CompartmentId": *resp.CompartmentId,
		"Id":            *resp.Id,
	}

	if resp.CidrBlock != nil {
		props["CidrBlock"] = *resp.CidrBlock
	}
	if resp.CidrBlocks != nil {
		props["CidrBlocks"] = resp.CidrBlocks
	}
	if resp.DisplayName != nil {
		props["DisplayName"] = *resp.DisplayName
	}
	if resp.DnsLabel != nil {
		props["DnsLabel"] = *resp.DnsLabel
	}
	if resp.DefaultDhcpOptionsId != nil {
		props["DefaultDhcpOptionsId"] = *resp.DefaultDhcpOptionsId
	}
	if resp.DefaultRouteTableId != nil {
		props["DefaultRouteTableId"] = *resp.DefaultRouteTableId
	}
	if resp.DefaultSecurityListId != nil {
		props["DefaultSecurityListId"] = *resp.DefaultSecurityListId
	}
	if resp.FreeformTags != nil {
		props["FreeformTags"] = resp.FreeformTags
	}
	if resp.DefinedTags != nil {
		props["DefinedTags"] = resp.DefinedTags
	}

	propBytes, err := json.Marshal(props)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal VCN properties: %w", err)
	}

	return &resource.ReadResult{
		ResourceType: "OCI::Core::VCN",
		Properties:   string(propBytes),
	}, nil
}

func (p *VCNProvisioner) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	// CompartmentId is required for listing
	compartmentId, ok := request.AdditionalProperties["CompartmentId"]
	if !ok {
		return nil, fmt.Errorf("CompartmentId is required for listing VCNs")
	}

	listReq := core.ListVcnsRequest{
		CompartmentId: common.String(compartmentId),
	}

	resp, err := client.ListVcns(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list VCNs: %w", err)
	}

	nativeIDs := make([]string, 0, len(resp.Items))
	for _, vcn := range resp.Items {
		nativeIDs = append(nativeIDs, *vcn.Id)
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}
