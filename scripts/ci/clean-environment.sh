#!/bin/bash
# © 2025 Platform Engineering Labs Inc.
# SPDX-License-Identifier: FSL-1.1-ALv2
#
# Clean OCI Environment Hook
# ==========================
# This script cleans up OCI test resources before AND after conformance tests.
# It is idempotent - safe to run multiple times.
#
# Test resources are identified by the "formae-plugin-sdk-test" prefix in their names.
# Resources are cleaned up in dependency order (children before parents).

set -euo pipefail

OCI_PROFILE="${OCI_CONFIG_PROFILE:-DEFAULT}"
TEST_PREFIX="${TEST_PREFIX:-formae-plugin-sdk-test-}"

# Get compartment ID from environment or fail
COMPARTMENT_ID="${OCI_COMPARTMENT_ID:-}"
if [[ -z "$COMPARTMENT_ID" ]]; then
    echo "ERROR: OCI_COMPARTMENT_ID must be set"
    echo "Set it to the compartment where test resources are created."
    exit 1
fi

echo "=== Cleaning OCI test resources ==="
echo "Compartment: $COMPARTMENT_ID"
echo "Looking for resources with '$TEST_PREFIX' in name..."
echo ""

# Helper to check if OCI CLI is available
if ! command -v oci &> /dev/null; then
    echo "OCI CLI not found. Skipping cleanup."
    exit 0
fi

# Helper function to delete resources by display name prefix
delete_by_prefix() {
    local resource_type="$1"
    local list_cmd="$2"
    local delete_cmd="$3"
    local id_field="${4:-id}"

    echo "Cleaning $resource_type..."

    # List resources and filter by prefix
    local resources
    resources=$($list_cmd --compartment-id "$COMPARTMENT_ID" --profile "$OCI_PROFILE" \
        --query "data[?starts_with(\"display-name\", '$TEST_PREFIX')].$id_field" \
        --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true)

    if [[ -z "$resources" ]]; then
        echo "  No test $resource_type found"
        return
    fi

    for id in $resources; do
        echo "  Deleting: $id"
        $delete_cmd --profile "$OCI_PROFILE" --force 2>/dev/null || true
    done
}

# 1. Delete Policies with test prefix (before compartments)
echo "Cleaning Identity test policies..."
POLICIES=$(oci iam policy list --compartment-id "$COMPARTMENT_ID" --profile "$OCI_PROFILE" \
    --query "data[?starts_with(name, '$TEST_PREFIX')].id" --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true)

for pol in $POLICIES; do
    echo "  Deleting policy: $pol"
    oci iam policy delete --policy-id "$pol" --profile "$OCI_PROFILE" --force 2>/dev/null || true
done

# 2. Delete Block Volumes with test prefix
echo "Cleaning Block Volume test volumes..."
VOLUMES=$(oci bv volume list --compartment-id "$COMPARTMENT_ID" --profile "$OCI_PROFILE" \
    --query "data[?starts_with(\"display-name\", '$TEST_PREFIX')].id" --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true)

for vol in $VOLUMES; do
    echo "  Deleting volume: $vol"
    oci bv volume delete --volume-id "$vol" --profile "$OCI_PROFILE" --force 2>/dev/null || true
done

# 3. Delete Object Storage buckets with test prefix
echo "Cleaning Object Storage test buckets..."
NAMESPACE=$(oci os ns get --profile "$OCI_PROFILE" --query 'data' --raw-output 2>/dev/null || true)
if [[ -n "$NAMESPACE" ]]; then
    BUCKETS=$(oci os bucket list --compartment-id "$COMPARTMENT_ID" --namespace-name "$NAMESPACE" --profile "$OCI_PROFILE" \
        --query "data[?starts_with(name, '$TEST_PREFIX')].name" --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true)

    for bucket in $BUCKETS; do
        echo "  Deleting bucket: $bucket"
        # Delete all objects first
        oci os object bulk-delete --bucket-name "$bucket" --namespace-name "$NAMESPACE" --profile "$OCI_PROFILE" --force 2>/dev/null || true
        # Delete the bucket
        oci os bucket delete --bucket-name "$bucket" --namespace-name "$NAMESPACE" --profile "$OCI_PROFILE" --force 2>/dev/null || true
    done
fi

# 4. Delete Node Pools (before clusters)
echo "Cleaning Container Engine test node pools..."
NODE_POOLS=$(oci ce node-pool list --compartment-id "$COMPARTMENT_ID" --profile "$OCI_PROFILE" \
    --query "data[?starts_with(name, '$TEST_PREFIX')].id" --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true)

for np in $NODE_POOLS; do
    echo "  Deleting node pool: $np"
    oci ce node-pool delete --node-pool-id "$np" --profile "$OCI_PROFILE" --force 2>/dev/null || true
done

# 5. Delete OKE Clusters
echo "Cleaning Container Engine test clusters..."
CLUSTERS=$(oci ce cluster list --compartment-id "$COMPARTMENT_ID" --profile "$OCI_PROFILE" \
    --query "data[?starts_with(name, '$TEST_PREFIX')].id" --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true)

for cluster in $CLUSTERS; do
    echo "  Deleting cluster: $cluster"
    oci ce cluster delete --cluster-id "$cluster" --profile "$OCI_PROFILE" --force 2>/dev/null || true
done

# 6. Delete Subnets (before VCNs)
echo "Cleaning test subnets..."
SUBNETS=$(oci network subnet list --compartment-id "$COMPARTMENT_ID" --profile "$OCI_PROFILE" \
    --query "data[?starts_with(\"display-name\", '$TEST_PREFIX')].id" --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true)

for subnet in $SUBNETS; do
    echo "  Deleting subnet: $subnet"
    oci network subnet delete --subnet-id "$subnet" --profile "$OCI_PROFILE" --force 2>/dev/null || true
done

# 7. Delete Route Tables (before gateways)
echo "Cleaning test route tables..."
ROUTE_TABLES=$(oci network route-table list --compartment-id "$COMPARTMENT_ID" --profile "$OCI_PROFILE" \
    --query "data[?starts_with(\"display-name\", '$TEST_PREFIX')].id" --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true)

for rt in $ROUTE_TABLES; do
    echo "  Deleting route table: $rt"
    oci network route-table delete --rt-id "$rt" --profile "$OCI_PROFILE" --force 2>/dev/null || true
done

# 8. Delete Security Lists
echo "Cleaning test security lists..."
SEC_LISTS=$(oci network security-list list --compartment-id "$COMPARTMENT_ID" --profile "$OCI_PROFILE" \
    --query "data[?starts_with(\"display-name\", '$TEST_PREFIX')].id" --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true)

for sl in $SEC_LISTS; do
    echo "  Deleting security list: $sl"
    oci network security-list delete --security-list-id "$sl" --profile "$OCI_PROFILE" --force 2>/dev/null || true
done

# 9. Delete Network Security Groups
echo "Cleaning test network security groups..."
NSGS=$(oci network nsg list --compartment-id "$COMPARTMENT_ID" --profile "$OCI_PROFILE" \
    --query "data[?starts_with(\"display-name\", '$TEST_PREFIX')].id" --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true)

for nsg in $NSGS; do
    echo "  Deleting NSG: $nsg"
    oci network nsg delete --nsg-id "$nsg" --profile "$OCI_PROFILE" --force 2>/dev/null || true
done

# 10. Delete Internet Gateways
echo "Cleaning test internet gateways..."
IGS=$(oci network internet-gateway list --compartment-id "$COMPARTMENT_ID" --profile "$OCI_PROFILE" \
    --query "data[?starts_with(\"display-name\", '$TEST_PREFIX')].id" --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true)

for ig in $IGS; do
    echo "  Deleting internet gateway: $ig"
    oci network internet-gateway delete --ig-id "$ig" --profile "$OCI_PROFILE" --force 2>/dev/null || true
done

# 11. Delete NAT Gateways
echo "Cleaning test NAT gateways..."
NATS=$(oci network nat-gateway list --compartment-id "$COMPARTMENT_ID" --profile "$OCI_PROFILE" \
    --query "data[?starts_with(\"display-name\", '$TEST_PREFIX')].id" --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true)

for nat in $NATS; do
    echo "  Deleting NAT gateway: $nat"
    oci network nat-gateway delete --nat-gateway-id "$nat" --profile "$OCI_PROFILE" --force 2>/dev/null || true
done

# 12. Delete DHCP Options
echo "Cleaning test DHCP options..."
DHCP_OPTIONS=$(oci network dhcp-options list --compartment-id "$COMPARTMENT_ID" --profile "$OCI_PROFILE" \
    --query "data[?starts_with(\"display-name\", '$TEST_PREFIX')].id" --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true)

for dhcp in $DHCP_OPTIONS; do
    echo "  Deleting DHCP options: $dhcp"
    oci network dhcp-options delete --dhcp-id "$dhcp" --profile "$OCI_PROFILE" --force 2>/dev/null || true
done

# 13. Delete Service Gateways
echo "Cleaning test service gateways..."
SGS=$(oci network service-gateway list --compartment-id "$COMPARTMENT_ID" --profile "$OCI_PROFILE" \
    --query "data[?starts_with(\"display-name\", '$TEST_PREFIX')].id" --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true)

for sg in $SGS; do
    echo "  Deleting service gateway: $sg"
    oci network service-gateway delete --service-gateway-id "$sg" --profile "$OCI_PROFILE" --force 2>/dev/null || true
done

# 14. Delete VCNs (after all network dependencies)
echo "Cleaning test VCNs..."
VCNS=$(oci network vcn list --compartment-id "$COMPARTMENT_ID" --profile "$OCI_PROFILE" \
    --query "data[?starts_with(\"display-name\", '$TEST_PREFIX')].id" --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true)

for vcn in $VCNS; do
    echo "  Deleting VCN: $vcn"
    oci network vcn delete --vcn-id "$vcn" --profile "$OCI_PROFILE" --force 2>/dev/null || true
done

# 15. Delete Compartments with test prefix (be careful!)
echo "Cleaning test compartments..."
COMPARTMENTS=$(oci iam compartment list --compartment-id "$COMPARTMENT_ID" --profile "$OCI_PROFILE" \
    --query "data[?starts_with(name, '$TEST_PREFIX') && \"lifecycle-state\"=='ACTIVE'].id" --output json 2>/dev/null | jq -r '.[]' 2>/dev/null || true)

for comp in $COMPARTMENTS; do
    echo "  Deleting compartment: $comp"
    oci iam compartment delete --compartment-id "$comp" --profile "$OCI_PROFILE" --force 2>/dev/null || true
done

echo ""
echo "=== Cleanup complete ==="
