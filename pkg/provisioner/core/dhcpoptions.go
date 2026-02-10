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

type DhcpOptionsProvisioner struct {
	clients *client.Clients
}

var _ provisioner.Provisioner = &DhcpOptionsProvisioner{}

func init() {
	provisioner.Register("OCI::Core::DhcpOptions", NewDhcpOptionsProvisioner)
}

func NewDhcpOptionsProvisioner(clients *client.Clients) provisioner.Provisioner {
	return &DhcpOptionsProvisioner{clients: clients}
}

func parseDhcpOptions(optionsData any) ([]core.DhcpOption, error) {
	if optionsData == nil {
		return []core.DhcpOption{}, nil
	}

	optionsList, ok := optionsData.([]any)
	if !ok {
		return nil, fmt.Errorf("options must be an array")
	}

	options := make([]core.DhcpOption, 0, len(optionsList))
	for i, optData := range optionsList {
		optMap, ok := optData.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("option %d must be an object", i)
		}

		optType, _ := extractStringField(optMap, "type", "Type")

		switch optType {
		case "DomainNameServer":
			serverType, _ := extractStringField(optMap, "serverType", "ServerType")
			opt := core.DhcpDnsOption{
				ServerType: core.DhcpDnsOptionServerTypeEnum(serverType),
			}
			if customDns := extractStringSliceField(optMap, "customDnsServers", "CustomDnsServers"); len(customDns) > 0 {
				opt.CustomDnsServers = customDns
			}
			options = append(options, opt)

		case "SearchDomain":
			opt := core.DhcpSearchDomainOption{}
			if names := extractStringSliceField(optMap, "searchDomainNames", "SearchDomainNames"); len(names) > 0 {
				opt.SearchDomainNames = names
			}
			options = append(options, opt)

		default:
			return nil, fmt.Errorf("option %d: unknown type %q", i, optType)
		}
	}

	return options, nil
}

func extractStringSliceField(m map[string]any, lowerKey, upperKey string) []string {
	var slice []any
	if s, ok := m[lowerKey].([]any); ok {
		slice = s
	} else if s, ok := m[upperKey].([]any); ok {
		slice = s
	}
	if len(slice) == 0 {
		return nil
	}
	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func serializeDhcpOptions(options []core.DhcpOption) []map[string]any {
	result := make([]map[string]any, 0, len(options))
	for _, opt := range options {
		switch v := opt.(type) {
		case core.DhcpDnsOption:
			m := map[string]any{
				"type":       "DomainNameServer",
				"serverType": string(v.ServerType),
			}
			if len(v.CustomDnsServers) > 0 {
				m["customDnsServers"] = v.CustomDnsServers
			}
			result = append(result, m)
		case core.DhcpSearchDomainOption:
			m := map[string]any{
				"type": "SearchDomain",
			}
			if len(v.SearchDomainNames) > 0 {
				m["searchDomainNames"] = v.SearchDomainNames
			}
			result = append(result, m)
		}
	}
	return result
}

func (p *DhcpOptionsProvisioner) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	svc, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	var props map[string]any
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	options, err := parseDhcpOptions(props["Options"])
	if err != nil {
		return nil, fmt.Errorf("failed to parse Options: %w", err)
	}

	createDetails := core.CreateDhcpDetails{
		CompartmentId: common.String(props["CompartmentId"].(string)),
		VcnId:         common.String(props["VcnId"].(string)),
		Options:       options,
	}

	if displayName, ok := util.ExtractString(props, "DisplayName"); ok {
		createDetails.DisplayName = common.String(displayName)
	}
	if domainNameType, ok := util.ExtractString(props, "DomainNameType"); ok {
		createDetails.DomainNameType = core.CreateDhcpDetailsDomainNameTypeEnum(domainNameType)
	}
	if freeformTags, ok := util.ExtractFreeformTags(props, "FreeformTags"); ok {
		createDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractDefinedTags(props, "DefinedTags"); ok {
		createDetails.DefinedTags = definedTags
	}

	createReq := core.CreateDhcpOptionsRequest{
		CreateDhcpDetails: createDetails,
	}

	resp, err := svc.CreateDhcpOptions(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create DhcpOptions: %w", err)
	}

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCreate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        *resp.Id,
		},
	}, nil
}

func (p *DhcpOptionsProvisioner) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	svc, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	getReq := core.GetDhcpOptionsRequest{
		DhcpId: common.String(request.NativeID),
	}

	resp, err := svc.GetDhcpOptions(ctx, getReq)
	if err != nil {
		if serviceErr, ok := common.IsServiceError(err); ok && serviceErr.GetHTTPStatusCode() == 404 {
			return &resource.ReadResult{
				ResourceType: "OCI::Core::DhcpOptions",
				ErrorCode:    resource.OperationErrorCodeNotFound,
			}, nil
		}
		return nil, fmt.Errorf("failed to read DhcpOptions: %w", err)
	}

	properties := buildDhcpOptionsProperties(resp.DhcpOptions)

	propBytes, err := json.Marshal(properties)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DhcpOptions properties: %w", err)
	}

	return &resource.ReadResult{
		ResourceType: "OCI::Core::DhcpOptions",
		Properties:   string(propBytes),
	}, nil
}

func (p *DhcpOptionsProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	svc, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	props, err := util.ApplyPatchDocument(ctx, request, p.Read)
	if err != nil {
		return nil, err
	}

	updateDetails := core.UpdateDhcpDetails{}

	if displayName, ok := util.ExtractString(props, "DisplayName"); ok {
		updateDetails.DisplayName = common.String(displayName)
	}
	if optionsData, ok := props["Options"]; ok {
		options, err := parseDhcpOptions(optionsData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Options: %w", err)
		}
		updateDetails.Options = options
	}
	if domainNameType, ok := util.ExtractString(props, "DomainNameType"); ok {
		updateDetails.DomainNameType = core.UpdateDhcpDetailsDomainNameTypeEnum(domainNameType)
	}
	if freeformTags, ok := util.ExtractFreeformTags(props, "FreeformTags"); ok {
		updateDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractDefinedTags(props, "DefinedTags"); ok {
		updateDetails.DefinedTags = definedTags
	}

	updateReq := core.UpdateDhcpOptionsRequest{
		DhcpId:            common.String(request.NativeID),
		UpdateDhcpDetails: updateDetails,
	}

	resp, err := svc.UpdateDhcpOptions(ctx, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update DhcpOptions: %w", err)
	}

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationUpdate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        *resp.Id,
		},
	}, nil
}

func (p *DhcpOptionsProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	svc, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	readReq := &resource.ReadRequest{
		NativeID: request.NativeID,
	}
	readRes, err := p.Read(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to read DhcpOptions before delete: %w", err)
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

	deleteReq := core.DeleteDhcpOptionsRequest{
		DhcpId: common.String(request.NativeID),
	}

	_, err = svc.DeleteDhcpOptions(ctx, deleteReq)
	if err != nil {
		return nil, fmt.Errorf("failed to delete DhcpOptions: %w", err)
	}

	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationDelete,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (p *DhcpOptionsProvisioner) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			RequestID:       request.RequestID,
		},
	}, nil
}

func (p *DhcpOptionsProvisioner) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	svc, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	compartmentId, ok := request.AdditionalProperties["CompartmentId"]
	if !ok {
		return nil, fmt.Errorf("CompartmentId is required for listing DhcpOptions")
	}

	listReq := core.ListDhcpOptionsRequest{
		CompartmentId: common.String(compartmentId),
	}

	if vcnId, ok := request.AdditionalProperties["VcnId"]; ok {
		listReq.VcnId = common.String(vcnId)
	}

	resp, err := svc.ListDhcpOptions(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list DhcpOptions: %w", err)
	}

	nativeIDs := make([]string, 0, len(resp.Items))
	for _, item := range resp.Items {
		nativeIDs = append(nativeIDs, *item.Id)
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

func buildDhcpOptionsProperties(dhcp core.DhcpOptions) map[string]any {
	properties := map[string]any{
		"CompartmentId": *dhcp.CompartmentId,
		"VcnId":         *dhcp.VcnId,
		"Id":            *dhcp.Id,
		"Options":       serializeDhcpOptions(dhcp.Options),
	}

	if dhcp.DisplayName != nil {
		properties["DisplayName"] = *dhcp.DisplayName
	}
	if dhcp.DomainNameType != "" {
		properties["DomainNameType"] = string(dhcp.DomainNameType)
	}
	if dhcp.FreeformTags != nil {
		properties["FreeformTags"] = util.FreeformTagsToList(dhcp.FreeformTags)
	}
	if dhcp.DefinedTags != nil {
		properties["DefinedTags"] = util.DefinedTagsToList(dhcp.DefinedTags)
	}

	return properties
}
