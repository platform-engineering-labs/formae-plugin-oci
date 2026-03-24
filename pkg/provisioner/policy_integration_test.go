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
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	ociidentity "github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner/identity"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requirePolicyPermissions creates and immediately deletes a canary policy
// to verify the test user has IAM permissions. Skips the test if not.
func requirePolicyPermissions(t *testing.T, ctx context.Context) {
	t.Helper()
	clients := newTestClients(t)
	svc, err := clients.GetIdentityClient()
	require.NoError(t, err)

	compartmentID := getTestCompartmentID(t)
	canaryName := fmt.Sprintf("formae-test-canary-%d", time.Now().UnixNano())
	resp, err := svc.CreatePolicy(ctx, ociidentity.CreatePolicyRequest{
		CreatePolicyDetails: ociidentity.CreatePolicyDetails{
			CompartmentId: common.String(compartmentID),
			Name:          common.String(canaryName),
			Description:   common.String("permission check"),
			Statements:    []string{"allow group Administrators to inspect all-resources in tenancy"},
		},
	})
	if err != nil {
		t.Skipf("Skipping policy tests: insufficient IAM permissions in compartment %s: %v", compartmentID, err)
	}
	_, _ = svc.DeletePolicy(ctx, ociidentity.DeletePolicyRequest{PolicyId: resp.Id})
}

func TestPolicy_Create(t *testing.T) {
	ctx := context.Background()
	requirePolicyPermissions(t, ctx)
	compartmentID := getTestCompartmentID(t)
	clients := newTestClients(t)
	prov := identity.NewPolicyProvisioner(clients)

	policyName := fmt.Sprintf("formae-test-create-%d", time.Now().Unix())

	props := mustMarshalJSON(t, map[string]any{
		"CompartmentId": compartmentID,
		"Name":          policyName,
		"Description":   "Integration test policy",
		"Statements":    []string{"allow group TestGroup to inspect all-resources in tenancy"},
	})

	req := &resource.CreateRequest{
		ResourceType: "OCI::Identity::Policy",
		Properties:   props,
		TargetConfig: newTestTargetConfig(),
	}

	result, err := prov.Create(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.ProgressResult)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
	assert.NotEmpty(t, result.ProgressResult.NativeID)
	t.Logf("Created policy: %s", result.ProgressResult.NativeID)

	// Cleanup via SDK directly
	t.Cleanup(func() {
		svc, _ := clients.GetIdentityClient()
		_, _ = svc.DeletePolicy(ctx, ociidentity.DeletePolicyRequest{
			PolicyId: common.String(result.ProgressResult.NativeID),
		})
	})
}

func TestPolicy_Read(t *testing.T) {
	ctx := context.Background()
	requirePolicyPermissions(t, ctx)
	compartmentID := getTestCompartmentID(t)
	clients := newTestClients(t)
	prov := identity.NewPolicyProvisioner(clients)

	// Create via SDK
	svc, err := clients.GetIdentityClient()
	require.NoError(t, err)

	policyName := fmt.Sprintf("formae-test-read-%d", time.Now().Unix())
	createResp, err := svc.CreatePolicy(ctx, ociidentity.CreatePolicyRequest{
		CreatePolicyDetails: ociidentity.CreatePolicyDetails{
			CompartmentId: common.String(compartmentID),
			Name:          common.String(policyName),
			Description:   common.String("Read integration test"),
			Statements:    []string{"allow group TestGroup to inspect all-resources in tenancy"},
		},
	})
	require.NoError(t, err)
	nativeID := *createResp.Id
	t.Logf("Created policy via SDK: %s", nativeID)

	t.Cleanup(func() {
		_, _ = svc.DeletePolicy(ctx, ociidentity.DeletePolicyRequest{
			PolicyId: common.String(nativeID),
		})
	})

	// Read via provisioner
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		TargetConfig: newTestTargetConfig(),
	})
	require.NoError(t, err)
	require.NotNil(t, readResult)
	assert.Equal(t, "OCI::Identity::Policy", readResult.ResourceType)
	assert.Empty(t, readResult.ErrorCode)

	var props map[string]any
	err = json.Unmarshal([]byte(readResult.Properties), &props)
	require.NoError(t, err)
	assert.Equal(t, policyName, props["Name"])
	assert.Equal(t, compartmentID, props["CompartmentId"])
	assert.Equal(t, "Read integration test", props["Description"])
}

func TestPolicy_Update(t *testing.T) {
	ctx := context.Background()
	requirePolicyPermissions(t, ctx)
	compartmentID := getTestCompartmentID(t)
	clients := newTestClients(t)
	prov := identity.NewPolicyProvisioner(clients)

	// Create via SDK
	svc, err := clients.GetIdentityClient()
	require.NoError(t, err)

	policyName := fmt.Sprintf("formae-test-update-%d", time.Now().Unix())
	createResp, err := svc.CreatePolicy(ctx, ociidentity.CreatePolicyRequest{
		CreatePolicyDetails: ociidentity.CreatePolicyDetails{
			CompartmentId: common.String(compartmentID),
			Name:          common.String(policyName),
			Description:   common.String("Before update"),
			Statements:    []string{"allow group TestGroup to inspect all-resources in tenancy"},
		},
	})
	require.NoError(t, err)
	nativeID := *createResp.Id

	t.Cleanup(func() {
		_, _ = svc.DeletePolicy(ctx, ociidentity.DeletePolicyRequest{
			PolicyId: common.String(nativeID),
		})
	})

	// Update via provisioner using DesiredProperties (full replacement)
	desiredProps := mustMarshalJSON(t, map[string]any{
		"CompartmentId": compartmentID,
		"Name":          policyName,
		"Description":   "After update",
		"Statements":    []string{"allow group TestGroup to read all-resources in tenancy"},
	})

	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          nativeID,
		ResourceType:      "OCI::Identity::Policy",
		DesiredProperties: desiredProps,
		TargetConfig:      newTestTargetConfig(),
	})
	require.NoError(t, err)
	require.NotNil(t, updateResult)
	assert.Equal(t, resource.OperationStatusSuccess, updateResult.ProgressResult.OperationStatus)
	assert.Equal(t, nativeID, updateResult.ProgressResult.NativeID)

	// Verify via SDK
	getResp, err := svc.GetPolicy(ctx, ociidentity.GetPolicyRequest{
		PolicyId: common.String(nativeID),
	})
	require.NoError(t, err)
	assert.Equal(t, "After update", *getResp.Description)
	assert.Contains(t, getResp.Statements, "allow group TestGroup to read all-resources in tenancy")
}

func TestPolicy_Delete(t *testing.T) {
	ctx := context.Background()
	requirePolicyPermissions(t, ctx)
	compartmentID := getTestCompartmentID(t)
	clients := newTestClients(t)
	prov := identity.NewPolicyProvisioner(clients)

	// Create via SDK
	svc, err := clients.GetIdentityClient()
	require.NoError(t, err)

	policyName := fmt.Sprintf("formae-test-delete-%d", time.Now().Unix())
	createResp, err := svc.CreatePolicy(ctx, ociidentity.CreatePolicyRequest{
		CreatePolicyDetails: ociidentity.CreatePolicyDetails{
			CompartmentId: common.String(compartmentID),
			Name:          common.String(policyName),
			Description:   common.String("Delete integration test"),
			Statements:    []string{"allow group TestGroup to inspect all-resources in tenancy"},
		},
	})
	require.NoError(t, err)
	nativeID := *createResp.Id

	// Safety cleanup in case delete fails
	t.Cleanup(func() {
		_, _ = svc.DeletePolicy(ctx, ociidentity.DeletePolicyRequest{
			PolicyId: common.String(nativeID),
		})
	})

	// Delete via provisioner
	deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{
		NativeID:     nativeID,
		TargetConfig: newTestTargetConfig(),
	})
	require.NoError(t, err)
	require.NotNil(t, deleteResult)
	assert.Equal(t, resource.OperationStatusSuccess, deleteResult.ProgressResult.OperationStatus)

	// Verify deleted via Read
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		TargetConfig: newTestTargetConfig(),
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationErrorCodeNotFound, readResult.ErrorCode)
}

func TestPolicy_List(t *testing.T) {
	ctx := context.Background()
	requirePolicyPermissions(t, ctx)
	compartmentID := getTestCompartmentID(t)
	clients := newTestClients(t)
	prov := identity.NewPolicyProvisioner(clients)

	// Create via SDK
	svc, err := clients.GetIdentityClient()
	require.NoError(t, err)

	policyName := fmt.Sprintf("formae-test-list-%d", time.Now().Unix())
	createResp, err := svc.CreatePolicy(ctx, ociidentity.CreatePolicyRequest{
		CreatePolicyDetails: ociidentity.CreatePolicyDetails{
			CompartmentId: common.String(compartmentID),
			Name:          common.String(policyName),
			Description:   common.String("List integration test"),
			Statements:    []string{"allow group TestGroup to inspect all-resources in tenancy"},
		},
	})
	require.NoError(t, err)
	nativeID := *createResp.Id

	t.Cleanup(func() {
		_, _ = svc.DeletePolicy(ctx, ociidentity.DeletePolicyRequest{
			PolicyId: common.String(nativeID),
		})
	})

	// List via provisioner
	listResult, err := prov.List(ctx, &resource.ListRequest{
		ResourceType: "OCI::Identity::Policy",
		TargetConfig: newTestTargetConfig(),
		AdditionalProperties: map[string]string{
			"CompartmentId": compartmentID,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, listResult)
	assert.NotEmpty(t, listResult.NativeIDs, "List should return at least one policy")

	// Verify our policy is in the list
	found := false
	for _, id := range listResult.NativeIDs {
		if id == nativeID {
			found = true
			break
		}
	}
	assert.True(t, found, "Created policy %s should appear in list results", nativeID)
}
