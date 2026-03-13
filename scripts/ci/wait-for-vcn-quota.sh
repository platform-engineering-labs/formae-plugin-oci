#!/bin/bash
# © 2025 Platform Engineering Labs Inc.
# SPDX-License-Identifier: FSL-1.1-ALv2
#
# Wait for VCN Quota Availability
# ================================
# Polls OCI until there are zero active VCNs in the test compartment.
# Each conformance test may need up to 2 VCNs (dependency + test resource),
# so we wait for a completely clean slate before starting.
# Used between sequential conformance test matrix jobs to avoid
# LimitExceeded failures from eventual consistency delays or
# leaked VCNs from prior failed runs.

set -euo pipefail

OCI_PROFILE="${OCI_CONFIG_PROFILE:-DEFAULT}"
POLL_INTERVAL=5
MAX_WAIT=300

COMPARTMENT_ID="${OCI_COMPARTMENT_ID:-}"
if [[ -z "$COMPARTMENT_ID" ]]; then
    echo "ERROR: OCI_COMPARTMENT_ID must be set"
    exit 1
fi

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
echo "  Warning: still $COUNT active VCN(s) after ${ELAPSED}s, proceeding anyway."
