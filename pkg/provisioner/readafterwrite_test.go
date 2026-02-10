// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package provisioner

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

type mockProvisioner struct {
	createResult *resource.CreateResult
	createErr    error
	updateResult *resource.UpdateResult
	updateErr    error
	readResult   *resource.ReadResult
	readErr      error

	readCalled bool
}

func (m *mockProvisioner) Create(_ context.Context, _ *resource.CreateRequest) (*resource.CreateResult, error) {
	return m.createResult, m.createErr
}

func (m *mockProvisioner) Update(_ context.Context, _ *resource.UpdateRequest) (*resource.UpdateResult, error) {
	return m.updateResult, m.updateErr
}

func (m *mockProvisioner) Read(_ context.Context, _ *resource.ReadRequest) (*resource.ReadResult, error) {
	m.readCalled = true
	return m.readResult, m.readErr
}

func (m *mockProvisioner) Delete(_ context.Context, _ *resource.DeleteRequest) (*resource.DeleteResult, error) {
	return nil, nil
}

func (m *mockProvisioner) Status(_ context.Context, _ *resource.StatusRequest) (*resource.StatusResult, error) {
	return nil, nil
}

func (m *mockProvisioner) List(_ context.Context, _ *resource.ListRequest) (*resource.ListResult, error) {
	return nil, nil
}

func TestReadAfterWrite_Create_SyncSuccess(t *testing.T) {
	inner := &mockProvisioner{
		createResult: &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				OperationStatus:    resource.OperationStatusSuccess,
				NativeID:           "ocid1.volume.oc1..abc",
				ResourceProperties: json.RawMessage(`{"Id":"ocid1.volume.oc1..abc"}`),
			},
		},
		readResult: &resource.ReadResult{
			Properties: `{"Id":"ocid1.volume.oc1..abc","CompartmentId":"ocid1.compartment.oc1..xyz","SizeInGBs":50}`,
		},
	}

	w := &readAfterWrite{inner: inner}
	result, err := w.Create(context.Background(), &resource.CreateRequest{
		ResourceType: "OCI::Core::Volume",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !inner.readCalled {
		t.Fatal("expected Read to be called after sync Create success")
	}

	// Properties should be replaced with the full set from Read
	got := string(result.ProgressResult.ResourceProperties)
	want := `{"Id":"ocid1.volume.oc1..abc","CompartmentId":"ocid1.compartment.oc1..xyz","SizeInGBs":50}`
	if got != want {
		t.Errorf("properties = %s, want %s", got, want)
	}
}

func TestReadAfterWrite_Create_AsyncSkipped(t *testing.T) {
	inner := &mockProvisioner{
		createResult: &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				OperationStatus: resource.OperationStatusInProgress,
				NativeID:        "ocid1.instance.oc1..abc",
				RequestID:       "ocid1.instance.oc1..abc",
			},
		},
	}

	w := &readAfterWrite{inner: inner}
	_, err := w.Create(context.Background(), &resource.CreateRequest{
		ResourceType: "OCI::Core::Instance",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inner.readCalled {
		t.Fatal("Read should NOT be called for async (InProgress) creates")
	}
}

func TestReadAfterWrite_Create_ErrorPassthrough(t *testing.T) {
	inner := &mockProvisioner{
		createErr: fmt.Errorf("failed to create"),
	}

	w := &readAfterWrite{inner: inner}
	_, err := w.Create(context.Background(), &resource.CreateRequest{})

	if err == nil {
		t.Fatal("expected error from Create")
	}
	if inner.readCalled {
		t.Fatal("Read should NOT be called when Create fails")
	}
}

func TestReadAfterWrite_Create_ReadFailureFallback(t *testing.T) {
	originalProps := json.RawMessage(`{"Id":"ocid1.volume.oc1..abc"}`)
	inner := &mockProvisioner{
		createResult: &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				OperationStatus:    resource.OperationStatusSuccess,
				NativeID:           "ocid1.volume.oc1..abc",
				ResourceProperties: originalProps,
			},
		},
		readErr: fmt.Errorf("read failed"),
	}

	w := &readAfterWrite{inner: inner}
	result, err := w.Create(context.Background(), &resource.CreateRequest{
		ResourceType: "OCI::Core::Volume",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// When Read fails, the original properties from Create should be preserved
	got := string(result.ProgressResult.ResourceProperties)
	want := string(originalProps)
	if got != want {
		t.Errorf("properties = %s, want %s (original from Create)", got, want)
	}
}

func TestReadAfterWrite_Create_ReadNotFoundFallback(t *testing.T) {
	originalProps := json.RawMessage(`{"Id":"ocid1.volume.oc1..abc"}`)
	inner := &mockProvisioner{
		createResult: &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				OperationStatus:    resource.OperationStatusSuccess,
				NativeID:           "ocid1.volume.oc1..abc",
				ResourceProperties: originalProps,
			},
		},
		readResult: &resource.ReadResult{
			ErrorCode: resource.OperationErrorCodeNotFound,
		},
	}

	w := &readAfterWrite{inner: inner}
	result, err := w.Create(context.Background(), &resource.CreateRequest{
		ResourceType: "OCI::Core::Volume",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// When Read returns NotFound, original properties should be preserved
	got := string(result.ProgressResult.ResourceProperties)
	want := string(originalProps)
	if got != want {
		t.Errorf("properties = %s, want %s (original from Create)", got, want)
	}
}

func TestReadAfterWrite_Update_SyncSuccess(t *testing.T) {
	inner := &mockProvisioner{
		updateResult: &resource.UpdateResult{
			ProgressResult: &resource.ProgressResult{
				OperationStatus:    resource.OperationStatusSuccess,
				NativeID:           "ocid1.volume.oc1..abc",
				ResourceProperties: json.RawMessage(`{"Id":"ocid1.volume.oc1..abc"}`),
			},
		},
		readResult: &resource.ReadResult{
			Properties: `{"Id":"ocid1.volume.oc1..abc","CompartmentId":"ocid1.compartment.oc1..xyz","SizeInGBs":100}`,
		},
	}

	w := &readAfterWrite{inner: inner}
	result, err := w.Update(context.Background(), &resource.UpdateRequest{
		ResourceType: "OCI::Core::Volume",
		NativeID:     "ocid1.volume.oc1..abc",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !inner.readCalled {
		t.Fatal("expected Read to be called after sync Update success")
	}

	got := string(result.ProgressResult.ResourceProperties)
	want := `{"Id":"ocid1.volume.oc1..abc","CompartmentId":"ocid1.compartment.oc1..xyz","SizeInGBs":100}`
	if got != want {
		t.Errorf("properties = %s, want %s", got, want)
	}
}

func TestReadAfterWrite_Update_ErrorPassthrough(t *testing.T) {
	inner := &mockProvisioner{
		updateErr: fmt.Errorf("failed to update"),
	}

	w := &readAfterWrite{inner: inner}
	_, err := w.Update(context.Background(), &resource.UpdateRequest{})

	if err == nil {
		t.Fatal("expected error from Update")
	}
	if inner.readCalled {
		t.Fatal("Read should NOT be called when Update fails")
	}
}
