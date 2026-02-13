# OCI Plugin Conformance Test Data

PKL test files for the [formae plugin SDK conformance test runner](https://github.com/platform-engineering-labs/formae/tree/main/pkg/plugin-conformance-tests). Each file declares a stack with dependencies + one target resource.

## Test Structure

- `<resource>.pkl` — base create
- `<resource>-update.pkl` — in-place update (changes `displayName` or `name`)

## Passing (CRUD + Discovery)

| Resource | Files | Notes |
|---|---|---|
| Bucket | `bucket.pkl`, `-update`, `-replace` | Tier 0. Replace tests `name`. |
| Policy | `policy.pkl`, `-update`, `-replace` | Tier 0. Replace tests `name`. |
| VCN | `vcn.pkl`, `-update` | Tier 0. No replace. |
| Volume | `volume.pkl`, `-update`, `-replace` | Tier 0. Replace tests `availabilityDomain`. Async create/delete. |
| NetworkSecurityGroup | `networksecuritygroup.pkl`, `-update` | VCN-child. |
| InternetGateway | `internetgateway.pkl`, `-update` | VCN-child. |
| RouteTable | `routetable.pkl`, `-update` | VCN-child. Empty `routeRules`. |
| SecurityList | `securitylist.pkl`, `-update` | VCN-child. Empty rules. |
| Subnet | `subnet.pkl`, `-update` | VCN-child. |
| DhcpOptions | `dhcpoptions.pkl`, `-update` | VCN-child. Known extraction union-type bug (doesn't affect CRUD). |
| NSGSecurityRule | `networksecuritygroupsecurityrule.pkl` | NSG-child. No update support (delete+recreate). Bug fixed: `List` NativeID format. |

## Blocked by Service Limits (test files ready, not a code issue)

| Resource | Files | Blocker |
|---|---|---|
| NatGateway | `natgateway.pkl`, `-update` | `LimitExceeded`: NAT gateway limit = 0 in us-chicago-1 |
| Cluster | `cluster.pkl`, `-update` | `LimitExceeded`: tenancy cluster limit exceeded |
| NodePool | `nodepool.pkl`, `-update` | Blocked on Cluster |
| VirtualNodePool | `virtualnodepool.pkl`, `-update` | Blocked on Cluster |

## Not Tested

| Resource | Reason |
|---|---|
| ServiceGateway | Needs runtime `serviceId` lookup that varies by region |
| Instance | No usable compute shapes in us-chicago-1 (ARM only, out of capacity) |

## Running Tests

```bash
# CRUD
FORMAE_BINARY=../formae/formae OCI_COMPARTMENT_ID=<ocid> make conformance-test-crud TEST=<resource>

# Discovery
FORMAE_BINARY=../formae/formae OCI_COMPARTMENT_ID=<ocid> make conformance-test-discovery TEST=<resource>
```

Resource names match the PKL filename (e.g. `networksecuritygroup`, `dhcpoptions`, `cluster`).
