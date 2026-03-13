// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

//go:build integration

package provisioner_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/client"
	"github.com/platform-engineering-labs/formae-plugin-oci/pkg/config"
	"github.com/stretchr/testify/require"
)

// getTestCompartmentID reads OCI_COMPARTMENT_ID from the environment.
// Skips the test if the variable is not set.
func getTestCompartmentID(t *testing.T) string {
	t.Helper()
	id := os.Getenv("OCI_COMPARTMENT_ID")
	if id == "" {
		t.Skip("OCI_COMPARTMENT_ID environment variable not set")
	}
	return id
}

// getTestProfile returns the OCI config profile from OCI_CONFIG_PROFILE (default: "DEFAULT").
func getTestProfile() string {
	if p := os.Getenv("OCI_CONFIG_PROFILE"); p != "" {
		return p
	}
	return ""
}

// newTestClients creates OCI SDK clients using the profile from OCI_CONFIG_PROFILE.
func newTestClients(t *testing.T) *client.Clients {
	t.Helper()
	ctx := context.Background()
	cfg := &config.Config{Profile: getTestProfile()}
	clients, err := client.NewClients(ctx, cfg)
	require.NoError(t, err, "failed to create OCI clients")
	return clients
}

// newTestTargetConfig creates a JSON target config with the configured profile.
func newTestTargetConfig() json.RawMessage {
	profile := getTestProfile()
	if profile != "" {
		b, _ := json.Marshal(map[string]string{"Profile": profile})
		return b
	}
	return json.RawMessage(`{}`)
}

// mustMarshalJSON marshals v to json.RawMessage and fails the test on error.
func mustMarshalJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err, "failed to marshal JSON")
	return b
}
