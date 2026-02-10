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

type SecurityListProvisioner struct {
	clients *client.Clients
}

var _ provisioner.Provisioner = &SecurityListProvisioner{}

func init() {
	provisioner.Register("OCI::Core::SecurityList", NewSecurityListProvisioner)
}

func NewSecurityListProvisioner(clients *client.Clients) provisioner.Provisioner {
	return &SecurityListProvisioner{clients: clients}
}

// Helper to extract string with lowercase or uppercase key
func extractStringField(m map[string]any, lowerKey, upperKey string) (string, bool) {
	if v, ok := m[lowerKey].(string); ok && v != "" {
		return v, true
	}
	if v, ok := m[upperKey].(string); ok && v != "" {
		return v, true
	}
	return "", false
}

// Helper to extract bool with lowercase or uppercase key
func extractBoolField(m map[string]any, lowerKey, upperKey string) (bool, bool) {
	if v, ok := m[lowerKey].(bool); ok {
		return v, true
	}
	if v, ok := m[upperKey].(bool); ok {
		return v, true
	}
	return false, false
}

// Helper to extract int with lowercase or uppercase key (JSON numbers come as float64)
func extractIntField(m map[string]any, lowerKey, upperKey string) (int, bool) {
	if v, ok := m[lowerKey].(float64); ok {
		return int(v), true
	}
	if v, ok := m[upperKey].(float64); ok {
		return int(v), true
	}
	return 0, false
}

// Helper to extract nested map with lowercase or uppercase key
func extractMapField(m map[string]any, lowerKey, upperKey string) (map[string]any, bool) {
	if v, ok := m[lowerKey].(map[string]any); ok {
		return v, true
	}
	if v, ok := m[upperKey].(map[string]any); ok {
		return v, true
	}
	return nil, false
}

func parsePortRange(data map[string]any) *core.PortRange {
	if data == nil {
		return nil
	}
	minVal, hasMin := extractIntField(data, "min", "Min")
	maxVal, hasMax := extractIntField(data, "max", "Max")
	if !hasMin || !hasMax {
		return nil
	}
	return &core.PortRange{
		Min: common.Int(minVal),
		Max: common.Int(maxVal),
	}
}

func parseTcpOptions(data map[string]any) *core.TcpOptions {
	if data == nil {
		return nil
	}
	opts := &core.TcpOptions{}
	if destRange, ok := extractMapField(data, "destinationPortRange", "DestinationPortRange"); ok {
		opts.DestinationPortRange = parsePortRange(destRange)
	}
	if srcRange, ok := extractMapField(data, "sourcePortRange", "SourcePortRange"); ok {
		opts.SourcePortRange = parsePortRange(srcRange)
	}
	if opts.DestinationPortRange == nil && opts.SourcePortRange == nil {
		return nil
	}
	return opts
}

func parseUdpOptions(data map[string]any) *core.UdpOptions {
	if data == nil {
		return nil
	}
	opts := &core.UdpOptions{}
	if destRange, ok := extractMapField(data, "destinationPortRange", "DestinationPortRange"); ok {
		opts.DestinationPortRange = parsePortRange(destRange)
	}
	if srcRange, ok := extractMapField(data, "sourcePortRange", "SourcePortRange"); ok {
		opts.SourcePortRange = parsePortRange(srcRange)
	}
	if opts.DestinationPortRange == nil && opts.SourcePortRange == nil {
		return nil
	}
	return opts
}

func parseIcmpOptions(data map[string]any) *core.IcmpOptions {
	if data == nil {
		return nil
	}
	icmpType, hasType := extractIntField(data, "type", "Type")
	if !hasType {
		return nil
	}
	opts := &core.IcmpOptions{
		Type: common.Int(icmpType),
	}
	if code, ok := extractIntField(data, "code", "Code"); ok {
		opts.Code = common.Int(code)
	}
	return opts
}

func parseIngressSecurityRules(rulesData any) ([]core.IngressSecurityRule, error) {
	if rulesData == nil {
		return []core.IngressSecurityRule{}, nil
	}

	rulesList, ok := rulesData.([]any)
	if !ok {
		return nil, fmt.Errorf("IngressSecurityRules must be an array")
	}

	rules := make([]core.IngressSecurityRule, 0, len(rulesList))
	for i, ruleData := range rulesList {
		ruleMap, ok := ruleData.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("IngressSecurityRule %d must be an object", i)
		}

		protocol, ok := extractStringField(ruleMap, "protocol", "Protocol")
		if !ok {
			return nil, fmt.Errorf("IngressSecurityRule %d: protocol is required", i)
		}

		source, ok := extractStringField(ruleMap, "source", "Source")
		if !ok {
			return nil, fmt.Errorf("IngressSecurityRule %d: source is required", i)
		}

		rule := core.IngressSecurityRule{
			Protocol: common.String(protocol),
			Source:   common.String(source),
		}

		if sourceType, ok := extractStringField(ruleMap, "sourceType", "SourceType"); ok {
			rule.SourceType = core.IngressSecurityRuleSourceTypeEnum(sourceType)
		}

		if isStateless, ok := extractBoolField(ruleMap, "isStateless", "IsStateless"); ok {
			rule.IsStateless = common.Bool(isStateless)
		}

		if tcpOpts, ok := extractMapField(ruleMap, "tcpOptions", "TcpOptions"); ok {
			rule.TcpOptions = parseTcpOptions(tcpOpts)
		}

		if udpOpts, ok := extractMapField(ruleMap, "udpOptions", "UdpOptions"); ok {
			rule.UdpOptions = parseUdpOptions(udpOpts)
		}

		if icmpOpts, ok := extractMapField(ruleMap, "icmpOptions", "IcmpOptions"); ok {
			rule.IcmpOptions = parseIcmpOptions(icmpOpts)
		}

		if description, ok := extractStringField(ruleMap, "description", "Description"); ok {
			rule.Description = common.String(description)
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

func parseEgressSecurityRules(rulesData any) ([]core.EgressSecurityRule, error) {
	if rulesData == nil {
		return []core.EgressSecurityRule{}, nil
	}

	rulesList, ok := rulesData.([]any)
	if !ok {
		return nil, fmt.Errorf("EgressSecurityRules must be an array")
	}

	rules := make([]core.EgressSecurityRule, 0, len(rulesList))
	for i, ruleData := range rulesList {
		ruleMap, ok := ruleData.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("EgressSecurityRule %d must be an object", i)
		}

		protocol, ok := extractStringField(ruleMap, "protocol", "Protocol")
		if !ok {
			return nil, fmt.Errorf("EgressSecurityRule %d: protocol is required", i)
		}

		destination, ok := extractStringField(ruleMap, "destination", "Destination")
		if !ok {
			return nil, fmt.Errorf("EgressSecurityRule %d: destination is required", i)
		}

		rule := core.EgressSecurityRule{
			Protocol:    common.String(protocol),
			Destination: common.String(destination),
		}

		if destType, ok := extractStringField(ruleMap, "destinationType", "DestinationType"); ok {
			rule.DestinationType = core.EgressSecurityRuleDestinationTypeEnum(destType)
		}

		if isStateless, ok := extractBoolField(ruleMap, "isStateless", "IsStateless"); ok {
			rule.IsStateless = common.Bool(isStateless)
		}

		if tcpOpts, ok := extractMapField(ruleMap, "tcpOptions", "TcpOptions"); ok {
			rule.TcpOptions = parseTcpOptions(tcpOpts)
		}

		if udpOpts, ok := extractMapField(ruleMap, "udpOptions", "UdpOptions"); ok {
			rule.UdpOptions = parseUdpOptions(udpOpts)
		}

		if icmpOpts, ok := extractMapField(ruleMap, "icmpOptions", "IcmpOptions"); ok {
			rule.IcmpOptions = parseIcmpOptions(icmpOpts)
		}

		if description, ok := extractStringField(ruleMap, "description", "Description"); ok {
			rule.Description = common.String(description)
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

// serializeIngressRules converts ingress rules to maps with camelCase keys to match Pkl schema.
// Note: Nested objects don't get outputKeyTransformation, so must match schema case exactly.
func serializeIngressRules(rules []core.IngressSecurityRule) []map[string]any {
	result := make([]map[string]any, len(rules))
	for i, rule := range rules {
		ruleMap := map[string]any{
			"protocol": *rule.Protocol,
			"source":   *rule.Source,
		}
		if rule.SourceType != "" {
			ruleMap["sourceType"] = string(rule.SourceType)
		}
		if rule.IsStateless != nil {
			ruleMap["isStateless"] = *rule.IsStateless
		}
		if rule.Description != nil {
			ruleMap["description"] = *rule.Description
		}
		if rule.TcpOptions != nil {
			tcpOpts := map[string]any{}
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
				ruleMap["tcpOptions"] = tcpOpts
			}
		}
		if rule.UdpOptions != nil {
			udpOpts := map[string]any{}
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
				ruleMap["udpOptions"] = udpOpts
			}
		}
		if rule.IcmpOptions != nil {
			icmpOpts := map[string]any{
				"type": *rule.IcmpOptions.Type,
			}
			if rule.IcmpOptions.Code != nil {
				icmpOpts["code"] = *rule.IcmpOptions.Code
			}
			ruleMap["icmpOptions"] = icmpOpts
		}
		result[i] = ruleMap
	}
	return result
}

// serializeEgressRules converts egress rules to maps with camelCase keys to match Pkl schema.
// Note: Nested objects don't get outputKeyTransformation, so must match schema case exactly.
func serializeEgressRules(rules []core.EgressSecurityRule) []map[string]any {
	result := make([]map[string]any, len(rules))
	for i, rule := range rules {
		ruleMap := map[string]any{
			"protocol":    *rule.Protocol,
			"destination": *rule.Destination,
		}
		if rule.DestinationType != "" {
			ruleMap["destinationType"] = string(rule.DestinationType)
		}
		if rule.IsStateless != nil {
			ruleMap["isStateless"] = *rule.IsStateless
		}
		if rule.Description != nil {
			ruleMap["description"] = *rule.Description
		}
		if rule.TcpOptions != nil {
			tcpOpts := map[string]any{}
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
				ruleMap["tcpOptions"] = tcpOpts
			}
		}
		if rule.UdpOptions != nil {
			udpOpts := map[string]any{}
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
				ruleMap["udpOptions"] = udpOpts
			}
		}
		if rule.IcmpOptions != nil {
			icmpOpts := map[string]any{
				"type": *rule.IcmpOptions.Type,
			}
			if rule.IcmpOptions.Code != nil {
				icmpOpts["code"] = *rule.IcmpOptions.Code
			}
			ruleMap["icmpOptions"] = icmpOpts
		}
		result[i] = ruleMap
	}
	return result
}

func (p *SecurityListProvisioner) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	var props map[string]any
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	ingressRules, err := parseIngressSecurityRules(props["IngressSecurityRules"])
	if err != nil {
		return nil, fmt.Errorf("failed to parse IngressSecurityRules: %w", err)
	}

	egressRules, err := parseEgressSecurityRules(props["EgressSecurityRules"])
	if err != nil {
		return nil, fmt.Errorf("failed to parse EgressSecurityRules: %w", err)
	}

	createDetails := core.CreateSecurityListDetails{
		CompartmentId:        common.String(props["CompartmentId"].(string)),
		VcnId:                common.String(props["VcnId"].(string)),
		IngressSecurityRules: ingressRules,
		EgressSecurityRules:  egressRules,
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

	createReq := core.CreateSecurityListRequest{
		CreateSecurityListDetails: createDetails,
	}

	resp, err := client.CreateSecurityList(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create SecurityList: %w", err)
	}

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCreate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        *resp.Id,
		},
	}, nil
}

func (p *SecurityListProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	props, err := util.ApplyPatchDocument(ctx, request, p.Read)
	if err != nil {
		return nil, err
	}

	updateDetails := core.UpdateSecurityListDetails{}

	if displayName, ok := util.ExtractString(props, "DisplayName"); ok {
		updateDetails.DisplayName = common.String(displayName)
	}

	if ingressRulesData, ok := props["IngressSecurityRules"]; ok {
		ingressRules, err := parseIngressSecurityRules(ingressRulesData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse IngressSecurityRules: %w", err)
		}
		updateDetails.IngressSecurityRules = ingressRules
	}

	if egressRulesData, ok := props["EgressSecurityRules"]; ok {
		egressRules, err := parseEgressSecurityRules(egressRulesData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse EgressSecurityRules: %w", err)
		}
		updateDetails.EgressSecurityRules = egressRules
	}

	if freeformTags, ok := util.ExtractFreeformTags(props, "FreeformTags"); ok {
		updateDetails.FreeformTags = freeformTags
	}

	if definedTags, ok := util.ExtractDefinedTags(props, "DefinedTags"); ok {
		updateDetails.DefinedTags = definedTags
	}

	updateReq := core.UpdateSecurityListRequest{
		SecurityListId:            common.String(request.NativeID),
		UpdateSecurityListDetails: updateDetails,
	}

	resp, err := client.UpdateSecurityList(ctx, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update SecurityList: %w", err)
	}

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationUpdate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        *resp.Id,
		},
	}, nil
}

func (p *SecurityListProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	readReq := &resource.ReadRequest{
		NativeID: request.NativeID,
	}
	readRes, err := p.Read(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to read SecurityList before delete: %w", err)
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

	deleteReq := core.DeleteSecurityListRequest{
		SecurityListId: common.String(request.NativeID),
	}

	_, err = client.DeleteSecurityList(ctx, deleteReq)
	if err != nil {
		return nil, fmt.Errorf("failed to delete SecurityList: %w", err)
	}

	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationDelete,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (p *SecurityListProvisioner) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			RequestID:       request.RequestID,
		},
	}, nil
}

func (p *SecurityListProvisioner) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	getReq := core.GetSecurityListRequest{
		SecurityListId: common.String(request.NativeID),
	}

	resp, err := client.GetSecurityList(ctx, getReq)
	if err != nil {
		if serviceErr, ok := common.IsServiceError(err); ok && serviceErr.GetHTTPStatusCode() == 404 {
			return &resource.ReadResult{
				ResourceType: "OCI::Core::SecurityList",
				ErrorCode:    resource.OperationErrorCodeNotFound,
			}, nil
		}
		return nil, fmt.Errorf("failed to read SecurityList: %w", err)
	}

	props := map[string]any{
		"CompartmentId":        *resp.CompartmentId,
		"VcnId":                *resp.VcnId,
		"Id":                   *resp.Id,
		"IngressSecurityRules": serializeIngressRules(resp.IngressSecurityRules),
		"EgressSecurityRules":  serializeEgressRules(resp.EgressSecurityRules),
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
		return nil, fmt.Errorf("failed to marshal SecurityList properties: %w", err)
	}

	return &resource.ReadResult{
		ResourceType: "OCI::Core::SecurityList",
		Properties:   string(propBytes),
	}, nil
}

func (p *SecurityListProvisioner) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	client, err := p.clients.GetVirtualNetworkClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get VirtualNetwork client: %w", err)
	}

	compartmentId, ok := request.AdditionalProperties["CompartmentId"]
	if !ok {
		return nil, fmt.Errorf("CompartmentId is required for listing SecurityLists")
	}

	listReq := core.ListSecurityListsRequest{
		CompartmentId: common.String(compartmentId),
	}

	if vcnId, ok := request.AdditionalProperties["VcnId"]; ok {
		listReq.VcnId = common.String(vcnId)
	}

	resp, err := client.ListSecurityLists(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list SecurityLists: %w", err)
	}

	nativeIDs := make([]string, 0, len(resp.Items))
	for _, sl := range resp.Items {
		nativeIDs = append(nativeIDs, *sl.Id)
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}
