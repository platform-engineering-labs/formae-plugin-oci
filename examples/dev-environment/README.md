# Dev Environment

SSH-accessible Oracle Linux VM with an object storage bucket for scratch data.

## What You Get

- VCN + Public Subnet
- Internet Gateway + Route Table
- Security List (SSH + HTTP allowed)
- VM.Standard.E4.Flex instance (1 OCPU, 6 GB RAM, public IP)
- Object Storage bucket (private, Standard tier)

## Prerequisites

1. OCI credentials at `~/.oci/config`
2. SSH public key

## Configuration

Set your SSH key:
```bash
export SSH_PUBLIC_KEY=$(cat ~/.ssh/id_ed25519.pub)
```

Edit `vars.pkl` to customize:
```pkl
projectName = "dev-environment"
ociRegion = "us-chicago-1"
vcnCidr = "10.0.0.0/16"
publicSubnetCidr = "10.0.1.0/24"
```

## Deploy

```bash
formae apply --mode reconcile main.pkl
formae status command --watch --output-layout detailed
```

## Connect

```bash
ssh opc@<PUBLIC_IP>
```

## Tear Down

```bash
formae destroy --query 'stack:dev-environment'
```

## Architecture

```
VCN (10.0.0.0/16)
├── Internet Gateway
├── Route Table
│   └── 0.0.0.0/0 → Internet Gateway
├── Security List
│   ├── Ingress: SSH (22)
│   └── Ingress: HTTP (80)
├── Public Subnet (10.0.1.0/24)
│   └── Instance (Oracle Linux 8, public IP)
└── Object Storage Bucket (private)
```

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `Invalid format for ssh public key` | Ensure `SSH_PUBLIC_KEY` is set and contains a full key line |
| SSH timeout | Verify your IP isn't blocked by upstream network policy; check the public IP was assigned |
| `Authorization failed or requested resource not found` | Check `compartmentId` in `vars.pkl` matches a real OCI compartment you can access |

## Security Note

SSH (22) and HTTP (80) are open to 0.0.0.0/0 for convenience. For anything beyond local experimentation, restrict the source CIDRs in the security list to your IP range.
