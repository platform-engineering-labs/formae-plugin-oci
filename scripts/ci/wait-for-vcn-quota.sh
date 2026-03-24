#!/bin/bash
# © 2025 Platform Engineering Labs Inc.
# SPDX-License-Identifier: FSL-1.1-ALv2
#
# Wait for VCN Quota Availability
# ================================
# Actively cleans up any lingering test VCNs and their dependencies,
# then polls until the VCN count reaches zero. This ensures the next
# conformance test has a clean slate even if the previous test's async
# cleanup hasn't completed.

set -euo pipefail

OCI_PROFILE="${OCI_CONFIG_PROFILE:-DEFAULT}"
TEST_PREFIX="${TEST_PREFIX:-formae-plugin-sdk-test-}"
POLL_INTERVAL=10
MAX_WAIT=600  # 10 minutes — VCN deletion can be slow with dependencies

COMPARTMENT_ID="${OCI_COMPARTMENT_ID:-}"
if [[ -z "$COMPARTMENT_ID" ]]; then
    echo "ERROR: OCI_COMPARTMENT_ID must be set"
    exit 1
fi

# Actively clean up any test VCNs that are still around.
# Deletes dependencies (subnets, gateways, etc.) before the VCN itself.
cleanup_test_vcns() {
    echo "Checking for lingering test VCNs..."

    VCNS=$(oci network vcn list --compartment-id "$COMPARTMENT_ID" --profile "$OCI_PROFILE" \
        --query "data[?\"lifecycle-state\"=='AVAILABLE'].id" \
        --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true)

    if [[ -z "$VCNS" ]]; then
        return
    fi

    for vcn_id in $VCNS; do
        echo "  Cleaning up VCN: $vcn_id"

        # Delete subnets
        for subnet in $(oci network subnet list --compartment-id "$COMPARTMENT_ID" --vcn-id "$vcn_id" --profile "$OCI_PROFILE" \
            --query "data[?\"lifecycle-state\"=='AVAILABLE'].id" --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true); do
            echo "    Deleting subnet: $subnet"
            oci network subnet delete --subnet-id "$subnet" --profile "$OCI_PROFILE" --force 2>/dev/null || true
        done

        # Delete security lists (non-default only — default can't be deleted before VCN)
        for sl in $(oci network security-list list --compartment-id "$COMPARTMENT_ID" --vcn-id "$vcn_id" --profile "$OCI_PROFILE" \
            --query "data[?starts_with(\"display-name\", '$TEST_PREFIX')].id" --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true); do
            echo "    Deleting security list: $sl"
            oci network security-list delete --security-list-id "$sl" --profile "$OCI_PROFILE" --force 2>/dev/null || true
        done

        # Delete route tables (non-default)
        for rt in $(oci network route-table list --compartment-id "$COMPARTMENT_ID" --vcn-id "$vcn_id" --profile "$OCI_PROFILE" \
            --query "data[?starts_with(\"display-name\", '$TEST_PREFIX')].id" --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true); do
            echo "    Deleting route table: $rt"
            oci network route-table delete --rt-id "$rt" --profile "$OCI_PROFILE" --force 2>/dev/null || true
        done

        # Delete internet gateways
        for ig in $(oci network internet-gateway list --compartment-id "$COMPARTMENT_ID" --vcn-id "$vcn_id" --profile "$OCI_PROFILE" \
            --query "data[?\"lifecycle-state\"=='AVAILABLE'].id" --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true); do
            echo "    Deleting internet gateway: $ig"
            oci network internet-gateway delete --ig-id "$ig" --profile "$OCI_PROFILE" --force 2>/dev/null || true
        done

        # Delete DHCP options (non-default)
        for dhcp in $(oci network dhcp-options list --compartment-id "$COMPARTMENT_ID" --vcn-id "$vcn_id" --profile "$OCI_PROFILE" \
            --query "data[?starts_with(\"display-name\", '$TEST_PREFIX')].id" --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true); do
            echo "    Deleting DHCP options: $dhcp"
            oci network dhcp-options delete --dhcp-id "$dhcp" --profile "$OCI_PROFILE" --force 2>/dev/null || true
        done

        # Delete NSGs
        for nsg in $(oci network nsg list --compartment-id "$COMPARTMENT_ID" --vcn-id "$vcn_id" --profile "$OCI_PROFILE" \
            --query "data[?\"lifecycle-state\"=='AVAILABLE'].id" --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true); do
            echo "    Deleting NSG: $nsg"
            oci network nsg delete --nsg-id "$nsg" --profile "$OCI_PROFILE" --force 2>/dev/null || true
        done

        # Now delete the VCN itself
        echo "    Deleting VCN: $vcn_id"
        oci network vcn delete --vcn-id "$vcn_id" --profile "$OCI_PROFILE" --force 2>/dev/null || true
    done
}

# Run active cleanup first
cleanup_test_vcns

# Then wait for all VCNs to be fully terminated
echo "Waiting for zero active VCNs in compartment..."
START_TIME=$(date +%s)

for i in $(seq 1 $((MAX_WAIT / POLL_INTERVAL))); do
    COUNT=$(oci network vcn list --compartment-id "$COMPARTMENT_ID" --profile "$OCI_PROFILE" \
        --query "length(data[?\"lifecycle-state\"!='TERMINATED'])" \
        --raw-output 2>/dev/null || echo "error")

    if [[ "$COUNT" == "error" ]]; then
        echo "  Warning: failed to query VCNs, retrying..."
        sleep "$POLL_INTERVAL"
        continue
    fi

    if [[ "$COUNT" -eq 0 ]]; then
        ELAPSED=$(( $(date +%s) - START_TIME ))
        echo "  Active VCNs: 0 — clean slate after ${ELAPSED}s"
        exit 0
    fi

    echo "  Active VCNs: $COUNT — waiting..."
    sleep "$POLL_INTERVAL"
done

ELAPSED=$(( $(date +%s) - START_TIME ))
echo "ERROR: still $COUNT active VCN(s) after ${ELAPSED}s — cannot proceed"
exit 1
