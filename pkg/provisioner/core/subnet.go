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

type SubnetProvisioner struct {
	clients *client.Clients
}

var _ provisioner.Provisioner = &SubnetProvisioner{}

func init() {
	provisioner.Register("OCI::Core::Subnet", NewSubnetProvisioner)
}

func NewSubnetProvisioner(clients *client.Clients) provisioner.Provisioner {
	return &SubnetProvisioner{clients: clients}
}

func (p *SubnetProvisioner) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	var props map[string]any
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	createDetails := core.CreateSubnetDetails{
		CompartmentId: common.String(props["CompartmentId"].(string)),
		VcnId:         common.String(props["VcnId"].(string)),
		CidrBlock:     common.String(props["CidrBlock"].(string)),
	}

	if ad, ok := util.ExtractString(props, "AvailabilityDomain"); ok {
		createDetails.AvailabilityDomain = common.String(ad)
	}
	if displayName, ok := util.ExtractString(props, "DisplayName"); ok {
		createDetails.DisplayName = common.String(displayName)
	}
	if dnsLabel, ok := util.ExtractString(props, "DnsLabel"); ok {
		createDetails.DnsLabel = common.String(dnsLabel)
	}
	if prohibit, ok := util.ExtractBool(props, "ProhibitPublicIpOnVnic"); ok {
		createDetails.ProhibitPublicIpOnVnic = common.Bool(prohibit)
	}
	if prohibit, ok := util.ExtractBool(props, "ProhibitInternetIngress"); ok {
		createDetails.ProhibitInternetIngress = common.Bool(prohibit)
	}
	if routeTableId, ok := util.ExtractString(props, "RouteTableId"); ok {
		createDetails.RouteTableId = common.String(routeTableId)
	}
	if securityListIds, ok := util.ExtractStringSlice(props, "SecurityListIds"); ok {
		createDetails.SecurityListIds = securityListIds
	}
	if ipv6CidrBlock, ok := util.ExtractString(props, "Ipv6CidrBlock"); ok {
		createDetails.Ipv6CidrBlock = common.String(ipv6CidrBlock)
	}
	if ipv6CidrBlocks, ok := util.ExtractStringSlice(props, "Ipv6CidrBlocks"); ok {
		createDetails.Ipv6CidrBlocks = ipv6CidrBlocks
	}
	if freeformTags, ok := util.ExtractTag(props, "FreeformTags"); ok {
		createDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractNestedTag(props, "DefinedTags"); ok {
		createDetails.DefinedTags = definedTags
	}

	createReq := core.CreateSubnetRequest{
		CreateSubnetDetails: createDetails,
	}

	resp, err := client.CreateSubnet(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create Subnet: %w", err)
	}

	// Build properties from Create response
	properties := map[string]any{
		"CompartmentId": *resp.CompartmentId,
		"VcnId":         *resp.VcnId,
		"Id":            *resp.Id,
		"CidrBlock":     *resp.CidrBlock,
	}

	if resp.AvailabilityDomain != nil {
		properties["AvailabilityDomain"] = *resp.AvailabilityDomain
	}
	if resp.DisplayName != nil {
		properties["DisplayName"] = *resp.DisplayName
	}
	if resp.DnsLabel != nil {
		properties["DnsLabel"] = *resp.DnsLabel
	}
	if resp.ProhibitPublicIpOnVnic != nil {
		properties["ProhibitPublicIpOnVnic"] = *resp.ProhibitPublicIpOnVnic
	}
	if resp.ProhibitInternetIngress != nil {
		properties["ProhibitInternetIngress"] = *resp.ProhibitInternetIngress
	}
	if resp.RouteTableId != nil {
		properties["RouteTableId"] = *resp.RouteTableId
	}
	if resp.SecurityListIds != nil {
		properties["SecurityListIds"] = resp.SecurityListIds
	}
	if resp.VirtualRouterIp != nil {
		properties["VirtualRouterIp"] = *resp.VirtualRouterIp
	}
	if resp.VirtualRouterMac != nil {
		properties["VirtualRouterMac"] = *resp.VirtualRouterMac
	}
	if resp.Ipv6CidrBlock != nil {
		properties["Ipv6CidrBlock"] = *resp.Ipv6CidrBlock
	}
	if resp.Ipv6CidrBlocks != nil {
		properties["Ipv6CidrBlocks"] = resp.Ipv6CidrBlocks
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

func (p *SubnetProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	props, err := util.ApplyPatchDocument(ctx, request, p.Read)
	if err != nil {
		return nil, err
	}

	updateDetails := core.UpdateSubnetDetails{}

	if displayName, ok := util.ExtractString(props, "DisplayName"); ok {
		updateDetails.DisplayName = common.String(displayName)
	}

	if routeTableId, ok := util.ExtractString(props, "RouteTableId"); ok {
		updateDetails.RouteTableId = common.String(routeTableId)
	}

	if securityListIds, ok := util.ExtractStringSlice(props, "SecurityListIds"); ok {
		updateDetails.SecurityListIds = securityListIds
	}

	if freeformTags, ok := util.ExtractTag(props, "FreeformTags"); ok {
		updateDetails.FreeformTags = freeformTags
	}

	if definedTags, ok := util.ExtractNestedTag(props, "DefinedTags"); ok {
		updateDetails.DefinedTags = definedTags
	}

	updateReq := core.UpdateSubnetRequest{
		SubnetId:            common.String(request.NativeID),
		UpdateSubnetDetails: updateDetails,
	}

	resp, err := client.UpdateSubnet(ctx, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update Subnet: %w", err)
	}

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationUpdate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        *resp.Id,
		},
	}, nil
}

func (p *SubnetProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	// Check if subnet exists
	readReq := &resource.ReadRequest{
		NativeID: request.NativeID,
	}
	readRes, err := p.Read(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to read Subnet before delete: %w", err)
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

	deleteReq := core.DeleteSubnetRequest{
		SubnetId: common.String(request.NativeID),
	}

	_, err = client.DeleteSubnet(ctx, deleteReq)
	if err != nil {
		return nil, fmt.Errorf("failed to delete Subnet: %w", err)
	}

	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationDelete,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (p *SubnetProvisioner) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			RequestID:       request.RequestID,
		},
	}, nil
}

func (p *SubnetProvisioner) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	getReq := core.GetSubnetRequest{
		SubnetId: common.String(request.NativeID),
	}

	resp, err := client.GetSubnet(ctx, getReq)
	if err != nil {
		if serviceErr, ok := common.IsServiceError(err); ok && serviceErr.GetHTTPStatusCode() == 404 {
			return &resource.ReadResult{
				ResourceType: "OCI::Core::Subnet",
				ErrorCode:    resource.OperationErrorCodeNotFound,
			}, nil
		}
		return nil, fmt.Errorf("failed to read Subnet: %w", err)
	}

	props := map[string]any{
		"CompartmentId": *resp.CompartmentId,
		"VcnId":         *resp.VcnId,
		"Id":            *resp.Id,
		"CidrBlock":     *resp.CidrBlock,
	}

	if resp.AvailabilityDomain != nil {
		props["AvailabilityDomain"] = *resp.AvailabilityDomain
	}
	if resp.DisplayName != nil {
		props["DisplayName"] = *resp.DisplayName
	}
	if resp.DnsLabel != nil {
		props["DnsLabel"] = *resp.DnsLabel
	}
	if resp.ProhibitPublicIpOnVnic != nil {
		props["ProhibitPublicIpOnVnic"] = *resp.ProhibitPublicIpOnVnic
	}
	if resp.ProhibitInternetIngress != nil {
		props["ProhibitInternetIngress"] = *resp.ProhibitInternetIngress
	}
	if resp.RouteTableId != nil {
		props["RouteTableId"] = *resp.RouteTableId
	}
	if resp.SecurityListIds != nil {
		props["SecurityListIds"] = resp.SecurityListIds
	}
	if resp.VirtualRouterIp != nil {
		props["VirtualRouterIp"] = *resp.VirtualRouterIp
	}
	if resp.VirtualRouterMac != nil {
		props["VirtualRouterMac"] = *resp.VirtualRouterMac
	}
	if resp.Ipv6CidrBlock != nil {
		props["Ipv6CidrBlock"] = *resp.Ipv6CidrBlock
	}
	if resp.Ipv6CidrBlocks != nil {
		props["Ipv6CidrBlocks"] = resp.Ipv6CidrBlocks
	}
	if resp.FreeformTags != nil {
		props["FreeformTags"] = resp.FreeformTags
	}
	if resp.DefinedTags != nil {
		props["DefinedTags"] = resp.DefinedTags
	}

	propBytes, err := json.Marshal(props)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Subnet properties: %w", err)
	}

	return &resource.ReadResult{
		ResourceType: "OCI::Core::Subnet",
		Properties:   string(propBytes),
	}, nil
}

func (p *SubnetProvisioner) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	compartmentId, ok := request.AdditionalProperties["CompartmentId"]
	if !ok {
		return nil, fmt.Errorf("CompartmentId is required for listing Subnets")
	}

	listReq := core.ListSubnetsRequest{
		CompartmentId: common.String(compartmentId),
	}

	// Optional: Filter by VcnId
	if vcnId, ok := request.AdditionalProperties["VcnId"]; ok {
		listReq.VcnId = common.String(vcnId)
	}

	resp, err := client.ListSubnets(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list Subnets: %w", err)
	}

	nativeIDs := make([]string, 0, len(resp.Items))
	for _, subnet := range resp.Items {
		nativeIDs = append(nativeIDs, *subnet.Id)
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}
