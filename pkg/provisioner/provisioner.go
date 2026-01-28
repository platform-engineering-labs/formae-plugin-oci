// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package provisioner

import (
	"context"

	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

// Provisioner defines the interface for OCI resource provisioners
type Provisioner interface {
	Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error)
	Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error)
	Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error)
	Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error)
	Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error)
	List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error)
}
