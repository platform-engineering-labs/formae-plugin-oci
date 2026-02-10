// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package provisioner

import (
	"fmt"

	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/client"
)

// ProvisionerFactory creates a new Provisioner instance
type ProvisionerFactory func(clients *client.Clients) Provisioner

var (
	provisioners = make(map[string]ProvisionerFactory)
)

// Register adds a provisioner factory for a resource type
func Register(resourceType string, factory ProvisionerFactory) {
	provisioners[resourceType] = factory
}

// Get returns a provisioner for the given resource type
func Get(resourceType string, clients *client.Clients) Provisioner {
	factory, ok := provisioners[resourceType]
	if !ok {
		return nil
	}
	return &readAfterWrite{inner: factory(clients)}
}

// GetFactory returns the factory function for a resource type (for testing)
func GetFactory(resourceType string) (ProvisionerFactory, error) {
	factory, ok := provisioners[resourceType]
	if !ok {
		return nil, fmt.Errorf("no provisioner registered for resource type: %s", resourceType)
	}
	return factory, nil
}

// ListRegistered returns all registered resource types
func ListRegistered() []string {
	types := make([]string, 0, len(provisioners))
	for t := range provisioners {
		types = append(types, t)
	}
	return types
}
