// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/client"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/util"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

// parseNativeID extracts the NSG ID and rule ID from the composite NativeID.
// Format: {nsgId}/{ruleId}
func parseNativeID(nativeID string) (nsgId, ruleId string, err error) {
	parts := strings.SplitN(nativeID, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid NativeID format: expected {nsgId}/{ruleId}, got %s", nativeID)
	}
	return parts[0], parts[1], nil
}

type NetworkSecurityGroupSecurityRuleProvisioner struct {
	clients *client.Clients
}

var _ provisioner.Provisioner = &NetworkSecurityGroupSecurityRuleProvisioner{}

func init() {
	provisioner.Register("OCI::Core::NetworkSecurityGroupSecurityRule", NewNetworkSecurityGroupSecurityRuleProvisioner)
}

func NewNetworkSecurityGroupSecurityRuleProvisioner(clients *client.Clients) provisioner.Provisioner {
	return &NetworkSecurityGroupSecurityRuleProvisioner{clients: clients}
}

func (p *NetworkSecurityGroupSecurityRuleProvisioner) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	var props map[string]any
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	securityRule := core.AddSecurityRuleDetails{
		Direction: core.AddSecurityRuleDetailsDirectionEnum(props["Direction"].(string)),
		Protocol:  common.String(props["Protocol"].(string)),
	}

	if description, ok := util.ExtractString(props, "Description"); ok {
		securityRule.Description = common.String(description)
	}
	if destination, ok := util.ExtractString(props, "Destination"); ok {
		securityRule.Destination = common.String(destination)
	}
	if destinationType, ok := util.ExtractString(props, "DestinationType"); ok {
		securityRule.DestinationType = core.AddSecurityRuleDetailsDestinationTypeEnum(destinationType)
	}
	if source, ok := util.ExtractString(props, "Source"); ok {
		securityRule.Source = common.String(source)
	}
	if sourceType, ok := util.ExtractString(props, "SourceType"); ok {
		securityRule.SourceType = core.AddSecurityRuleDetailsSourceTypeEnum(sourceType)
	}
	if isStateless, ok := util.ExtractBool(props, "IsStateless"); ok {
		securityRule.IsStateless = common.Bool(isStateless)
	}

	// TCP Options
	if tcpOptions, ok := props["TcpOptions"].(map[string]any); ok {
		tcpOpts := &core.TcpOptions{}
		if destPortRange, ok := tcpOptions["destinationPortRange"].(map[string]any); ok {
			minPort, minOk := destPortRange["min"]
			maxPort, maxOk := destPortRange["max"]
			if !minOk || !maxOk {
				return nil, fmt.Errorf("TCP destinationPortRange requires both min and max values")
			}
			tcpOpts.DestinationPortRange = &core.PortRange{
				Min: common.Int(int(minPort.(float64))),
				Max: common.Int(int(maxPort.(float64))),
			}
		}
		if srcPortRange, ok := tcpOptions["sourcePortRange"].(map[string]any); ok {
			minPort, minOk := srcPortRange["min"]
			maxPort, maxOk := srcPortRange["max"]
			if !minOk || !maxOk {
				return nil, fmt.Errorf("TCP sourcePortRange requires both min and max values")
			}
			tcpOpts.SourcePortRange = &core.PortRange{
				Min: common.Int(int(minPort.(float64))),
				Max: common.Int(int(maxPort.(float64))),
			}
		}
		securityRule.TcpOptions = tcpOpts
	}

	// UDP Options
	if udpOptions, ok := props["UdpOptions"].(map[string]any); ok {
		udpOpts := &core.UdpOptions{}
		if destPortRange, ok := udpOptions["destinationPortRange"].(map[string]any); ok {
			minPort, minOk := destPortRange["min"]
			maxPort, maxOk := destPortRange["max"]
			if !minOk || !maxOk {
				return nil, fmt.Errorf("UDP destinationPortRange requires both min and max values")
			}
			udpOpts.DestinationPortRange = &core.PortRange{
				Min: common.Int(int(minPort.(float64))),
				Max: common.Int(int(maxPort.(float64))),
			}
		}
		if srcPortRange, ok := udpOptions["sourcePortRange"].(map[string]any); ok {
			minPort, minOk := srcPortRange["min"]
			maxPort, maxOk := srcPortRange["max"]
			if !minOk || !maxOk {
				return nil, fmt.Errorf("UDP sourcePortRange requires both min and max values")
			}
			udpOpts.SourcePortRange = &core.PortRange{
				Min: common.Int(int(minPort.(float64))),
				Max: common.Int(int(maxPort.(float64))),
			}
		}
		securityRule.UdpOptions = udpOpts
	}

	// ICMP Options
	if icmpOptions, ok := props["IcmpOptions"].(map[string]any); ok {
		icmpOpts := &core.IcmpOptions{
			Type: common.Int(int(icmpOptions["Type"].(float64))),
		}
		if code, ok := icmpOptions["Code"].(float64); ok {
			icmpOpts.Code = common.Int(int(code))
		}
		securityRule.IcmpOptions = icmpOpts
	}

	nsgId, ok := util.ExtractResolvedReference(props, "NetworkSecurityGroupId")
	if !ok {
		return nil, fmt.Errorf("NetworkSecurityGroupId is required")
	}

	addReq := core.AddNetworkSecurityGroupSecurityRulesRequest{
		NetworkSecurityGroupId: common.String(nsgId),
		AddNetworkSecurityGroupSecurityRulesDetails: core.AddNetworkSecurityGroupSecurityRulesDetails{
			SecurityRules: []core.AddSecurityRuleDetails{securityRule},
		},
	}

	resp, err := client.AddNetworkSecurityGroupSecurityRules(ctx, addReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create NetworkSecurityGroupSecurityRule: %w", err)
	}

	// Use first rule ID as the NativeID
	if len(resp.SecurityRules) == 0 {
		return nil, fmt.Errorf("no security rules returned from OCI")
	}

	rule := resp.SecurityRules[0]
	ruleID := *rule.Id

	// Encode both NSG ID and rule ID in NativeID so Read/Delete can access the NSG ID
	// Format: {nsgId}/{ruleId}
	nativeID := fmt.Sprintf("%s/%s", nsgId, ruleID)

	// Validate that the created rule has the expected properties
	if err := validateCreatedRule(rule, securityRule); err != nil {
		return nil, fmt.Errorf("created rule validation failed: %w", err)
	}

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCreate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        nativeID,
		},
	}, nil
}

func (p *NetworkSecurityGroupSecurityRuleProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	// NSG rules don't support update - must delete and recreate
	return nil, fmt.Errorf("update not supported for NetworkSecurityGroupSecurityRule - delete and recreate instead")
}

func (p *NetworkSecurityGroupSecurityRuleProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	nsgId, ruleId, err := parseNativeID(request.NativeID)
	if err != nil {
		return nil, err
	}

	removeReq := core.RemoveNetworkSecurityGroupSecurityRulesRequest{
		NetworkSecurityGroupId: common.String(nsgId),
		RemoveNetworkSecurityGroupSecurityRulesDetails: core.RemoveNetworkSecurityGroupSecurityRulesDetails{
			SecurityRuleIds: []string{ruleId},
		},
	}

	_, err = client.RemoveNetworkSecurityGroupSecurityRules(ctx, removeReq)
	if err != nil {
		if serviceErr, ok := common.IsServiceError(err); ok && serviceErr.GetHTTPStatusCode() == 404 {
			// Already deleted
			return &resource.DeleteResult{
				ProgressResult: &resource.ProgressResult{
					Operation:       resource.OperationDelete,
					OperationStatus: resource.OperationStatusSuccess,
					NativeID:        request.NativeID,
				},
			}, nil
		}
		return nil, fmt.Errorf("failed to delete NetworkSecurityGroupSecurityRule: %w", err)
	}

	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationDelete,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (p *NetworkSecurityGroupSecurityRuleProvisioner) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			RequestID:       request.RequestID,
		},
	}, nil
}

func (p *NetworkSecurityGroupSecurityRuleProvisioner) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	nsgId, ruleId, err := parseNativeID(request.NativeID)
	if err != nil {
		return nil, err
	}

	rule, err := p.getSecurityRuleById(ctx, nsgId, ruleId)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return &resource.ReadResult{
			ResourceType: "OCI::Core::NetworkSecurityGroupSecurityRule",
			ErrorCode:    resource.OperationErrorCodeNotFound,
		}, nil
	}

	props := buildSecurityRuleProperties(nsgId, ruleId, rule)
	propBytes, err := json.Marshal(props)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal properties: %w", err)
	}

	return &resource.ReadResult{
		ResourceType: "OCI::Core::NetworkSecurityGroupSecurityRule",
		Properties:   string(propBytes),
	}, nil
}

// getSecurityRuleById fetches a security rule by listing rules in the NSG and finding the matching one.
// Returns nil if the rule is not found.
func (p *NetworkSecurityGroupSecurityRuleProvisioner) getSecurityRuleById(ctx context.Context, nsgId, ruleId string) (*core.SecurityRule, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	listReq := core.ListNetworkSecurityGroupSecurityRulesRequest{
		NetworkSecurityGroupId: common.String(nsgId),
	}

	resp, err := client.ListNetworkSecurityGroupSecurityRules(ctx, listReq)
	if err != nil {
		if serviceErr, ok := common.IsServiceError(err); ok && serviceErr.GetHTTPStatusCode() == 404 {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list security rules: %w", err)
	}

	for i := range resp.Items {
		if *resp.Items[i].Id == ruleId {
			return &resp.Items[i], nil
		}
	}

	return nil, nil
}

// buildSecurityRuleProperties builds the properties map from a security rule.
func buildSecurityRuleProperties(nsgId, ruleId string, rule *core.SecurityRule) map[string]any {
	props := map[string]any{
		"Id":                     ruleId,
		"NetworkSecurityGroupId": nsgId,
		"Direction":              string(rule.Direction),
		"Protocol":               *rule.Protocol,
	}

	if rule.Description != nil {
		props["Description"] = *rule.Description
	}
	if rule.Destination != nil {
		props["Destination"] = *rule.Destination
	}
	if rule.DestinationType != "" {
		props["DestinationType"] = string(rule.DestinationType)
	}
	if rule.Source != nil {
		props["Source"] = *rule.Source
	}
	if rule.SourceType != "" {
		props["SourceType"] = string(rule.SourceType)
	}
	if rule.IsStateless != nil {
		props["IsStateless"] = *rule.IsStateless
	}
	// Use camelCase for nested objects to match Pkl schema (outputKeyTransformation doesn't apply to nested objects)
	if rule.TcpOptions != nil {
		tcpOpts := make(map[string]any)
		if rule.TcpOptions.DestinationPortRange != nil {
			tcpOpts["destinationPortRange"] = map[string]any{
				"min": *rule.TcpOptions.DestinationPortRange.Min,
				"max": *rule.TcpOptions.DestinationPortRange.Max,
			}
		}
		if rule.TcpOptions.SourcePortRange != nil {
			tcpOpts["sourcePortRange"] = map[string]any{
				"min": *rule.TcpOptions.SourcePortRange.Min,
				"max": *rule.TcpOptions.SourcePortRange.Max,
			}
		}
		if len(tcpOpts) > 0 {
			props["TcpOptions"] = tcpOpts
		}
	}
	if rule.UdpOptions != nil {
		udpOpts := make(map[string]any)
		if rule.UdpOptions.DestinationPortRange != nil {
			udpOpts["destinationPortRange"] = map[string]any{
				"min": *rule.UdpOptions.DestinationPortRange.Min,
				"max": *rule.UdpOptions.DestinationPortRange.Max,
			}
		}
		if rule.UdpOptions.SourcePortRange != nil {
			udpOpts["sourcePortRange"] = map[string]any{
				"min": *rule.UdpOptions.SourcePortRange.Min,
				"max": *rule.UdpOptions.SourcePortRange.Max,
			}
		}
		if len(udpOpts) > 0 {
			props["UdpOptions"] = udpOpts
		}
	}
	if rule.IcmpOptions != nil {
		icmpOpts := map[string]any{
			"type": *rule.IcmpOptions.Type,
		}
		if rule.IcmpOptions.Code != nil {
			icmpOpts["code"] = *rule.IcmpOptions.Code
		}
		props["IcmpOptions"] = icmpOpts
	}

	return props
}

func (p *NetworkSecurityGroupSecurityRuleProvisioner) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	nsgId, ok := request.AdditionalProperties["NetworkSecurityGroupId"]
	if !ok {
		return nil, fmt.Errorf("NetworkSecurityGroupId is required for listing security rules")
	}

	listReq := core.ListNetworkSecurityGroupSecurityRulesRequest{
		NetworkSecurityGroupId: common.String(nsgId),
	}

	resp, err := client.ListNetworkSecurityGroupSecurityRules(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list security rules: %w", err)
	}

	nativeIDs := make([]string, 0, len(resp.Items))
	for _, rule := range resp.Items {
		nativeIDs = append(nativeIDs, *rule.Id)
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

// validateCreatedRule ensures the created rule matches the requested configuration
func validateCreatedRule(createdRule core.SecurityRule, requestedRule core.AddSecurityRuleDetails) error {
	// Validate basic properties
	if createdRule.Direction != core.SecurityRuleDirectionEnum(requestedRule.Direction) {
		return fmt.Errorf("direction mismatch: expected %s, got %s", requestedRule.Direction, createdRule.Direction)
	}

	if createdRule.Protocol != nil && requestedRule.Protocol != nil && *createdRule.Protocol != *requestedRule.Protocol {
		return fmt.Errorf("protocol mismatch: expected %s, got %s", *requestedRule.Protocol, *createdRule.Protocol)
	}

	// Validate TCP options if they were specified
	if requestedRule.TcpOptions != nil {
		if createdRule.TcpOptions == nil {
			return fmt.Errorf("TCP options were requested but not set in created rule")
		}

		// Validate destination port range
		if requestedRule.TcpOptions.DestinationPortRange != nil {
			if createdRule.TcpOptions.DestinationPortRange == nil {
				return fmt.Errorf("TCP destination port range was requested but not set in created rule")
			}
			reqMin, reqMax := *requestedRule.TcpOptions.DestinationPortRange.Min, *requestedRule.TcpOptions.DestinationPortRange.Max
			createdMin, createdMax := *createdRule.TcpOptions.DestinationPortRange.Min, *createdRule.TcpOptions.DestinationPortRange.Max
			if reqMin != createdMin || reqMax != createdMax {
				return fmt.Errorf("TCP destination port range mismatch: expected %d-%d, got %d-%d", reqMin, reqMax, createdMin, createdMax)
			}
		}

		// Validate source port range
		if requestedRule.TcpOptions.SourcePortRange != nil {
			if createdRule.TcpOptions.SourcePortRange == nil {
				return fmt.Errorf("TCP source port range was requested but not set in created rule")
			}
			reqMin, reqMax := *requestedRule.TcpOptions.SourcePortRange.Min, *requestedRule.TcpOptions.SourcePortRange.Max
			createdMin, createdMax := *createdRule.TcpOptions.SourcePortRange.Min, *createdRule.TcpOptions.SourcePortRange.Max
			if reqMin != createdMin || reqMax != createdMax {
				return fmt.Errorf("TCP source port range mismatch: expected %d-%d, got %d-%d", reqMin, reqMax, createdMin, createdMax)
			}
		}
	}

	// Validate UDP options if they were specified
	if requestedRule.UdpOptions != nil {
		if createdRule.UdpOptions == nil {
			return fmt.Errorf("UDP options were requested but not set in created rule")
		}

		// Similar validation logic for UDP ports...
		if requestedRule.UdpOptions.DestinationPortRange != nil {
			if createdRule.UdpOptions.DestinationPortRange == nil {
				return fmt.Errorf("UDP destination port range was requested but not set in created rule")
			}
			reqMin, reqMax := *requestedRule.UdpOptions.DestinationPortRange.Min, *requestedRule.UdpOptions.DestinationPortRange.Max
			createdMin, createdMax := *createdRule.UdpOptions.DestinationPortRange.Min, *createdRule.UdpOptions.DestinationPortRange.Max
			if reqMin != createdMin || reqMax != createdMax {
				return fmt.Errorf("UDP destination port range mismatch: expected %d-%d, got %d-%d", reqMin, reqMax, createdMin, createdMax)
			}
		}
	}

	return nil
}
