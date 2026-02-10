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

type RouteTableProvisioner struct {
	clients *client.Clients
}

var _ provisioner.Provisioner = &RouteTableProvisioner{}

func init() {
	provisioner.Register("OCI::Core::RouteTable", NewRouteTableProvisioner)
}

func NewRouteTableProvisioner(clients *client.Clients) provisioner.Provisioner {
	return &RouteTableProvisioner{clients: clients}
}

func parseRouteRules(routeRulesData any) ([]core.RouteRule, error) {
	if routeRulesData == nil {
		return nil, nil
	}

	routeRulesList, ok := routeRulesData.([]any)
	if !ok {
		return nil, fmt.Errorf("RouteRules must be an array")
	}

	routeRules := make([]core.RouteRule, 0, len(routeRulesList))
	for i, ruleData := range routeRulesList {
		ruleMap, ok := ruleData.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("RouteRule %d must be an object", i)
		}

		// Try lowercase first (from Pkl JSON), then uppercase (from Go JSON)
		networkEntityId, ok := ruleMap["networkEntityId"].(string)
		if !ok || networkEntityId == "" {
			networkEntityId, ok = ruleMap["NetworkEntityId"].(string)
			if !ok || networkEntityId == "" {
				return nil, fmt.Errorf("RouteRule %d: NetworkEntityId is required", i)
			}
		}

		rule := core.RouteRule{
			NetworkEntityId: common.String(networkEntityId),
		}

		if destination, ok := ruleMap["destination"].(string); ok && destination != "" {
			rule.Destination = common.String(destination)
		} else if destination, ok := ruleMap["Destination"].(string); ok && destination != "" {
			rule.Destination = common.String(destination)
		}

		if destinationType, ok := ruleMap["destinationType"].(string); ok && destinationType != "" {
			rule.DestinationType = core.RouteRuleDestinationTypeEnum(destinationType)
		} else if destinationType, ok := ruleMap["DestinationType"].(string); ok && destinationType != "" {
			rule.DestinationType = core.RouteRuleDestinationTypeEnum(destinationType)
		}

		if description, ok := ruleMap["description"].(string); ok && description != "" {
			rule.Description = common.String(description)
		} else if description, ok := ruleMap["Description"].(string); ok && description != "" {
			rule.Description = common.String(description)
		}

		routeRules = append(routeRules, rule)
	}

	return routeRules, nil
}

func (p *RouteTableProvisioner) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	var props map[string]any
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	createDetails := core.CreateRouteTableDetails{
		CompartmentId: common.String(props["CompartmentId"].(string)),
		VcnId:         common.String(props["VcnId"].(string)),
	}

	if displayName, ok := util.ExtractString(props, "DisplayName"); ok {
		createDetails.DisplayName = common.String(displayName)
	}
	if routeRulesData, ok := props["RouteRules"]; ok {
		routeRules, err := parseRouteRules(routeRulesData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse RouteRules: %w", err)
		}
		createDetails.RouteRules = routeRules
	}
	if freeformTags, ok := util.ExtractFreeformTags(props, "FreeformTags"); ok {
		createDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractDefinedTags(props, "DefinedTags"); ok {
		createDetails.DefinedTags = definedTags
	}

	createReq := core.CreateRouteTableRequest{
		CreateRouteTableDetails: createDetails,
	}

	resp, err := client.CreateRouteTable(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create RouteTable: %w", err)
	}

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCreate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        *resp.Id,
		},
	}, nil
}

func (p *RouteTableProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	props, err := util.ApplyPatchDocument(ctx, request, p.Read)
	if err != nil {
		return nil, err
	}

	updateDetails := core.UpdateRouteTableDetails{}

	if displayName, ok := util.ExtractString(props, "DisplayName"); ok {
		updateDetails.DisplayName = common.String(displayName)
	}

	if routeRulesData, ok := props["RouteRules"]; ok {
		routeRules, err := parseRouteRules(routeRulesData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse RouteRules: %w", err)
		}
		updateDetails.RouteRules = routeRules
	}

	if freeformTags, ok := util.ExtractFreeformTags(props, "FreeformTags"); ok {
		updateDetails.FreeformTags = freeformTags
	}

	if definedTags, ok := util.ExtractDefinedTags(props, "DefinedTags"); ok {
		updateDetails.DefinedTags = definedTags
	}

	updateReq := core.UpdateRouteTableRequest{
		RtId:                    common.String(request.NativeID),
		UpdateRouteTableDetails: updateDetails,
	}

	resp, err := client.UpdateRouteTable(ctx, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update RouteTable: %w", err)
	}

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationUpdate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        *resp.Id,
		},
	}, nil
}

func (p *RouteTableProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
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
		return nil, fmt.Errorf("failed to read RouteTable before delete: %w", err)
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

	deleteReq := core.DeleteRouteTableRequest{
		RtId: common.String(request.NativeID),
	}

	_, err = client.DeleteRouteTable(ctx, deleteReq)
	if err != nil {
		return nil, fmt.Errorf("failed to delete RouteTable: %w", err)
	}

	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationDelete,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (p *RouteTableProvisioner) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			RequestID:       request.RequestID,
		},
	}, nil
}

func (p *RouteTableProvisioner) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	getReq := core.GetRouteTableRequest{
		RtId: common.String(request.NativeID),
	}

	resp, err := client.GetRouteTable(ctx, getReq)
	if err != nil {
		if serviceErr, ok := common.IsServiceError(err); ok && serviceErr.GetHTTPStatusCode() == 404 {
			return &resource.ReadResult{
				ResourceType: "OCI::Core::RouteTable",
				ErrorCode:    resource.OperationErrorCodeNotFound,
			}, nil
		}
		return nil, fmt.Errorf("failed to read RouteTable: %w", err)
	}

	props := map[string]any{
		"CompartmentId": *resp.CompartmentId,
		"VcnId":         *resp.VcnId,
		"Id":            *resp.Id,
	}

	if resp.DisplayName != nil {
		props["DisplayName"] = *resp.DisplayName
	}

	// Always include RouteRules, even if empty
	// Use camelCase to match Pkl schema (nested objects don't get outputKeyTransformation)
	rules := make([]map[string]any, len(resp.RouteRules))
	for i, rule := range resp.RouteRules {
		ruleMap := map[string]any{
			"networkEntityId": *rule.NetworkEntityId,
		}
		if rule.Destination != nil {
			ruleMap["destination"] = *rule.Destination
		}
		if rule.DestinationType != "" {
			ruleMap["destinationType"] = string(rule.DestinationType)
		}
		if rule.Description != nil {
			ruleMap["description"] = *rule.Description
		}
		rules[i] = ruleMap
	}
	props["RouteRules"] = rules

	if resp.FreeformTags != nil {
		props["FreeformTags"] = util.FreeformTagsToList(resp.FreeformTags)
	}
	if resp.DefinedTags != nil {
		props["DefinedTags"] = util.DefinedTagsToList(resp.DefinedTags)
	}

	propBytes, err := json.Marshal(props)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RouteTable properties: %w", err)
	}

	return &resource.ReadResult{
		ResourceType: "OCI::Core::RouteTable",
		Properties:   string(propBytes),
	}, nil
}

func (p *RouteTableProvisioner) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	compartmentId, ok := request.AdditionalProperties["CompartmentId"]
	if !ok {
		return nil, fmt.Errorf("CompartmentId is required for listing RouteTables")
	}

	listReq := core.ListRouteTablesRequest{
		CompartmentId: common.String(compartmentId),
	}

	// Optional: Filter by VcnId
	if vcnId, ok := request.AdditionalProperties["VcnId"]; ok {
		listReq.VcnId = common.String(vcnId)
	}

	resp, err := client.ListRouteTables(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list RouteTables: %w", err)
	}

	nativeIDs := make([]string, 0, len(resp.Items))
	for _, rt := range resp.Items {
		nativeIDs = append(nativeIDs, *rt.Id)
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}
