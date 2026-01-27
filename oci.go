// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package main

import (
	"context"
	"fmt"

	"github.com/platform-engineering-labs/formae/pkg/plugin"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/client"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/config"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner"
	_ "github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner/containerengine"
	_ "github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner/core"
	_ "github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner/identity"
	_ "github.com/platform-engineering-labs/formae-plugin-oci/pkg/provisioner/objectstorage"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/util"
)

// Plugin implements the Formae ResourcePlugin interface.
// The SDK automatically provides identity methods (Name, Version, Namespace)
// and schema methods (SupportedResources, SchemaForResourceType) by reading
// formae-plugin.pkl and schema/pkl/ at startup.
type Plugin struct{}

// Compile-time check: Plugin must satisfy ResourcePlugin interface.
var _ plugin.ResourcePlugin = &Plugin{}

func (p *Plugin) RateLimit() plugin.RateLimitConfig {
	return plugin.RateLimitConfig{
		Scope:                            plugin.RateLimitScopeNamespace,
		MaxRequestsPerSecondForNamespace: 2,
	}
}

func (p *Plugin) DiscoveryFilters() []plugin.MatchFilter {
	return nil
}

func (p *Plugin) LabelConfig() plugin.LabelConfig {
	return plugin.LabelConfig{}
}

func (p *Plugin) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	cfg := config.FromTargetConfig(request.TargetConfig)
	clients, err := client.NewClients(ctx, cfg)
	if err != nil {
		return nil, err
	}

	prov := provisioner.Get(request.ResourceType, clients)
	if prov == nil {
		return nil, fmt.Errorf("no provisioner registered for resource type: %s", request.ResourceType)
	}

	result, err := prov.Create(ctx, request)
	if err != nil {
		// Try to convert OCI service errors to recoverable errors
		if handledResult, handledErr := util.HandleCreateError(err, request.ResourceType, request.ResourceType); handledErr == nil && handledResult != nil {
			return handledResult, nil
		}
		return nil, err
	}

	return result, nil
}

func (p *Plugin) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	cfg := config.FromTargetConfig(request.TargetConfig)
	clients, err := client.NewClients(ctx, cfg)
	if err != nil {
		return nil, err
	}

	prov := provisioner.Get(request.ResourceType, clients)
	if prov == nil {
		return nil, fmt.Errorf("no provisioner registered for resource type: %s", request.ResourceType)
	}

	result, err := prov.Update(ctx, request)
	if err != nil {
		// Try to convert OCI service errors to recoverable errors
		if handledResult, handledErr := util.HandleUpdateError(err, request.ResourceType, request.NativeID, request.ResourceType); handledErr == nil && handledResult != nil {
			return handledResult, nil
		}
		return nil, err
	}

	return result, nil
}

func (p *Plugin) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	cfg := config.FromTargetConfig(request.TargetConfig)
	clients, err := client.NewClients(ctx, cfg)
	if err != nil {
		return nil, err
	}

	prov := provisioner.Get(request.ResourceType, clients)
	if prov == nil {
		return nil, fmt.Errorf("no provisioner registered for resource type: %s", request.ResourceType)
	}

	result, err := prov.Delete(ctx, request)
	if err != nil {
		// Try to convert OCI service errors to recoverable errors
		if handledResult, handledErr := util.HandleDeleteError(err, request.ResourceType, request.NativeID, request.ResourceType); handledErr == nil && handledResult != nil {
			return handledResult, nil
		}
		return nil, err
	}

	return result, nil
}

func (p *Plugin) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	cfg := config.FromTargetConfig(request.TargetConfig)
	clients, err := client.NewClients(ctx, cfg)
	if err != nil {
		return nil, err
	}

	prov := provisioner.Get(request.ResourceType, clients)
	if prov == nil {
		return nil, fmt.Errorf("no provisioner registered for resource type: %s", request.ResourceType)
	}

	return prov.Status(ctx, request)
}

func (p *Plugin) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	cfg := config.FromTargetConfig(request.TargetConfig)
	clients, err := client.NewClients(ctx, cfg)
	if err != nil {
		return nil, err
	}

	prov := provisioner.Get(request.ResourceType, clients)
	if prov == nil {
		return nil, fmt.Errorf("no provisioner registered for resource type: %s", request.ResourceType)
	}

	return prov.Read(ctx, request)
}

func (p *Plugin) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	cfg := config.FromTargetConfig(request.TargetConfig)
	clients, err := client.NewClients(ctx, cfg)
	if err != nil {
		return nil, err
	}

	prov := provisioner.Get(request.ResourceType, clients)
	if prov == nil {
		return &resource.ListResult{
			NativeIDs: []string{},
		}, nil
	}

	return prov.List(ctx, request)
}

