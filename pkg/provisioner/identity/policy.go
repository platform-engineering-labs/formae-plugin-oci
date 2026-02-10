// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/client"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/util"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

type PolicyProvisioner struct {
	clients *client.Clients
}

var _ provisioner.Provisioner = &PolicyProvisioner{}

func init() {
	provisioner.Register("OCI::Identity::Policy", NewPolicyProvisioner)
}

func NewPolicyProvisioner(clients *client.Clients) provisioner.Provisioner {
	return &PolicyProvisioner{clients: clients}
}

func (p *PolicyProvisioner) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	svc, err := p.clients.GetIdentityClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Identity client: %w", err)
	}

	var props map[string]any
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	statements, _ := util.ExtractStringSlice(props, "Statements")

	createDetails := identity.CreatePolicyDetails{
		CompartmentId: common.String(props["CompartmentId"].(string)),
		Name:          common.String(props["Name"].(string)),
		Description:   common.String(props["Description"].(string)),
		Statements:    statements,
	}

	if versionDate, ok := util.ExtractString(props, "VersionDate"); ok {
		if t, err := time.Parse("2006-01-02", versionDate); err == nil {
			createDetails.VersionDate = &common.SDKDate{Date: t}
		}
	}
	if freeformTags, ok := util.ExtractFreeformTags(props, "FreeformTags"); ok {
		createDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractDefinedTags(props, "DefinedTags"); ok {
		createDetails.DefinedTags = definedTags
	}

	createReq := identity.CreatePolicyRequest{
		CreatePolicyDetails: createDetails,
	}

	resp, err := svc.CreatePolicy(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create Policy: %w", err)
	}

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCreate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        *resp.Id,
		},
	}, nil
}

func (p *PolicyProvisioner) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	svc, err := p.clients.GetIdentityClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Identity client: %w", err)
	}

	getReq := identity.GetPolicyRequest{
		PolicyId: common.String(request.NativeID),
	}

	resp, err := svc.GetPolicy(ctx, getReq)
	if err != nil {
		if serviceErr, ok := common.IsServiceError(err); ok && serviceErr.GetHTTPStatusCode() == 404 {
			return &resource.ReadResult{
				ResourceType: "OCI::Identity::Policy",
				ErrorCode:    resource.OperationErrorCodeNotFound,
			}, nil
		}
		return nil, fmt.Errorf("failed to read Policy: %w", err)
	}

	properties := buildPolicyProperties(resp.Policy)

	propBytes, err := json.Marshal(properties)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Policy properties: %w", err)
	}

	return &resource.ReadResult{
		ResourceType: "OCI::Identity::Policy",
		Properties:   string(propBytes),
	}, nil
}

func (p *PolicyProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	svc, err := p.clients.GetIdentityClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Identity client: %w", err)
	}

	props, err := util.ApplyPatchDocument(ctx, request, p.Read)
	if err != nil {
		return nil, err
	}

	updateDetails := identity.UpdatePolicyDetails{}

	if description, ok := util.ExtractString(props, "Description"); ok {
		updateDetails.Description = common.String(description)
	}
	if statements, ok := util.ExtractStringSlice(props, "Statements"); ok {
		updateDetails.Statements = statements
	}
	if versionDate, ok := util.ExtractString(props, "VersionDate"); ok {
		if t, err := time.Parse("2006-01-02", versionDate); err == nil {
			updateDetails.VersionDate = &common.SDKDate{Date: t}
		}
	}
	if freeformTags, ok := util.ExtractFreeformTags(props, "FreeformTags"); ok {
		updateDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractDefinedTags(props, "DefinedTags"); ok {
		updateDetails.DefinedTags = definedTags
	}

	updateReq := identity.UpdatePolicyRequest{
		PolicyId:            common.String(request.NativeID),
		UpdatePolicyDetails: updateDetails,
	}

	resp, err := svc.UpdatePolicy(ctx, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update Policy: %w", err)
	}

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationUpdate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        *resp.Id,
		},
	}, nil
}

func (p *PolicyProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	svc, err := p.clients.GetIdentityClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Identity client: %w", err)
	}

	readReq := &resource.ReadRequest{
		NativeID: request.NativeID,
	}
	readRes, err := p.Read(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to read Policy before delete: %w", err)
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

	deleteReq := identity.DeletePolicyRequest{
		PolicyId: common.String(request.NativeID),
	}

	_, err = svc.DeletePolicy(ctx, deleteReq)
	if err != nil {
		return nil, fmt.Errorf("failed to delete Policy: %w", err)
	}

	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationDelete,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (p *PolicyProvisioner) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			RequestID:       request.RequestID,
		},
	}, nil
}

func (p *PolicyProvisioner) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	svc, err := p.clients.GetIdentityClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Identity client: %w", err)
	}

	compartmentId, ok := request.AdditionalProperties["CompartmentId"]
	if !ok {
		return nil, fmt.Errorf("CompartmentId is required for listing Policies")
	}

	listReq := identity.ListPoliciesRequest{
		CompartmentId: common.String(compartmentId),
	}

	resp, err := svc.ListPolicies(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list Policies: %w", err)
	}

	nativeIDs := make([]string, 0, len(resp.Items))
	for _, policy := range resp.Items {
		nativeIDs = append(nativeIDs, *policy.Id)
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

func buildPolicyProperties(policy identity.Policy) map[string]any {
	properties := map[string]any{
		"Id": *policy.Id,
	}

	if policy.CompartmentId != nil {
		properties["CompartmentId"] = *policy.CompartmentId
	}
	if policy.Name != nil {
		properties["Name"] = *policy.Name
	}
	if policy.Description != nil {
		properties["Description"] = *policy.Description
	}
	if policy.Statements != nil {
		properties["Statements"] = policy.Statements
	}
	if policy.VersionDate != nil {
		properties["VersionDate"] = policy.VersionDate.String()
	}
	if policy.FreeformTags != nil {
		properties["FreeformTags"] = util.FreeformTagsToList(policy.FreeformTags)
	}
	if policy.DefinedTags != nil {
		properties["DefinedTags"] = util.DefinedTagsToList(policy.DefinedTags)
	}

	return properties
}
