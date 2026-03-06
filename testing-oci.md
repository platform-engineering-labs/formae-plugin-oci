# OCI Plugin Testing Runbook

## Repos & Build

```
# Plugin repo
~/Workspace/github.com/platform-engineering-labs/formae-plugin-oci

# Formae core (binary lives here)
~/Workspace/github.com/platform-engineering-labs/formae
```

### Build & Install

```bash
make build          # compile plugin
make install-dev    # install plugin locally (~/.pel/formae/plugins/)
make test           # run unit tests
make verify-schema  # validate PKL schemas
```

### Formae Binary

```bash
cd ../formae && ./formae <command>
```

### Log Location

```
~/.pel/formae/log/client.log   # client-side logs
~/.pel/formae/log/formae.log   # server/agent logs (plugin operations, discovery)
```

### Agent Management

```bash
# Discovery happens automatically when the agent is running with a discoverable target.
# After updating the plugin binary, restart the agent to pick up changes:
./formae agent stop && ./formae agent start
```

---

## MCP Skills Reference (Preferred Testing Workflow)

Use MCP skills via Claude Code for all testing. These are the primary tools:

| Step | MCP Skill | MCP Tool | Notes |
|------|-----------|----------|-------|
| Check targets | `/formae-targets` | `list_targets` | Verify OCI target is connected |
| Overview stats | `/formae-discover` | `get_agent_stats` | Start here — see resource counts |
| List resources | `/formae-resources` | `list_resources` | Always add type filter |
| Force discovery | `/formae-discover` | `force_discover` | After plugin update + agent restart |
| Extract to PKL | `/formae-import` | `extract_resources` | Never change resource labels |
| Apply (simulate!) | `/formae-apply` | `apply_forma` | Always `simulate:true` first |
| Apply (real) | `/formae-apply` | `apply_forma` | Then `simulate:false` |
| Check status | `/formae-status` | `get_command_status` | Poll until complete |
| Destroy (simulate!) | `/formae-destroy` | `destroy_forma` | Always `simulate:true` first |
| Destroy (real) | `/formae-destroy` | `destroy_forma` | Then `simulate:false` |
| Check drift | `/formae-drift` | `list_drift` | After apply, verify no drift |
| List stacks | `/formae-stacks` | `list_stacks` | |
| List plugins | `/formae-plugins` | `list_plugins` | Verify plugin version |

### Key MCP Rules

- **Always simulate before apply/destroy** — never skip the simulate step
- **Start with `get_agent_stats`** — get the overview, then drill down with targeted queries
- **Never `list_resources` with just `managed:false`** — always add a type filter
- **Never change resource labels** during import/extraction

---

## Formae CLI Reference (Fallback)

```bash
# Apply a stack (create/update resources)
./formae apply --mode reconcile <file.pkl> --yes

# Destroy managed resources
./formae destroy <file.pkl> --yes

# Check operation status by command ID
./formae status command --query 'id:<command-id>'

# Check in-progress operations
./formae status command --query 'state:inprogress'

# Evaluate PKL (dry run / validate)
./formae eval <file.pkl>

# Extract discovered resources to PKL
./formae extract <output.pkl> --query 'type:<ResourceType>'

# List inventory (query by type, use --max-results for large result sets)
./formae inventory resources --query 'type:<ResourceType>' --max-results 50
./formae inventory targets --max-results 20

# List/query stacks and resources
./formae inventory resources --query 'type:OCI::Identity::Policy' --max-results 50
./formae inventory resources --query 'managed:false' --max-results 50
```

---

## Testing Workflow Per Resource

For each resource, follow this end-to-end cycle. Prefer MCP skills; CLI commands shown as fallback.

### 1. Write a test forma (or extract from discovery)

Write a minimal PKL file with a stack, target, and one resource. Use `/formae-import`
(or `formae extract`) if you want to test adoption of an existing resource.

### 2. Evaluate PKL (dry run)

```bash
./formae eval <test-file.pkl>
```

### 3. Apply to create the resource

Use `/formae-apply` with `simulate:true`, then `simulate:false`. Check status with `/formae-status`.

```bash
# CLI fallback:
./formae apply --mode reconcile <test-file.pkl> --yes
./formae status command --query 'id:<command-id>'
```

### 4. Re-apply (idempotency check)

Apply again — should show "No changes needed".

### 5. Destroy the managed resource

Use `/formae-destroy` with `simulate:true`, then `simulate:false`. Check status with `/formae-status`.

```bash
# CLI fallback:
./formae destroy <test-file.pkl> --yes
./formae status command --query 'id:<command-id>'
```

### 6. Verify resource is gone

```bash
# OCI CLI - should 404
oci <service> <resource> get --<id-param> <id>
```

### Discovery testing (optional additional flow)

Use `/formae-discover` to force discovery, `/formae-resources` to list, `/formae-import` to extract.

```bash
# CLI fallback:
./formae inventory resources --query 'type:<ResourceType>' --max-results 50
./formae extract ./extracted.pkl --query 'type:<ResourceType>'
./formae eval extracted.pkl
```

---

## Conformance Test Results (2026-02-12)

Ran plugin SDK conformance tests (CRUD + Discovery) for basic resources that only need a CompartmentId as parent.

### Policy (CRUD: 18/18, Discovery: 9/9)

| Test | Steps | Result | Duration |
|---|---|---|---|
| CRUD | 1-18 (create, sync, update description, replace name, destroy) | **PASS** | ~30s |
| Discovery | 1-9 | **PASS** | ~25s |

Bugs fixed during testing:
- Policy statements must scope to the policy's own compartment or below. "in tenancy" fails if the policy lives in a child compartment. Used `in compartment id \(testCompartmentId)`.

### VCN (CRUD: 12/12, Discovery: 9/9)

| Test | Steps | Result | Duration |
|---|---|---|---|
| CRUD | 1-9 + 10-12 (create, sync, update displayName, destroy — no replace) | **PASS** | ~15s |
| Discovery | 1-9 | **PASS** | ~15s |

No replace test — VCN has no meaningful createOnly fields besides compartmentId.

### Volume (CRUD: 18/18, Discovery: 9/9)

| Test | Steps | Result | Duration |
|---|---|---|---|
| CRUD | 1-18 (create, sync, update displayName, replace availabilityDomain, destroy) | **PASS** | ~96s |
| Discovery | 1-9 | **PASS** | ~41s |

Bug fixed during testing:
- **Volume provisioner async create**: The `Create` method returned `OperationStatusSuccess` immediately, but volume creation is async (goes through PROVISIONING state). The conformance test's update step failed with "Incorrect State" because the volume wasn't AVAILABLE yet. Fixed by returning `OperationStatusInProgress` from Create/Delete and implementing lifecycle state polling in `Status` (matching the Instance provisioner pattern).

### VCN-Dependent Resources — Conformance Tests (2026-02-12)

Ran plugin SDK conformance tests (CRUD + Discovery) for VCN-dependent resources. Each test includes Compartment + VCN as dependencies. No replace tests — all VCN-children only have compartmentId/vcnId as createOnly fields.

| Resource | CRUD Steps | CRUD Result | CRUD Duration | Discovery Steps | Discovery Result | Discovery Duration |
|---|---|---|---|---|---|---|
| NetworkSecurityGroup | 1-12, 16-18 | **PASS** | ~32s | 1-7 | **PASS** | ~32s |
| InternetGateway | 1-12, 16-18 | **PASS** | ~21s | 1-7 | **PASS** | ~19s |
| NatGateway | — | **SKIP** | — | — | **SKIP** | — |
| RouteTable | 1-12, 16-18 | **PASS** | ~20s | 1-7 | **PASS** | ~32s |
| SecurityList | 1-12, 16-18 | **PASS** | ~32s | 1-7 | **PASS** | ~20s |
| Subnet | 1-12, 16-18 | **PASS** | ~20s | 1-7 | **PASS** | ~20s |
| DhcpOptions | 1-12, 16-18 | **PASS** | ~20s | 1-7 | **PASS** | ~89s |

- **NatGateway skipped** — OCI returns `LimitExceeded: NAT gateway limit per VCN reached` on create in us-chicago-1. The tenancy has a 0 NAT gateway service limit in this region. Not a plugin bug — provisioner code is identical pattern to InternetGateway.
- **ServiceGateway skipped** — needs runtime `serviceId` lookup (`oci network service list`) that varies by region.
- **DhcpOptions discovery slow** (~89s) — likely due to the known union-type extraction issue causing extra retry cycles, but passes.

### NetworkSecurityGroupSecurityRule — Conformance Tests (2026-02-12)

Tier 2 resource: Compartment → VCN → NSG → NSGSecurityRule. Update not supported (delete+recreate only).

| Test | Steps | Result | Duration |
|---|---|---|---|
| CRUD | 1-9, 16-18 (create, sync, destroy — no update/replace) | **PASS** | ~18s |
| Discovery | 1-7 | **PASS** | ~20s |

Bug fixed during testing:
- **NSG Security Rule List returned bare rule IDs**: The `List` method returned just `ruleId` but `Read` expects composite `nsgId/ruleId` format. Discovery failed with `invalid NativeID format`. Fixed `List` to return `fmt.Sprintf("%s/%s", nsgId, *rule.Id)`.

---

## Lifeline E2E Results (2026-02-09)

Full multi-resource lifecycle test using `examples/lifeline/` (12 resources: VCN, IGW, RouteTable, 2 Subnets, 2 NSGs, 4 NSG Rules, plus cross-cutting patch targets).

| Step | Result | Details |
|---|---|---|
| `make build && make test` | Pass | All unit tests pass (including 7 ReadAfterWrite tests) |
| `make install` | Pass | Plugin installed to v0.1.0 |
| Eval (dry run) | Pass | PKL validates cleanly |
| Apply (simulate) | Pass | 12 resources planned |
| Apply (real) | **12/12 success, 8s** | All resources created with complete properties |
| Re-apply (idempotency) | **No changes needed** | ReadAfterWrite ensures complete properties on first apply |
| Extract | Pass | Extracted to PKL with correct labels and properties |
| Cross-cutting patch (simulate) | Pass | 3 updates planned (IGW + 2 NSGs) |
| Cross-cutting patch (real) | **3/3 success** | FreeformTag added to all 3 resources |
| Destroy (simulate) | Pass | 11 deletes planned |
| Destroy (real) | **11/11 success, 18s** | All resources cleaned up |

## P0 Resource Progress

| Resource Type | Impl | E2E Tested | Notes |
|---|---|---|---|
| `OCI::Identity::Compartment` | Yes | - | Pre-existing |
| `OCI::Identity::Policy` | Yes | Yes | Conformance CRUD (18/18) + Discovery (9/9) pass |
| `OCI::Core::VCN` | Yes | Yes | Conformance CRUD (12/12) + Discovery (9/9) pass; Lifeline E2E pass |
| `OCI::Core::Subnet` | Yes | Yes | Conformance CRUD (12/12) + Discovery (7/7) pass; Lifeline E2E pass |
| `OCI::Core::InternetGateway` | Yes | Yes | Conformance CRUD (12/12) + Discovery (7/7) pass; Lifeline E2E pass |
| `OCI::Core::NatGateway` | Yes | - | Conformance skipped: LimitExceeded in us-chicago-1 (service limit 0) |
| `OCI::Core::ServiceGateway` | Yes | - | Conformance skipped: needs runtime serviceId lookup |
| `OCI::Core::RouteTable` | Yes | Yes | Conformance CRUD (12/12) + Discovery (7/7) pass; Lifeline E2E pass |
| `OCI::Core::SecurityList` | Yes | Yes | Conformance CRUD (12/12) + Discovery (7/7) pass |
| `OCI::Core::NetworkSecurityGroup` | Yes | Yes | Conformance CRUD (12/12) + Discovery (7/7) pass; Lifeline E2E pass |
| `OCI::Core::NetworkSecurityGroupSecurityRule` | Yes | Yes | Conformance CRUD (9/9) + Discovery (7/7) pass; List NativeID bug fixed |
| `OCI::Core::DhcpOptions` | Yes | Yes | Conformance CRUD (12/12) + Discovery (7/7) pass |
| `OCI::Core::Volume` | Yes | Yes | Conformance CRUD (18/18) + Discovery (9/9) pass; async create/delete fix applied |
| `OCI::Core::Instance` | Yes | eval only | us-chicago-1 has no usable compute shapes (ARM only, all out of capacity or 0-0 memory ratio). Plugin logic verified: async create/status/delete paths, retry on OutOfHostCapacity. Needs a region with x86 Flex capacity for full E2E. |
| `OCI::ContainerEngine::Cluster` | Yes | - | Conformance skipped: LimitExceeded — tenancy cluster limit reached in us-chicago-1 |
| `OCI::ContainerEngine::NodePool` | Yes | - | Conformance skipped: blocked on Cluster limit |
| `OCI::ContainerEngine::VirtualNodePool` | Yes | - | Conformance skipped: blocked on Cluster limit |

## Known Bugs / Investigation Items

1. **DhcpOptions extraction union type issue**: When extracting `DhcpOptions`, the
   `options` field uses `Listing<DhcpDnsOption|DhcpSearchDomainOption>`. The extraction
   engine doesn't discriminate between union types — it uses `DhcpDnsOption` for all
   entries, including `SearchDomain` options. This drops `searchDomainNames` and uses the
   wrong class. Fixed nested class fields to be optional to avoid extraction crashes, but
   the wrong class is still used. This is a formae core extraction limitation.

2. **Agent restart required after plugin update**: After `make build && make install-dev`
   (or manual binary copy), the formae agent must be stopped and restarted to pick up the
   new plugin binary. Discovery will not use the updated code until restart.

---

## OCI CLI Commands Per P0 Resource

### OCI::Identity::Policy

```bash
# Create
oci iam policy create --compartment-id $C --name test-policy \
  --description "test" \
  --statements '["Allow group Administrators to manage all-resources in compartment test"]'

# Get
oci iam policy get --policy-id <id>

# List
oci iam policy list --compartment-id $C

# Delete
oci iam policy delete --policy-id <id> --force
```

### OCI::Core::DhcpOptions

```bash
# Create
oci network dhcp-options create --compartment-id $C --vcn-id $VCN \
  --options '[{"type":"DomainNameServer","serverType":"VcnLocalPlusInternet"}]'

# Get
oci network dhcp-options get --dhcp-id <id>

# List
oci network dhcp-options list --compartment-id $C --vcn-id $VCN

# Delete
oci network dhcp-options delete --dhcp-id <id> --force
```

### OCI::Core::Volume

```bash
# Create
oci bv volume create --compartment-id $C --availability-domain $AD \
  --display-name test-volume --size-in-gbs 50

# Get
oci bv volume get --volume-id <id>

# List
oci bv volume list --compartment-id $C

# Delete
oci bv volume delete --volume-id <id> --force
```

### OCI::Core::Instance

```bash
# Create
oci compute instance launch --compartment-id $C --availability-domain $AD \
  --shape VM.Standard.E4.Flex \
  --shape-config '{"ocpus":1,"memoryInGBs":16}' \
  --image-id <image-ocid> --subnet-id $SUBNET \
  --display-name test-instance

# Get
oci compute instance get --instance-id <id>

# List
oci compute instance list --compartment-id $C

# Terminate
oci compute instance terminate --instance-id <id> --preserve-boot-volume false --force
```

### OCI::Identity::Compartment

```bash
oci iam compartment create --compartment-id $TENANCY --name test --description "test"
oci iam compartment get --compartment-id <id>
oci iam compartment list --compartment-id $TENANCY
oci iam compartment delete --compartment-id <id> --force
```

### OCI::Core::VCN

```bash
oci network vcn create --compartment-id $C --cidr-block 10.0.0.0/16 --display-name test-vcn
oci network vcn get --vcn-id <id>
oci network vcn list --compartment-id $C
oci network vcn delete --vcn-id <id> --force
```

### OCI::Core::Subnet

```bash
oci network subnet create --compartment-id $C --vcn-id $VCN --cidr-block 10.0.0.0/24
oci network subnet get --subnet-id <id>
oci network subnet list --compartment-id $C --vcn-id $VCN
oci network subnet delete --subnet-id <id> --force
```

### OCI::Core::InternetGateway

```bash
oci network internet-gateway create --compartment-id $C --vcn-id $VCN --is-enabled true
oci network internet-gateway get --ig-id <id>
oci network internet-gateway list --compartment-id $C --vcn-id $VCN
oci network internet-gateway delete --ig-id <id> --force
```

### OCI::Core::NatGateway

```bash
oci network nat-gateway create --compartment-id $C --vcn-id $VCN
oci network nat-gateway get --nat-gateway-id <id>
oci network nat-gateway list --compartment-id $C --vcn-id $VCN
oci network nat-gateway delete --nat-gateway-id <id> --force
```

### OCI::Core::ServiceGateway

```bash
oci network service-gateway create --compartment-id $C --vcn-id $VCN \
  --services '[{"serviceId":"<all-services-id>"}]'
oci network service-gateway get --service-gateway-id <id>
oci network service-gateway list --compartment-id $C --vcn-id $VCN
oci network service-gateway delete --service-gateway-id <id> --force
```

### OCI::Core::RouteTable

```bash
oci network route-table create --compartment-id $C --vcn-id $VCN --route-rules '[]'
oci network route-table get --rt-id <id>
oci network route-table list --compartment-id $C --vcn-id $VCN
oci network route-table delete --rt-id <id> --force
```

### OCI::Core::SecurityList

```bash
oci network security-list create --compartment-id $C --vcn-id $VCN \
  --ingress-security-rules '[]' --egress-security-rules '[]'
oci network security-list get --security-list-id <id>
oci network security-list list --compartment-id $C --vcn-id $VCN
oci network security-list delete --security-list-id <id> --force
```

### OCI::Core::NetworkSecurityGroup

```bash
oci network nsg create --compartment-id $C --vcn-id $VCN --display-name test-nsg
oci network nsg get --nsg-id <id>
oci network nsg list --compartment-id $C --vcn-id $VCN
oci network nsg delete --nsg-id <id> --force
```
