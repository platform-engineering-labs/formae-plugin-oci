// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package identity

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/client"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/util"
)

type CompartmentProvisioner struct {
	clients *client.Clients
}

var _ provisioner.Provisioner = &CompartmentProvisioner{}

func init() {
	provisioner.Register("OCI::Identity::Compartment", NewCompartmentProvisioner)
}

func NewCompartmentProvisioner(clients *client.Clients) provisioner.Provisioner {
	return &CompartmentProvisioner{clients: clients}
}

func (p *CompartmentProvisioner) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	client, err := p.clients.GetIdentityClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Identity client: %w", err)
	}

	var props map[string]any
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	createDetails := identity.CreateCompartmentDetails{
		CompartmentId: common.String(props["CompartmentId"].(string)),
		Name:          common.String(props["Name"].(string)),
		Description:   common.String(props["Description"].(string)),
	}

	if freeformTags, ok := util.ExtractTag(props, "FreeformTags"); ok {
		createDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractNestedTag(props, "DefinedTags"); ok {
		createDetails.DefinedTags = definedTags
	}

	createReq := identity.CreateCompartmentRequest{
		CreateCompartmentDetails: createDetails,
	}

	resp, err := client.CreateCompartment(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create Compartment: %w", err)
	}

	// Build properties from Create response
	properties := map[string]any{
		"Id": *resp.Id,
	}

	// CompartmentId for root compartment may be nil, use Id as fallback
	if resp.CompartmentId != nil {
		properties["CompartmentId"] = *resp.CompartmentId
	} else {
		fmt.Printf("DEBUG: CompartmentId is nil for compartment %s, using Id as fallback\n", *resp.Id)
		properties["CompartmentId"] = *resp.Id
	}

	if resp.Name != nil {
		properties["Name"] = *resp.Name
	}
	if resp.Description != nil {
		properties["Description"] = *resp.Description
	}
	if resp.FreeformTags != nil {
		properties["FreeformTags"] = resp.FreeformTags
	}
	if resp.DefinedTags != nil {
		properties["DefinedTags"] = resp.DefinedTags
	}

	propertiesBytes, err := json.Marshal(properties)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal properties: %w", err)
	}

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCreate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           *resp.Id,
			ResourceProperties: json.RawMessage(propertiesBytes),
		},
	}, nil
}

func (p *CompartmentProvisioner) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	client, err := p.clients.GetIdentityClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Identity client: %w", err)
	}

	getReq := identity.GetCompartmentRequest{
		CompartmentId: common.String(request.NativeID),
	}

	resp, err := client.GetCompartment(ctx, getReq)
	if err != nil {
		if serviceErr, ok := common.IsServiceError(err); ok && serviceErr.GetHTTPStatusCode() == 404 {
			return &resource.ReadResult{
				ResourceType: "OCI::Identity::Compartment",
				ErrorCode:    resource.OperationErrorCodeNotFound,
			}, nil
		}
		return nil, fmt.Errorf("failed to read Compartment: %w", err)
	}

	properties := map[string]any{
		"Id": *resp.Id,
	}

	// CompartmentId for root compartment may be nil, use Id as fallback
	if resp.CompartmentId != nil {
		properties["CompartmentId"] = *resp.CompartmentId
	} else {
		fmt.Printf("DEBUG: CompartmentId is nil for compartment %s, using Id as fallback\n", *resp.Id)
		properties["CompartmentId"] = *resp.Id
	}

	if resp.Name != nil {
		properties["Name"] = *resp.Name
	}
	if resp.Description != nil {
		properties["Description"] = *resp.Description
	}
	if resp.FreeformTags != nil {
		properties["FreeformTags"] = resp.FreeformTags
	}
	if resp.DefinedTags != nil {
		properties["DefinedTags"] = resp.DefinedTags
	}

	propertiesBytes, err := json.Marshal(properties)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal properties: %w", err)
	}

	return &resource.ReadResult{
		ResourceType: "OCI::Identity::Compartment",
		Properties:   string(propertiesBytes),
	}, nil
}

func (p *CompartmentProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	client, err := p.clients.GetIdentityClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Identity client: %w", err)
	}

	props, err := util.ApplyPatchDocument(ctx, request, p.Read)
	if err != nil {
		return nil, err
	}

	updateDetails := identity.UpdateCompartmentDetails{}

	// Name and Description are updatable
	if name, ok := util.ExtractString(props, "Name"); ok {
		updateDetails.Name = common.String(name)
	}
	if description, ok := util.ExtractString(props, "Description"); ok {
		updateDetails.Description = common.String(description)
	}
	if freeformTags, ok := util.ExtractTag(props, "FreeformTags"); ok {
		updateDetails.FreeformTags = freeformTags
	}
	if definedTags, ok := util.ExtractNestedTag(props, "DefinedTags"); ok {
		updateDetails.DefinedTags = definedTags
	}

	updateReq := identity.UpdateCompartmentRequest{
		CompartmentId:            common.String(request.NativeID),
		UpdateCompartmentDetails: updateDetails,
	}

	resp, err := client.UpdateCompartment(ctx, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update Compartment: %w", err)
	}

	properties := map[string]any{
		"Id": *resp.Id,
	}

	// CompartmentId for root compartment may be nil, use Id as fallback
	if resp.CompartmentId != nil {
		properties["CompartmentId"] = *resp.CompartmentId
	} else {
		fmt.Printf("DEBUG: CompartmentId is nil for compartment %s, using Id as fallback\n", *resp.Id)
		properties["CompartmentId"] = *resp.Id
	}

	if resp.Name != nil {
		properties["Name"] = *resp.Name
	}
	if resp.Description != nil {
		properties["Description"] = *resp.Description
	}
	if resp.FreeformTags != nil {
		properties["FreeformTags"] = resp.FreeformTags
	}
	if resp.DefinedTags != nil {
		properties["DefinedTags"] = resp.DefinedTags
	}

	propertiesBytes, err := json.Marshal(properties)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal properties: %w", err)
	}

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationUpdate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           *resp.Id,
			ResourceProperties: json.RawMessage(propertiesBytes),
		},
	}, nil
}

func (p *CompartmentProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	client, err := p.clients.GetIdentityClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Identity client: %w", err)
	}

	// Check if exists
	readReq := &resource.ReadRequest{
		NativeID: request.NativeID,
	}
	readRes, err := p.Read(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to read Compartment before delete: %w", err)
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

	deleteReq := identity.DeleteCompartmentRequest{
		CompartmentId: common.String(request.NativeID),
	}

	_, err = client.DeleteCompartment(ctx, deleteReq)
	if err != nil {
		return nil, fmt.Errorf("failed to delete Compartment: %w", err)
	}

	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationDelete,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (p *CompartmentProvisioner) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	// Compartment operations are synchronous, no status check needed
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			RequestID:       request.RequestID,
		},
	}, nil
}

func (p *CompartmentProvisioner) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	client, err := p.clients.GetIdentityClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Identity client: %w", err)
	}

	// Get CompartmentId from request, or use tenancy OCID as root
	var compartmentId string
	if id, ok := request.AdditionalProperties["CompartmentId"]; ok {
		compartmentId = id
	} else {
		// No CompartmentId provided - use tenancy OCID from config provider
		provider := p.clients.GetConfigurationProvider()
		tenancyID, err := provider.TenancyOCID()
		if err != nil {
			return nil, fmt.Errorf("failed to get tenancy OCID for root compartment discovery: %w", err)
		}
		compartmentId = tenancyID
	}

	listReq := identity.ListCompartmentsRequest{
		CompartmentId:          common.String(compartmentId),
		CompartmentIdInSubtree: common.Bool(false), // Only direct children - natural tree traversal
		AccessLevel:            identity.ListCompartmentsAccessLevelAccessible,
	}

	// DEBUG logging
	fmt.Printf("DEBUG Compartment List: compartmentId=%s, subtree=%v\n", compartmentId, *listReq.CompartmentIdInSubtree)

	resp, err := client.ListCompartments(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list Compartments: %w", err)
	}

	// DEBUG logging
	fmt.Printf("DEBUG Compartment List: found %d child compartments\n", len(resp.Items))

	var nativeIDs []string

	// If no CompartmentId was provided in the request, we're at the root (tenancy level)
	// Include the root compartment (tenancy) itself as a discoverable resource
	if _, ok := request.AdditionalProperties["CompartmentId"]; !ok {
		nativeIDs = append(nativeIDs, compartmentId)
		fmt.Printf("DEBUG Compartment List: added root compartment (tenancy) id=%s\n", compartmentId)
	}
	for _, compartment := range resp.Items {
		nativeIDs = append(nativeIDs, *compartment.Id)
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}
