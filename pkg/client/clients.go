// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package client

import (
	"context"
	"sync"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/containerengine"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/config"
)

// Clients manages OCI service clients with lazy initialization
type Clients struct {
	provider common.ConfigurationProvider

	mu              sync.Mutex
	virtualNetwork  *core.VirtualNetworkClient
	objectStorage   *objectstorage.ObjectStorageClient
	identity        *identity.IdentityClient
	containerEngine *containerengine.ContainerEngineClient
}

// NewClients creates a new Clients instance with the given configuration
func NewClients(ctx context.Context, cfg *config.Config) (*Clients, error) {
	provider, err := cfg.ToConfigProvider(ctx)
	if err != nil {
		return nil, err
	}

	return &Clients{provider: provider}, nil
}

// GetVirtualNetworkClient returns a cached or newly created VirtualNetworkClient
func (c *Clients) GetVirtualNetworkClient() (*core.VirtualNetworkClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.virtualNetwork == nil {
		client, err := core.NewVirtualNetworkClientWithConfigurationProvider(c.provider)
		if err != nil {
			return nil, err
		}
		c.virtualNetwork = &client
	}
	return c.virtualNetwork, nil
}

// GetObjectStorageClient returns a cached or newly created ObjectStorageClient
func (c *Clients) GetObjectStorageClient() (*objectstorage.ObjectStorageClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.objectStorage == nil {
		client, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(c.provider)
		if err != nil {
			return nil, err
		}
		c.objectStorage = &client
	}
	return c.objectStorage, nil
}

// GetIdentityClient returns a cached or newly created IdentityClient
func (c *Clients) GetIdentityClient() (*identity.IdentityClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.identity == nil {
		client, err := identity.NewIdentityClientWithConfigurationProvider(c.provider)
		if err != nil {
			return nil, err
		}
		c.identity = &client
	}
	return c.identity, nil
}

// GetContainerEngineClient returns a cached or newly created ContainerEngineClient
func (c *Clients) GetContainerEngineClient() (*containerengine.ContainerEngineClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.containerEngine == nil {
		client, err := containerengine.NewContainerEngineClientWithConfigurationProvider(c.provider)
		if err != nil {
			return nil, err
		}
		c.containerEngine = &client
	}
	return c.containerEngine, nil
}

// GetConfigurationProvider returns the underlying OCI ConfigurationProvider
func (c *Clients) GetConfigurationProvider() common.ConfigurationProvider {
	return c.provider
}
