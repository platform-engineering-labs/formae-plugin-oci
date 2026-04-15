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

	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner/identity"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompartmentRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newTestPolicyClient(t, map[route]canned{
			{"GET", "/20160918/compartments/ocid1.compartment..aaa"}: {200, newTestCompartmentBody("ACTIVE")},
		})
		p := identity.NewCompartmentProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.compartment..aaa"})
		require.NoError(t, err)
		assert.Empty(t, result.ErrorCode)

		var props map[string]any
		require.NoError(t, json.Unmarshal([]byte(result.Properties), &props))
		assert.Equal(t, "test-compartment", props["Name"])
		assert.Equal(t, "test description", props["Description"])
	})

	t.Run("not_found", func(t *testing.T) {
		svc := newTestPolicyClient(t, map[route]canned{
			{"GET", "/20160918/compartments/ocid1.compartment..missing"}: {404, `{"code":"NotAuthorizedOrNotFound","message":"not found"}`},
		})
		p := identity.NewCompartmentProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.compartment..missing"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})

	t.Run("terminal_state", func(t *testing.T) {
		svc := newTestPolicyClient(t, map[route]canned{
			{"GET", "/20160918/compartments/ocid1.compartment..aaa"}: {200, newTestCompartmentBody("DELETED")},
		})
		p := identity.NewCompartmentProvisionerWithSvc(svc)

		result, err := p.Read(context.Background(), &resource.ReadRequest{NativeID: "ocid1.compartment..aaa"})
		require.NoError(t, err)
		assert.Equal(t, resource.OperationErrorCodeNotFound, result.ErrorCode)
	})
}

func TestCompartmentCreate(t *testing.T) {
	svc := newTestPolicyClient(t, map[route]canned{
		{"POST", "/20160918/compartments"}: {200, newTestCompartmentBody("ACTIVE")},
	})
	p := identity.NewCompartmentProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{
		"CompartmentId": "ocid1.tenancy..xxx",
		"Name":          "test-compartment",
		"Description":   "test description",
	})
	require.NoError(t, err)

	result, err := p.Create(context.Background(), &resource.CreateRequest{
		ResourceType: "OCI::Identity::Compartment",
		Properties:   props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusInProgress, result.ProgressResult.OperationStatus)
	assert.Equal(t, "ocid1.compartment..aaa", result.ProgressResult.NativeID)
}

func TestCompartmentUpdate(t *testing.T) {
	svc := newTestPolicyClient(t, map[route]canned{
		{"GET", "/20160918/compartments/ocid1.compartment..aaa"}: {200, newTestCompartmentBody("ACTIVE")},
		{"PUT", "/20160918/compartments/ocid1.compartment..aaa"}: {200, newTestCompartmentBody("ACTIVE")},
	})
	p := identity.NewCompartmentProvisionerWithSvc(svc)

	props, err := json.Marshal(map[string]any{
		"Name":        "updated-compartment",
		"Description": "updated description",
	})
	require.NoError(t, err)

	result, err := p.Update(context.Background(), &resource.UpdateRequest{
		NativeID:          "ocid1.compartment..aaa",
		ResourceType:      "OCI::Identity::Compartment",
		DesiredProperties: props,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus)
	assert.Equal(t, "ocid1.compartment..aaa", result.ProgressResult.NativeID)
}

func TestCompartmentDelete(t *testing.T) {
	svc := newTestPolicyClient(t, map[route]canned{
		{"GET", "/20160918/compartments/ocid1.compartment..aaa"}:    {200, newTestCompartmentBody("ACTIVE")},
		{"DELETE", "/20160918/compartments/ocid1.compartment..aaa"}: {204, ""},
	})
	p := identity.NewCompartmentProvisionerWithSvc(svc)

	result, err := p.Delete(context.Background(), &resource.DeleteRequest{NativeID: "ocid1.compartment..aaa"})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusInProgress, result.ProgressResult.OperationStatus)
	assert.Equal(t, "ocid1.compartment..aaa", result.ProgressResult.NativeID)
}

func TestCompartmentList(t *testing.T) {
	svc := newTestPolicyClient(t, map[route]canned{
		{"GET", "/20160918/compartments"}: {200, fmt.Sprintf(`[%s]`, newTestCompartmentBody("ACTIVE"))},
	})
	p := identity.NewCompartmentProvisionerWithSvc(svc)

	result, err := p.List(context.Background(), &resource.ListRequest{
		ResourceType: "OCI::Identity::Compartment",
		AdditionalProperties: map[string]string{
			"CompartmentId": "ocid1.tenancy..xxx",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"ocid1.compartment..aaa"}, result.NativeIDs)
}

// Helpers

func newTestCompartmentBody(lifecycleState string) string {
	return fmt.Sprintf(`{
		"id": "ocid1.compartment..aaa",
		"compartmentId": "ocid1.tenancy..xxx",
		"name": "test-compartment",
		"description": "test description",
		"lifecycleState": %q
	}`, lifecycleState)
}
