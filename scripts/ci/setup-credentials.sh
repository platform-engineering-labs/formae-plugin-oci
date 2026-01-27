#!/bin/bash
# © 2025 Platform Engineering Labs Inc.
# SPDX-License-Identifier: FSL-1.1-ALv2
#
# Setup OCI Credentials Hook
# ==========================
# This script verifies that OCI credentials are properly configured
# before running conformance tests.
#
# For local development:
#   - Run `oci setup config` to create ~/.oci/config
#   - Or set OCI_CONFIG_FILE and OCI_CONFIG_PROFILE
#
# For CI (GitHub Actions):
#   - Use instance principal (OCI_CLI_AUTH=instance_principal)
#   - Or create config file from secrets

set -euo pipefail

OCI_CONFIG="${OCI_CONFIG_FILE:-$HOME/.oci/config}"
OCI_PROFILE="${OCI_CONFIG_PROFILE:-DEFAULT}"

echo "Verifying OCI credentials..."
echo ""

# Check for OCI CLI
if ! command -v oci &> /dev/null; then
    echo "ERROR: OCI CLI not found"
    echo ""
    echo "Install with: brew install oci-cli"
    echo "Or: pip install oci-cli"
    exit 1
fi

# Check for instance principal auth (CI scenario)
if [[ "${OCI_CLI_AUTH:-}" == "instance_principal" ]]; then
    echo "Using instance principal authentication"
    echo ""
    echo "Verifying credentials..."
    if ! oci iam region list --auth instance_principal > /dev/null 2>&1; then
        echo "ERROR: Instance principal authentication failed"
        exit 1
    fi
    echo ""
    echo "OCI credentials verified successfully (instance principal)!"
    exit 0
fi

# Check for config file
if [[ ! -f "$OCI_CONFIG" ]]; then
    echo "ERROR: OCI config file not found at $OCI_CONFIG"
    echo ""
    echo "Run 'oci setup config' to create one, or set:"
    echo "  - OCI_CONFIG_FILE"
    echo "  - OCI_CONFIG_PROFILE"
    exit 1
fi

echo "Using OCI config file: $OCI_CONFIG"
echo "Using profile: $OCI_PROFILE"

# Check if profile exists in config
if ! grep -q "^\[$OCI_PROFILE\]" "$OCI_CONFIG" 2>/dev/null; then
    echo "ERROR: Profile '$OCI_PROFILE' not found in $OCI_CONFIG"
    echo ""
    echo "Available profiles:"
    grep "^\[" "$OCI_CONFIG" | tr -d '[]'
    exit 1
fi

# Extract tenancy and region from config for display
TENANCY=$(grep -A 20 "^\[$OCI_PROFILE\]" "$OCI_CONFIG" | grep "^tenancy" | head -1 | cut -d= -f2 | tr -d ' ')
REGION=$(grep -A 20 "^\[$OCI_PROFILE\]" "$OCI_CONFIG" | grep "^region" | head -1 | cut -d= -f2 | tr -d ' ')

echo "  Tenancy: ${TENANCY:0:20}..."
echo "  Region: $REGION"

# Verify credentials work
echo ""
echo "Verifying credentials with OCI CLI..."
if ! oci iam region list --profile "$OCI_PROFILE" > /dev/null 2>&1; then
    echo "ERROR: OCI credentials are invalid or expired"
    echo ""
    echo "Check your API key and config, or run 'oci setup config'"
    exit 1
fi

echo ""
echo "OCI credentials verified successfully!"
