// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

//go:build integration

package provisioner_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner/core"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNSGSecurityRuleRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/networkSecurityGroups/ocid1.nsg..aaa/securityRules"}: {200, fmt.Sprintf(`[%s]`, newTestNSGSecurityRuleBody())},
		})
		p := core.NewNetworkSecurityGroupSecurityRuleProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.nsg..aaa/rule-001"})
		require.NoError(t, err)
		assert.Empty(t, result.ErrorCode)

		var props map[string]any
		require.NoError(t, json.Unmarshal([]byte(result.Properties), &props))
		assert.Equal(t, "INGRESS", props["Direction"])
		assert.Equal(t, "6", props["Protocol"])
	})

	t.Run("not_found", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/networkSecurityGroups/ocid1.nsg..aaa/securityRules"}: {200, `[]`},
		})
		p := core.NewNetworkSecurityGroupSecurityRuleProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.nsg..aaa/rule-missing"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})

	t.Run("nsg_not_found", func(t *testing.T) {
		svc := newTestVirtualNetworkClient(t, map[route]canned{
			{"GET", "/20160918/networkSecurityGroups/ocid1.nsg..missing/securityRules"}: {404, `{"code":"NotAuthorizedOrNotFound","message":"not found"}`},
		})
		p := core.NewNetworkSecurityGroupSecurityRuleProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.nsg..missing/rule-001"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})
}

func TestNSGSecurityRuleCreate(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"POST", "/20160918/networkSecurityGroups/ocid1.nsg..aaa/actions/addSecurityRules"}: {
			200,
			fmt.Sprintf(`{"securityRules": [%s]}`, newTestNSGSecurityRuleBody()),
		},
	})
	p := core.NewNetworkSecurityGroupSecurityRuleProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{
		"NetworkSecurityGroupId": "ocid1.nsg..aaa",
		"Direction":              "INGRESS",
		"Protocol":               "6",
		"Source":                 "10.0.0.0/16",
		"SourceType":             "CIDR_BLOCK",
		"Description":            "Allow TCP from VCN",
	})
	require.NoError(t, err)

	result, err := p.Create(context.Background(), &resource.CreateRequest{
		ResourceType: "OCI::Core::NetworkSecurityGroupSecurityRule",
		Properties:   props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
	assert.Equal(t, "ocid1.nsg..aaa/rule-001", result.ProgressResult.NativeID)
}

func TestNSGSecurityRuleUpdate(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{})
	p := core.NewNetworkSecurityGroupSecurityRuleProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{"Description": "updated"})
	require.NoError(t, err)

	_, err = p.Update(context.Background(), &resource.UpdateRequest{
		NativeID:          "ocid1.nsg..aaa/rule-001",
		ResourceType:      "OCI::Core::NetworkSecurityGroupSecurityRule",
		DesiredProperties: props,
	})
	require.Error(t, err, "Update should return an error for NSG security rules")
	assert.Contains(t, err.Error(), "update not supported")
}

func TestNSGSecurityRuleDelete(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"POST", "/20160918/networkSecurityGroups/ocid1.nsg..aaa/actions/removeSecurityRules"}: {204, ""},
	})
	p := core.NewNetworkSecurityGroupSecurityRuleProvisionerWithSvc(svc)

	result, err := p.Delete(context.Background(), &resource.DeleteRequest{NativeID: "ocid1.nsg..aaa/rule-001"})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
}

func TestNSGSecurityRuleList(t *testing.T) {
	svc := newTestVirtualNetworkClient(t, map[route]canned{
		{"GET", "/20160918/networkSecurityGroups/ocid1.nsg..aaa/securityRules"}: {200, fmt.Sprintf(`[%s]`, newTestNSGSecurityRuleBody())},
	})
	p := core.NewNetworkSecurityGroupSecurityRuleProvisionerWithSvc(svc)

	result, err := p.List(context.Background(), &resource.ListRequest{
		ResourceType: "OCI::Core::NetworkSecurityGroupSecurityRule",
		AdditionalProperties: map[string]string{
			"NetworkSecurityGroupId": "ocid1.nsg..aaa",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"ocid1.nsg..aaa/rule-001"}, result.NativeIDs)
}

// Helpers

func newTestNSGSecurityRuleBody() string {
	return `{
		"id": "rule-001",
		"direction": "INGRESS",
		"protocol": "6",
		"source": "10.0.0.0/16",
		"sourceType": "CIDR_BLOCK",
		"description": "Allow TCP from VCN",
		"isStateless": false,
		"isValid": true
	}`
}
