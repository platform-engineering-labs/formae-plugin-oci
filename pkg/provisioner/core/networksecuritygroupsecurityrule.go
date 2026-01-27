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

	// Validate that the created rule has the expected properties
	if err := validateCreatedRule(rule, securityRule); err != nil {
		return nil, fmt.Errorf("created rule validation failed: %w", err)
	}

	// Build properties from the Create response to avoid eventual consistency issues with List
	properties := map[string]any{
		"Id":                       ruleID,
		"NetworkSecurityGroupId":   nsgId,
		"Direction":                string(rule.Direction),
		"Protocol":                 *rule.Protocol,
	}

	if rule.Description != nil {
		properties["Description"] = *rule.Description
	}
	if rule.Destination != nil {
		properties["Destination"] = *rule.Destination
	}
	if rule.DestinationType != "" {
		properties["DestinationType"] = string(rule.DestinationType)
	}
	if rule.Source != nil {
		properties["Source"] = *rule.Source
	}
	if rule.SourceType != "" {
		properties["SourceType"] = string(rule.SourceType)
	}
	if rule.IsStateless != nil {
		properties["IsStateless"] = *rule.IsStateless
	}
	if rule.TcpOptions != nil {
		tcpOpts := make(map[string]any)
		if rule.TcpOptions.DestinationPortRange != nil {
			tcpOpts["DestinationPortRange"] = map[string]any{
				"Min": *rule.TcpOptions.DestinationPortRange.Min,
				"Max": *rule.TcpOptions.DestinationPortRange.Max,
			}
		}
		if rule.TcpOptions.SourcePortRange != nil {
			tcpOpts["SourcePortRange"] = map[string]any{
				"Min": *rule.TcpOptions.SourcePortRange.Min,
				"Max": *rule.TcpOptions.SourcePortRange.Max,
			}
		}
		if len(tcpOpts) > 0 {
			properties["TcpOptions"] = tcpOpts
		}
	}
	if rule.UdpOptions != nil {
		udpOpts := make(map[string]any)
		if rule.UdpOptions.DestinationPortRange != nil {
			udpOpts["DestinationPortRange"] = map[string]any{
				"Min": *rule.UdpOptions.DestinationPortRange.Min,
				"Max": *rule.UdpOptions.DestinationPortRange.Max,
			}
		}
		if rule.UdpOptions.SourcePortRange != nil {
			udpOpts["SourcePortRange"] = map[string]any{
				"Min": *rule.UdpOptions.SourcePortRange.Min,
				"Max": *rule.UdpOptions.SourcePortRange.Max,
			}
		}
		if len(udpOpts) > 0 {
			properties["UdpOptions"] = udpOpts
		}
	}
	if rule.IcmpOptions != nil {
		icmpOpts := map[string]any{
			"Type": *rule.IcmpOptions.Type,
		}
		if rule.IcmpOptions.Code != nil {
			icmpOpts["Code"] = *rule.IcmpOptions.Code
		}
		properties["IcmpOptions"] = icmpOpts
	}

	propertiesBytes, err := json.Marshal(properties)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal properties: %w", err)
	}

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCreate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           ruleID,
			ResourceProperties: json.RawMessage(propertiesBytes),
		},
	}, nil
}

func (p *NetworkSecurityGroupSecurityRuleProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	// NSG rules don't support update - must delete and recreate
	return nil, fmt.Errorf("update not supported for NetworkSecurityGroupSecurityRule - delete and recreate instead")
}

func (p *NetworkSecurityGroupSecurityRuleProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	// NetworkSecurityGroupSecurityRule delete requires the parent NSG ID, which was previously stored in metadata.
	// Since metadata has been removed from the SDK, this provisioner needs architectural changes to work.
	// The NSG ID would need to be encoded in the NativeID or passed through another mechanism.
	return nil, fmt.Errorf("NetworkSecurityGroupSecurityRule delete is not supported: requires NetworkSecurityGroupId which is no longer available in DeleteRequest")
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
	// NetworkSecurityGroupSecurityRule read requires the parent NSG ID, which was previously stored in metadata.
	// Since metadata has been removed from the SDK, this provisioner needs architectural changes to work.
	// The NSG ID would need to be encoded in the NativeID or passed through another mechanism.
	return nil, fmt.Errorf("NetworkSecurityGroupSecurityRule read is not supported: requires NetworkSecurityGroupId which is no longer available in ReadRequest")
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
