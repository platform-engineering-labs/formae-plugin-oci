# OCI Plugin for Formae

Formae plugin for managing Oracle Cloud Infrastructure resources.

## Quick Start

1. Clone and install
```
git clone git@github.com:platform-engineering-labs/formae-plugin-oci.git
cd formae-plugin-oci
make install
```

2. Configure OCI credentials (if not already done)
```
oci setup config
```

3. Verify credentials
```
./scripts/ci/setup-credentials.sh
```

4. Start the agent
```
formae agent start
```

5. Verify plugin is loaded
```
formae plugin list
```

6. Run an example
```
cd examples/lifeline
formae eval target.pkl
formae apply basic_infrastructure.pkl
```

7. Monitor and destroy
```
formae status command
formae destroy basic_infrastructure.pkl
```

If you get "resource is taken" errors, kill stale agents: `pkill -f formae`

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

## Configuration

Configure an OCI target in your forma file:

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
- Config file (~/.oci/config)
- Environment variables
- Instance principal (on OCI compute)

## Examples

See [examples/](examples/) for usage patterns:
- `lifeline/` - VCN networking infrastructure
- `oke/` - OKE Kubernetes cluster

## Development

```
make build          # Build plugin
make test           # Run tests
make install        # Install locally
make install-dev    # Install as v0.0.0 (for debug builds)
```

## Conformance Tests

Run against real OCI resources:

```
export OCI_COMPARTMENT_ID="ocid1.compartment.oc1..example"
export OCI_NAMESPACE="$(oci os ns get --query 'data' --raw-output)"
make conformance-test VERSION=0.77.16-internal
```

## License

FSL-1.1-ALv2
