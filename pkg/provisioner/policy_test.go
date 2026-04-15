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

	ociidentity "github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner/identity"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicyRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newTestPolicyClient(t, map[route]canned{
			{"GET", "/20160918/policies/ocid1.policy..aaa"}: {200, newTestPolicyBody("ACTIVE")},
		})
		p := identity.NewPolicyProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.policy..aaa"})
		require.NoError(t, err)
		assert.Empty(t, result.ErrorCode)

		var props map[string]any
		require.NoError(t, json.Unmarshal([]byte(result.Properties), &props))
		assert.Equal(t, "test-policy", props["Name"])
		assert.Equal(t, "test", props["Description"])
	})

	t.Run("not_found", func(t *testing.T) {
		svc := newTestPolicyClient(t, map[route]canned{
			{"GET", "/20160918/policies/ocid1.policy..missing"}: {404, `{"code":"NotAuthorizedOrNotFound","message":"not found"}`},
		})
		p := identity.NewPolicyProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.policy..missing"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})

	t.Run("terminal_state", func(t *testing.T) {
		svc := newTestPolicyClient(t, map[route]canned{
			{"GET", "/20160918/policies/ocid1.policy..aaa"}: {200, newTestPolicyBody("DELETED")},
		})
		p := identity.NewPolicyProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.policy..aaa"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})
}

func TestPolicyCreate(t *testing.T) {
	svc := newTestPolicyClient(t, map[route]canned{
		{"POST", "/20160918/policies"}: {200, newTestPolicyBody("ACTIVE")},
	})
	p := identity.NewPolicyProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{
		"CompartmentId": "ocid1.compartment..xxx",
		"Name":          "test-policy",
		"Description":   "test",
		"Statements":    []string{"allow group Admins to inspect all-resources in tenancy"},
	})
	require.NoError(t, err)

	result, err := p.Create(context.Background(), &resource.CreateRequest{
		ResourceType: "OCI::Identity::Policy",
		Properties:   props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
	assert.Equal(t, "ocid1.policy..aaa", result.ProgressResult.NativeID)
}

func TestPolicyUpdate(t *testing.T) {
	svc := newTestPolicyClient(t, map[route]canned{
		{"PUT", "/20160918/policies/ocid1.policy..aaa"}: {200, newTestPolicyBody("ACTIVE")},
	})
	p := identity.NewPolicyProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{
		"Description": "updated",
		"Statements":  []string{"allow group Admins to read all-resources in tenancy"},
	})
	require.NoError(t, err)

	result, err := p.Update(context.Background(), &resource.UpdateRequest{
		NativeID:          "ocid1.policy..aaa",
		ResourceType:      "OCI::Identity::Policy",
		DesiredProperties: props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
	assert.Equal(t, "ocid1.policy..aaa", result.ProgressResult.NativeID)
}

func TestPolicyDelete(t *testing.T) {
	svc := newTestPolicyClient(t, map[route]canned{
		{"GET", "/20160918/policies/ocid1.policy..aaa"}:    {200, newTestPolicyBody("ACTIVE")},
		{"DELETE", "/20160918/policies/ocid1.policy..aaa"}: {204, ""},
	})
	p := identity.NewPolicyProvisionerWithSvc(svc)

	result, err := p.Delete(context.Background(), &resource.DeleteRequest{
		NativeID: "ocid1.policy..aaa",
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
	assert.Equal(t, "ocid1.policy..aaa", result.ProgressResult.NativeID)
}

func TestPolicyList(t *testing.T) {
	svc := newTestPolicyClient(t, map[route]canned{
		{"GET", "/20160918/policies"}: {200, fmt.Sprintf(`[%s]`, newTestPolicyBody("ACTIVE"))},
	})
	p := identity.NewPolicyProvisionerWithSvc(svc)

	result, err := p.List(context.Background(), &resource.ListRequest{
		ResourceType: "OCI::Identity::Policy",
		AdditionalProperties: map[string]string{
			"CompartmentId": "ocid1.compartment..xxx",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"ocid1.policy..aaa"}, result.NativeIDs)
}

// Helpers

func newTestPolicyClient(t *testing.T, responses map[route]canned) *ociidentity.IdentityClient {
	t.Helper()
	host := newTestDispatcher(t, responses)
	c, err := ociidentity.NewIdentityClientWithConfigurationProvider(fakeOCIConfigProvider(t))
	require.NoError(t, err)
	applyTestRetryPolicy(&c)
	c.Host = host
	return &c
}

func newTestPolicyBody(lifecycleState string) string {
	return fmt.Sprintf(`{
		"id": "ocid1.policy..aaa",
		"compartmentId": "ocid1.compartment..xxx",
		"name": "test-policy",
		"description": "test",
		"statements": ["allow group Admins to inspect all-resources in tenancy"],
		"lifecycleState": %q
	}`, lifecycleState)
}
