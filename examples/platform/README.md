# Platform

Every OCI resource type the plugin supports, wired into a coherent 3-tier platform. Doubles as a schema smoke test (`formae eval`) and a breadth demo.

## What You Get

- IAM Policy (read-only audit)
- VCN + DHCP Options + Internet Gateway + NAT Gateway + Service Gateway
- Two Route Tables (public and private)
- Security List + 3 Network Security Groups (web/app/db) + NSG Security Rule
- Public Subnet + Private Subnet
- Block Volume + Bastion Instance
- OKE Cluster + NodePool + VirtualNodePool
- Object Storage Bucket

17 resource types, 22 instances.

## Prerequisites

1. OCI credentials at `~/.oci/config`
2. SSH public key
3. A compartment OCID you can create resources in

## Configuration

Set your SSH key and compartment:
```bash
export SSH_PUBLIC_KEY=$(cat ~/.ssh/id_ed25519.pub)
export OCI_COMPARTMENT_ID=ocid1.compartment.oc1..your-compartment-ocid
```

Edit `vars.pkl` to customize:
```pkl
projectName = "oci-showcase"
ociRegion = "us-chicago-1"
vcnCidr = "10.0.0.0/16"
k8sVersion = "v1.34.1"
```

## Eval (schema smoke test)

```bash
formae eval main.pkl
```

This compiles all PKL and resolves every `.res` reference — valuable even without applying.

## Deploy

```bash
formae apply --mode reconcile main.pkl
formae status command --watch --output-layout detailed
```

## Tear Down

```bash
formae destroy --query 'stack:oci-showcase'
```

Destroy may require multiple runs — NetworkSecurityGroupSecurityRule resources sometimes fail to delete on the first attempt.

## Architecture

```
IAM
└── Policy (read-only audit)

VCN (10.0.0.0/16)
├── DHCP Options (VCN DNS)
├── Internet Gateway
├── NAT Gateway
├── Service Gateway (OCI services)
├── Route Table (public)  → Internet Gateway
├── Route Table (private) → NAT + Service Gateway
├── Security List
├── Network Security Group (web)
├── Network Security Group (app)
├── Network Security Group (db)
│   └── NSG Security Rule
├── Public Subnet (10.0.1.0/24)
│   ├── Bastion Instance (Oracle Linux 8)
│   │   └── Block Volume (50 GB)
│   └── OKE Cluster (public endpoint)
│       └── Service Load Balancer targets
└── Private Subnet (10.0.2.0/24)
    ├── OKE Node Pool (managed)
    └── OKE Virtual Node Pool (serverless)

Object Storage
└── Bucket (Standard, private)
```

## File Index

| File | Resources |
|---|---|
| `main.pkl` | Composes everything |
| `vars.pkl` | Stack, Target, Prop-based inputs |
| `infrastructure/identity.pkl` | Policy |
| `infrastructure/network.pkl` | VCN, DhcpOptions, gateways, route tables, security list |
| `infrastructure/security.pkl` | 3 NSGs + NSG rule |
| `infrastructure/subnets.pkl` | Public + private subnet |
| `infrastructure/compute.pkl` | Volume + Instance |
| `infrastructure/containers.pkl` | OKE Cluster + NodePool + VirtualNodePool |
| `infrastructure/storage.pkl` | Bucket |

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `Unable to parse OCID as any format` | Set `OCI_COMPARTMENT_ID` env var or update `vars.pkl` default |
| `Invalid format for ssh public key` | Ensure `SSH_PUBLIC_KEY` is set and contains a full key line |
| Service limit exceeded | Check OCI console for quota limits in your compartment |
| OKE cluster creation slow | Cluster + node pools can take 10-15 minutes to provision |

## Simpler Starting Point

If this is too much, see `examples/dev-environment/` for a 7-resource intro.
