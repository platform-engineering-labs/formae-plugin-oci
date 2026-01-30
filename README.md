# OCI Plugin for Formae

[![CI](https://github.com/platform-engineering-labs/formae-plugin-oci/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/platform-engineering-labs/formae-plugin-oci/actions/workflows/ci.yml)
[![Nightly](https://github.com/platform-engineering-labs/formae-plugin-oci/actions/workflows/nightly.yml/badge.svg?branch=main)](https://github.com/platform-engineering-labs/formae-plugin-oci/actions/workflows/nightly.yml)

Formae plugin for managing Oracle Cloud Infrastructure resources.

## Supported Resources

| Resource Type | Description |
|---------------|-------------|
| `OCI::Identity::Compartment` | Compartments |
| `OCI::Core::Vcn` | Virtual Cloud Networks |
| `OCI::Core::Subnet` | Subnets |
| `OCI::Core::InternetGateway` | Internet gateways |
| `OCI::Core::NatGateway` | NAT gateways |
| `OCI::Core::ServiceGateway` | Service gateways |
| `OCI::Core::RouteTable` | Route tables |
| `OCI::Core::SecurityList` | Security lists |
| `OCI::Core::NetworkSecurityGroup` | Network security groups |
| `OCI::Core::NetworkSecurityGroupSecurityRule` | NSG security rules |
| `OCI::ContainerEngine::Cluster` | OKE clusters |
| `OCI::ContainerEngine::NodePool` | OKE node pools |
| `OCI::ContainerEngine::VirtualNodePool` | OKE virtual node pools |
| `OCI::ObjectStorage::Bucket` | Object storage buckets |

## Installation

```bash
make install
```

## Configuration

Configure an OCI target in your Forma file:

```pkl
import "@oci/oci.pkl"

new formae.Target {
    label = "my-oci-target"
    config = new oci.Config {
        region = "us-ashburn-1"
        profile = "DEFAULT"
    }
}
```

Authentication uses the OCI SDK's default config provider:
- Config file (`~/.oci/config`)
- Environment variables
- Instance principal (on OCI compute)

## Examples

See [examples/](examples/) for usage patterns:

- `lifeline/` - VCN networking infrastructure
- `oke/` - OKE Kubernetes cluster

**Note:** Update `vars.pkl` with your compartment ID and region before running.

## Development

```bash
make build          # Build plugin
make test           # Run tests
make install        # Install locally
make install-dev    # Install as v0.0.0 (for debug builds)
make gen-pkl        # Resolve PKL dependencies
```

## Conformance Tests

Run against real OCI resources:

```bash
make setup-credentials                           # Verify OCI login
make conformance-test   # Run full suite
```

## License

FSL-1.1-ALv2
